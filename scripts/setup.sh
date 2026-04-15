#!/bin/bash

#* --------------- Variables --------------- *#

CONFIG_PATH=/usr/share/mhserver
CONFIG_NAME=mhserver.conf

EXECUTABLE_PATH=/opt/mhserver

SUB_SERVERS=(files music images llm)
BASE_PORT=30543

MAX_CHUNK_SIZE=52428800 # bytes = 50 mb
MIN_CHUNK_SIZE=4096 # bytes = 4 kb

is_resetup_session=false

#* --------------- Functions --------------- *#

# $1 - prompt; Use $(yn_input) to get value
yn_input() {
    local user_input=""
    while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
        read -e -p "$1 (y/n): " user_input
    done
    echo $user_input
}

# $1 - want save; $2 - where to save
write_to_file() {
    echo -e $1 | sudo tee -a $2 > /dev/null

    if [ $? -ne 0 ]; then
        echo -e "\aFailed write to file"
        exit 1
    fi
}

#* -------- Root project tree check -------- *#

if [[ !(-e mhserver) ]]; then
    cd ..

    if [[ !(-e mhserver) ]]; then
        echo "Run mhserver setup script only with builded project!"
        exit 1
    fi
fi


#* --------- Create executable dir --------- *#

if [[ !(-e $EXECUTABLE_PATH) ]]; then
    sudo mkdir $EXECUTABLE_PATH
