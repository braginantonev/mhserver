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

# $1 - prompt; Use $(yn_input) to get value
yn_input() {
    local user_input=""
    while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
        read -e -p "$1 (y/n): " user_input
    done
    echo $user_input
}

clear_install() {
    sudo useradd mhserver -G network,storage --system --shell /usr/sbin/nologin 

    # Make application tree
    for p in "${PATHS[@]}"; do
        sudo mkdir $p
        sudo chown mhserver:mhserver $p
    done

    local db_pass=$(openssl rand -base64 32)

    touch .env
    tee .env >/dev/null <<-EOF
JWT_SIGNATURE = "$(openssl rand -base64 32)"
DATABASE_PASSWORD = "$db_pass"
SERVER_TOKEN = "$(openssl rand -base64 8)"
WORKSPACE_PATH = "$INSTALL_PATH"
EOF

    sudo cp -a . $INSTALL_PATH/
    sudo chown -R mhserver:mhserver $INSTALL_PATH
    sudo chmod 666 $INSTALL_PATH/server.conf

    sudo install -o mhserver -g mhserver -m 0660 .env $INSTALL_PATH
    rm .env 

    echo "Create mhserver db user..."
    sudo mariadb -u root -e "create user if not exists 'mhserver'@'localhost' identified by '$db_pass';"
    if [ $? -ne 0 ]; then
        echo -e "\aFailed create mariadb user"
        exit 1
    fi

    echo "Create server databases..."
    sudo mariadb -u root <<-EOF
CREATE DATABASE IF NOT EXISTS mhs_main;
GRANT ALL PRIVILEGES ON mhs_main.* TO 'mhserver'@'localhost';
EOF
    if [ $? -ne 0 ]; then
        echo -e "\aError in generating server mariadb databases"
        exit 1
    fi

    echo -e "Create server tables..."
    mariadb -u mhserver -D mhs_main --password=$db_pass <<-EOF
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
