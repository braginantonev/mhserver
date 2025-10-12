#!/bin/bash

CONFIG_PATH=/usr/share/mhserver/
ENV_PATH=$(pwd)/.env
CONFIG_NAME=mhserver.conf

SUB_SERVERS=(main files music images llm)

if [[ !(-e $CONFIG_PATH) ]]; then
    sudo mkdir $CONFIG_PATH
fi

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

echo "Hello! Let's setup your home server"
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
echo "Server workspace path is set to $workspacePath"

if [[ !(-e $workspacePath) ]]; then
    sudo mkdir $workspacePath
fi

#* --- Create .env for go app --- *#

if [[ -f $ENV_PATH ]]; then
    rm $ENV_PATH
fi

touch $ENV_PATH
echo -e "CONFIG_PATH=\"$CONFIG_PATH$CONFIG_NAME\"" >> $ENV_PATH
echo -e "WORKSPACE_PATH=\"$workspacePath\"" >> $ENV_PATH

#* ---  Setup server name --- *#
echo # Skip the line

server_name=""
while [ -z $server_name ]; do
    read -p "Enter your server name: " server_name
done

server_name=mhserver_$server_name
echo -e "server_name = \"$server_name\"" | sudo tee -a $CONFIG_NAME > /dev/null

if [ $? -ne 0 ]; then
    echo -e "\aInternal error. Please tell me about this in Github Issues."
    exit 1
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

sql_driver=""
while [ -z $sql_driver ]; do
    read -p "What sql-driver you use? (mysql or mariadb): " sql_driver
done

echo "Generating database..."
sudo $sql_driver -u root -e "CREATE DATABASE IF NOT EXISTS $server_name;
CREATE USER IF NOT EXISTS 'mhserver'@'localhost' IDENTIFIED BY '$db_pass';
GRANT ALL PRIVILEGES ON $server_name.* TO 'mhserver'@'localhost';
"

if [ $? -ne 0 ]; then
    echo -e "\aError in generating server databases"
    exit 1
fi

#* ---- Create table: Users ---- *#

echo "Database has been generated"
echo -e "\nGenerating users table..."

echo "NOTE: Use your new password"
$sql_driver -u mhserver -p -e "USE $server_name;
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user VARCHAR(30) NOT NULL,
    password VARCHAR(256) NOT NULL
);"
#Todo: Добавить создание остальных таблиц

if [ $? -ne 0 ]; then
    echo -e "\aError in creating database tables"
    exit 1
fi

echo -e "\nSetup subservers..."

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
    echo -e "hostname = \"mhserver_$server\"" | sudo tee -a $CONFIG_NAME > /dev/null

    user_input=""
    while [ -z $user_input ]; do
        read -p "Enter subserver IP (use 'localhost' for current pc): " user_input
    done
    echo -e "ip = \"$user_input\"" | sudo tee -a $CONFIG_NAME > /dev/null

    user_input=""
    while [ -z $user_input ]; do
        read -p "Enter subserver port: " user_input
    done
    echo -e "port = \"$user_input\"" | sudo tee -a $CONFIG_NAME > /dev/null

done

sudo chmod 600 $CONFIG_NAME
echo "MHServer has been configured"