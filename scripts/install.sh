#!/bin/bash

# CONSTANTS
EXECUTABLE_NAME=mhserver

## PATHS
INSTALL_PATH=/opt/mhserver
LOGS_PATH=$INSTALL_PATH/logs
USER_SPACE_PATH=$INSTALL_PATH/uspace

PATHS=("$INSTALL_PATH" "$LOGS_PATH" "$USER_SPACE_PATH")

## DIRS
SCRIPTS_DIR=scripts

DB_PASSWORD=""

# $1 - prompt; Use $(yn_input) to get value
yn_input() {
    local user_input=""
    while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
        read -e -p "$1 (y/n): " user_input
    done
    echo $user_input
}

create_database() {
    local local_address=$(ip route get 8.8.8.8 | grep -oP 'src \K\S+')
    local db_serve=$(echo $local_address | cut -d. -f1-3).%

    echo "Database will access servers in net $db_serve (also localhost)"

    echo "Create mhserver db user..."
    sudo mariadb -u root <<-EOF 
CREATE USER IF NOT EXISTS 'mhserver'@'$db_serve' identified by '$DB_PASSWORD';
CREATE USER IF NOT EXISTS 'mhserver'@'localhost' identified by '$DB_PASSWORD';
EOF
    if [ $? -ne 0 ]; then
        echo -e "\aFailed create mariadb user"
        exit 1
    fi

    echo "Create server databases..."
    sudo mariadb -u root <<-EOF
CREATE DATABASE IF NOT EXISTS mhs_main;
GRANT ALL PRIVILEGES ON mhs_main.* TO 'mhserver'@'$db_serve';
GRANT ALL PRIVILEGES ON mhs_main.* TO 'mhserver'@'localhost';
EOF
    if [ $? -ne 0 ]; then
        echo -e "\aError in generating server mariadb databases"
        exit 1
    fi
}

create_database_tables() {
    echo -e "Create server tables..."
    mariadb -u mhserver -D mhs_main --password=$DB_PASSWORD <<-EOF
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user VARCHAR(30) NOT NULL,
    password VARCHAR(256) NOT NULL
);

CREATE TABLE IF NOT EXISTS register_secret_keys (
    id INT AUTO_INCREMENT PRIMARY KEY,
    secret_key VARCHAR(64) NOT NULL
);
EOF
    if [ $? -ne 0 ]; then
        echo -e "\aError in creating database tables"
        exit 1
    fi
}

clear_install() {
    sudo useradd mhserver -G network,storage --system --shell /usr/sbin/nologin 

    # Make application tree
    for p in "${PATHS[@]}"; do
        sudo mkdir $p
        sudo chown mhserver:mhserver $p
    done

    DB_PASSWORD=$(openssl rand -base64 32)

    touch .env
    tee .env >/dev/null <<-EOF
JWT_SIGNATURE = "$(openssl rand -base64 32)"
DATABASE_PASSWORD = "$DB_PASSWORD"
SERVER_TOKEN = "$(openssl rand -base64 8)"
WORKSPACE_PATH = "$INSTALL_PATH"
EOF

    sudo cp -a . $INSTALL_PATH/
    sudo chown -R mhserver:mhserver $INSTALL_PATH
    sudo chmod 666 $INSTALL_PATH/server.conf

    sudo install -o mhserver -g mhserver -m 0660 .env $INSTALL_PATH
    rm .env 

    if [[ $(yn_input "Do you want to create server database here?") == "y" ]]; then
        create_database
        create_database_tables
    fi

    ./create_ssl.sh

    sudo touch /etc/systemd/system/mhserver.service
    sudo tee /etc/systemd/system/mhserver.service >/dev/null <<-EOF
[Unit]
Description=My home server
After=network-online.target
Requires=mariadb.service

[Service]
Type=simple

User=mhserver
Group=mhserver

ExecStart=/opt/mhserver/mhserver
ExecReload=/opt/mhserver/mhserver

Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload

    echo -e "\nMHServer installed. Now you can configure and use them"
}

simple_update() {
    local row=$(ls)
    local current_dir=$(pwd)
    mapfile -t files <<< "$row"

    # install new files
    for f in ${files[@]}; do
        if [[ !(-e $f) ]]; then
            sudo install -o mhserver -g mhserver $current_dir/$f $INSTALL_PATH
        fi
    done

    # update database tables
    #if [ -d "/var/lib/mysql" ] || systemctl list-unit-files | grep -q "mariadb.service"; then
    #    create_database_tables
    #fi

    # update executable
    sudo install -o mhserver -g mhserver mhserver $INSTALL_PATH
}

# install modes:
#   0 - clear install
#   1 - simple update
MODE=0

if [[ -e $INSTALL_PATH ]]; then
    echo -n "MHServer already installed. "
    if [[ $(yn_input "Do you want to simple update?") == "y" ]]; then 
        MODE=1
    else
        exit 0
    fi
fi

case $MODE in
    0)
    clear_install
    ;;
    1)
    simple_update
    ;;
esac