else
    if [[ $(yn_input "MHServer already installed. Reinstall (Update)?") == 'y' ]]; then
        sudo rm -rf $EXECUTABLE_PATH/*
    fi
fi

echo "Coping executable files to $EXECUTABLE_PATH ..."
sudo cp -r * $EXECUTABLE_PATH


#* --------- Create config path ------------ *#

if [[ !(-e $CONFIG_PATH) ]]; then
    sudo mkdir $CONFIG_PATH
fi

#* ------- Create configuration file ------- *#

echo # Skip line

cd $CONFIG_PATH

if [[ !(-e $CONFIG_NAME) ]]; then
    sudo touch $CONFIG_NAME
else
    echo "Server configuration is already exist"

    if [[ $(yn_input "Do you want setup mhserver again?") == 'n' ]]; then
        exit 0
    fi

    echo -e "\a\n\033[0;33mWARNING! If you continue, old configuration will be deleted!"
    echo -e "\033[0;33mIf you really want to continue, I strongly recommend that you keep your old database password and jwt signature.\n"
    echo -e -n "\033[0;37m" 

    if [[ $(yn_input "Do you really want continue?") == 'n' ]]; then
        exit 0
    fi 

    is_resetup_session=true

    sudo rm $CONFIG_NAME
    sudo touch $CONFIG_NAME
fi

sudo chmod 600 $CONFIG_NAME


#* ----------------- Setup ----------------- *#

echo -e "\nHello! Let's setup your home server"


#* ---- Create server workspace folder ----- *#

workspacePath=""

echo # Skip the line

if [[ $(yn_input "Do you wan't set uniq server workspace path?") == 'y' ]]; then
    while [ -z $workspacePath ]; do
        read -e -p "Enter your new path (use full path): " workspacePath
    done
else
    workspacePath=~/.mhserver/
fi

echo "Server workspace path is set to $workspacePath"
write_to_file "workspace_path = \"$workspacePath\"" $CONFIG_NAME

if [[ !(-e $workspacePath) ]]; then
    sudo mkdir $workspacePath
fi


#* ---- Generate jwt secrete signature ----- *#

echo # Skip the line

jwt_signature=""

if $is_resetup_session; then
    if [[ $(yn_input "Do you want use old jwt signature?") == 'y' ]]; then
        while [ -z $jwt_signature ]; do
            read -e -p "Enter your old jwt signature: " jwt_signature
        done
    fi
fi

if [[ -z $jwt_signature ]]; then
    echo "Generation JWT signature..."

    jwt_signature=$(openssl rand -base64 32)
fi

write_to_file "jwt_signature = \"$jwt_signature\"" $CONFIG_NAME


#* --- Set new password for server database --- *#

echo # Skip the line

if $is_resetup_session; then
    if [[ $(yn_input "Do you want use old db password?") == 'y' ]]; then
        while true; do
            db_password=""
            while [ -z $db_password ]; do
                read -e -s -p "Enter your old db password: " db_password
            done

            confirmed_db_password=""
            while [ -z $confirmed_db_password ]; do
                read -e -s -p "Confirm your old db password: " confirmed_db_password
            done

            if [[ $db_password == $confirmed_db_password ]]; then
                break
            else
                echo -e "Passwords not ident\n"
            fi
        done
    fi
fi

if [[ -z $db_password ]]; then
    db_password=$(openssl rand -base64 32)
fi

write_to_file "db_pass = \"$db_password\"" $CONFIG_NAME


#* --- Create server user (MariaDB) and user database --- *#

echo # Skip the line

echo "Create mhserver db user..."
sudo mariadb -u root -e "create user if not exists 'mhserver'@'localhost' identified by '$db_password';"
if [ $? -ne 0 ]; then
    echo -e "\aFailed create mariadb user"
    exit 1
fi

echo "Create mhserver_tests db user..."
sudo mariadb -u root -e "create user if not exists 'mhserver_tests'@'localhost';"
if [ $? -ne 0 ]; then
    echo -e "\aFailed create mariadb user"
    exit 1
fi

echo "Create server databases..."
sudo mariadb -u root < $EXECUTABLE_PATH/sql/create-db.sql
if [ $? -ne 0 ]; then
    echo -e "\aError in generating server mariadb databases"
    exit 1
fi


#* ---------- Create server tables ---------- *#

echo -e "Create server tables..."
mariadb -u mhserver --password=$db_password -D mhs_main < $EXECUTABLE_PATH/sql/tables.sql
if [ $? -ne 0 ]; then
    echo -e "\aError in creating database tables"
    exit 1
fi

echo -e "Create test server tables..."
mariadb -u mhserver_tests -D mhs_main_test < $EXECUTABLE_PATH/sql/tables.sql
if [ $? -ne 0 ]; then
    echo -e "\aError in creating database tables"
    exit 1
fi

#* -------------- Memory setup --------------- *#

echo # Skip the line

available_ram=""
while [ -z $available_ram ] || [ $available_ram -gt 100 ] || [ $available_ram -le 0 ]; do
    read -e -p "Available server RAM percentage: " available_ram
done

write_to_file "\n[memory]" $CONFIG_NAME

total_memory=$(($(free | grep "Mem" | awk '{print $2}') * 1024 * available_ram / 100))
write_to_file "available_ram = $total_memory # bytes" $CONFIG_NAME
write_to_file "max_chunk_size = $MAX_CHUNK_SIZE # bytes" $CONFIG_NAME
write_to_file "min_chunk_size = $MIN_CHUNK_SIZE # bytes" $CONFIG_NAME


#* ------------ Subservers setup ----------- *#

echo -e "\nSetup main server..."

write_to_file "\n[subservers.main]" $CONFIG_NAME
write_to_file "enabled = true" $CONFIG_NAME

main_address=""
while [ -z $main_address ]; do
    read -e -p "Enter address of your server (e.g. my.server.com, 145.123.123.12, localhost): " main_address
done
write_to_file "address = \"$main_address\"" $CONFIG_NAME

main_port=$BASE_PORT
if [[ $(yn_input "Do you want use own port ($main_port using by default)?") == 'y' ]]; then
    while [ -z $main_port ] || [ $main_port -eq $BASE_PORT ]; do
        read -e -p "Enter your server port: " main_port
    done
fi
write_to_file "port = $main_port" $CONFIG_NAME

echo -e "\nSetup services..."

grpc_port=$((main_port+1))
if [[ $(yn_input "Do you want use own port for services ($grpc_port using by default)?") == 'y' ]]; then
    while [ -z $grpc_port ] || [ $grpc_port -eq $main_port ]; do
        read -e -p "Enter your port for services: " main_port
    done
fi

echo # SKip the line

write_to_file "\n# Subservers have duplicate 'address' and 'port' fields for backward compatibility," $CONFIG_NAME
write_to_file "# in case I want to revert to microservice architecture in the future." $CONFIG_NAME

for server in ${SUB_SERVERS[*]}; do
    write_to_file "\n[subservers.$server]" $CONFIG_NAME

    if [[ $(yn_input "Do you want use $server subserver?") == 'n' ]]; then
        write_to_file "enabled = false" $CONFIG_NAME
    else
        write_to_file "enabled = true" $CONFIG_NAME
    fi

    write_to_file "address = \"localhost\" # 'localhost' for monolithic arch" $CONFIG_NAME
    write_to_file "port = $grpc_port" $CONFIG_NAME
done

#* ------- HTTPS/TLS configuration --------- *#

echo # SKip the line

sh /opt/mhserver/scripts/create-ssl-cert.sh


#* ------- Generate register secrets ------- *#

echo # SKip the line

sh /opt/mhserver/scripts/generate_reg_keys.sh --db_pass=$db_password


#* -------- Enable server service ---------- *#

echo # Skip the line

if [[ $(yn_input "Create mhserver systemd service?") == 'y' ]]; then
    sudo cp $EXECUTABLE_PATH/mhserver.service /etc/systemd/system
    sudo systemctl daemon-reload

    if [[ $(yn_input "Start mhserver service right now?") == 'y' ]]; then
        sudo systemctl enable --now mhserver
    fi
fi

echo -e "MHServer will be configured successfully"
