#!/bin/bash

workspacePath=""
confFileName=mhserver.conf

echo "Hello! Let's setup your home server"

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
    mkdir $workspacePath
fi

#* --- Create .env for go app --- *#

if [[ -f ".env" ]]; then
    rm .env
fi

touch .env
echo -e "WORKSPACE_PATH=\"$workspacePath\"" >> .env

cd $workspacePath

#* --- Create configuration file --- *#

if [[ !(-f $confFileName) ]]; then
    touch $confFileName
else
    echo -e "\nServer configuration is already exist"
    read -p "Do you want setup mhserver again? (y/n): " user_input

    if [ $user_input != 'y' ]; then
        exit 0
    else
        rm $confFileName
        touch $confFileName
        echo # Skip the line
    fi
fi

confPath=$workspacePath$confFileName

#* ---  Setup server name --- *#

server_name=""
db_server_user_name=""
while [ -z $server_name ]; do
    read -p "Enter your server name: " server_name 
done

db_server_user_name=mhserver_$server_name
server_name=mhserver-$server_name
echo -e "ServerName = \"$server_name\"" >> $confPath

#* --- Set new password for server database --- *#

user_db_pass=""
echo -e "\nEnter a new password for server database"

while true; do
    read -p "Password: " -e -s user_db_pass
    read -p "Confirm password: " -e -s confirm_pass

    if [[ $user_db_pass == $confirm_pass ]]; then
        break
    else
        echo -e "Passwords do not match. Try again\n"
    fi
done

#* --- Create server user (mysql) and user database --- *#
echo # Skip the line

sql_driver=""
while [ -z $sql_driver ]; do
    read -p "What sql-driver you use? (mysql or mariadb): " sql_driver
done

echo "Generating database..."
sudo $sql_driver -u root -e "CREATE DATABASE IF NOT EXISTS $db_server_user_name;
CREATE USER IF NOT EXISTS 'mhserver'@'localhost' IDENTIFIED BY '$user_db_pass';
GRANT ALL PRIVILEGES ON $db_server_user_name.* TO 'mhserver'@'localhost';
"

if [ $? -ne 0 ]; then
    echo -e "\aError in generating server databases"
    exit 1
fi

#* ---- Create table: Users ---- *#

echo "Database has been generated"
echo -e "\nGenerating users table..."

echo "NOTE: Use your new password"
$sql_driver -u mhserver -p -e "USE $db_server_user_name;
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

#* --- IP:Port --- *#
echo # Skip the line

ip=""
while [ -z $ip ]; do
    read -p "Enter server IP: " ip
done
echo -e "IP = \"$ip\"" >> $confPath

port=""
while [ -z $port ]; do
    read -p "Enter server port: " port
done
echo -e "Port = \"$port\"" >> $confPath

echo -e "JWTSignature = \"$(openssl rand -base64 32)\"" >> $confPath
