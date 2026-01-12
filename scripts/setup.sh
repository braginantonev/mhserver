#!/bin/bash

CONFIG_PATH=/usr/share/mhserver/
CONFIG_NAME=mhserver.conf

TEMP_PATH=/tmp/mhserver_setup
SUB_SERVERS=(main files music images llm)

MAX_CHUNK_SIZE=52428800
MIN_CHUNK_SIZE=4096

if [[ -e mhserver ]]; then
    if [[ !(-e /opt/mhserver) ]]; then
        sudo mkdir /opt/mhserver
    fi

    echo -e "\nReplace executable file to /opt/mhserver ..."
    sudo cp mhserver /opt/mhserver
fi

echo # Skip line

if [[ !(-e $CONFIG_PATH) ]]; then
    sudo mkdir $CONFIG_PATH
fi

cd ..

if [[ -e mhserver.service ]]; then
    user_input=""
    while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
        read -p "Create mhserver systemd service? (y/n): " user_input
    done

    if [[ $user_input == 'y' ]]; then
        sudo cp mhserver.service $CONFIG_PATH
        sudo cp ${CONFIG_PATH}mhserver.service /etc/systemd/system

        sudo systemctl daemon-reload
    fi
fi

#* --- Copy sql commands to /tmp/ --- *#

if [[ !(-e $TEMP_PATH) ]]; then
    mkdir $TEMP_PATH
fi

cp -r sql $TEMP_PATH

cd $CONFIG_PATH

#* --- Create configuration file --- *#

if [[ !(-f $CONFIG_NAME) ]]; then
    sudo touch $CONFIG_NAME
else
    echo "Server configuration is already exist"
    read -p "Do you want setup mhserver again? (y/n): " user_input

    if [ $user_input != 'y' ]; then
        exit 0
    else
        sudo rm $CONFIG_NAME
        sudo touch $CONFIG_NAME
        echo # Skip the line
    fi
fi

sudo chmod 600 $CONFIG_NAME

echo "Hello! Let's setup your home server"

#* --- Create server workspace folder --- *#
workspacePath=""

echo # Skip the line
read -p "Do you wan't set uniq server workspace path? (y/n): " workspacePath
if [[ $workspacePath != 'y' ]]; then
    workspacePath=~/.mhserver/
else
    workspacePath=""
    while [ -z $workspacePath ]; do
        read -p "Enter your new path (use full path): " workspacePath
    done
fi

echo -e "workspace_path = \"$workspacePath\"" | sudo tee -a $CONFIG_NAME > /dev/null

echo "Server workspace path is set to $workspacePath"

if [[ !(-e $workspacePath) ]]; then
    sudo mkdir $workspacePath
fi

#* Generate jwt secrete signature
echo "Generation JWT signature..."
echo -e "jwt_signature = \"$(openssl rand -base64 32)\"" | sudo tee -a $CONFIG_NAME > /dev/null

#* --- Set new password for server database --- *#
db_pass=""
echo -e "\nEnter a new password for server database"

while true; do
    read -p "Password: " -e -s db_pass
    read -p "Confirm password: " -e -s confirm_pass

    if [[ $db_pass == $confirm_pass ]]; then
        break
    else
        echo -e "Passwords do not match. Try again\n"
    fi
done

echo "db_pass = \"$db_pass\"" | sudo tee -a $CONFIG_NAME > /dev/null

#* --- Create server user (mysql) and user database --- *#
echo # Skip the line

echo "Create mhserver db user..."
sudo mariadb -u root -e "create user if not exists 'mhserver'@'localhost' identified by '$db_pass';"
if [ $? -ne 0 ]; then
    echo -e "\aFailed create $sql_driver user"
    exit 1
fi

echo "Create mhserver_tests db user..."
sudo mariadb -u root -e "create user if not exists 'mhserver_tests'@'localhost';"
if [ $? -ne 0 ]; then
    echo -e "\aFailed create $sql_driver user"
    exit 1
fi

echo "Create server databases..."
sudo mariadb -u root < $TEMP_PATH/sql/create-db.sql
if [ $? -ne 0 ]; then
    echo -e "\aError in generating server $sql_driver databases"
    exit 1
fi

#* ---- Create users table ---- *#

echo -e "Create users table..."
mariadb -u mhserver --password=$db_pass -D mhs_main < $TEMP_PATH/sql/tables.sql
if [ $? -ne 0 ]; then
    echo -e "\aError in creating database tables"
    exit 1
fi

echo -e "Create tests users table..."
mariadb -u mhserver_tests -D mhs_main_test < $TEMP_PATH/sql/tables.sql
if [ $? -ne 0 ]; then
    echo -e "\aError in creating database tables"
    exit 1
fi

#* ---- Memory setup --- *#
echo # Skip the line

available_ram=""
while [ -z $available_ram ] || [ $available_ram -gt 100 ] || [ $available_ram -le 0 ]; do
    read -p "Available server RAM percentage: " available_ram
done

echo -e "\n[memory]" | sudo tee -a $CONFIG_NAME > /dev/null
total_memory=$(($(free | grep "Mem" | awk '{print $2}') * 1024 * available_ram / 100))
echo "available_ram = $total_memory # bytes" | sudo tee -a $CONFIG_NAME > /dev/null
echo "max_chunk_size = $MAX_CHUNK_SIZE # bytes" | sudo tee -a $CONFIG_NAME > /dev/null
echo "min_chunk_size = $MIN_CHUNK_SIZE # bytes" | sudo tee -a $CONFIG_NAME > /dev/null

for server in ${SUB_SERVERS[*]}
do
    echo
    echo -e "\n[subservers.$server]" | sudo tee -a $CONFIG_NAME > /dev/null

    user_input=""
    while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
        read -p "Do you want use $server subserver? (y/n): " user_input
    done

    if [[ $user_input == 'n' ]]; then
        echo "enabled = false" | sudo tee -a $CONFIG_NAME > /dev/null
        continue
    fi

    echo "enabled = true" | sudo tee -a $CONFIG_NAME > /dev/null

    user_input=""
    read -p "Enter subserver IP (localhost by default): " user_input

    if [[ -z $user_input ]]; then
        echo -e "ip = \"localhost\"" | sudo tee -a $CONFIG_NAME > /dev/null
    else
        echo -e "ip = \"$user_input\"" | sudo tee -a $CONFIG_NAME > /dev/null
    fi

    user_input=""
    while [ -z $user_input ]; do
        read -p "Enter subserver port (use a unique port): " user_input
    done
    echo -e "port = \"$user_input\"" | sudo tee -a $CONFIG_NAME > /dev/null

done

user_input=""
while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
    read -p "Start mhserver right now? (y/n): " user_input
done

if [[ $user_input == 'y' ]]; then
    sudo systemctl enable mhserver
    sudo systemctl start mhserver
    echo -e "MHServer will been configured and started successfully"
    exit 0
fi

echo -e "MHServer will be configured successfully"
