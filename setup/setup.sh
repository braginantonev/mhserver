#!/bin/bash

#* Variables
workspacePath=~/.mhserver/
confFileName=mhserver.conf

if [[ !(-e $workspacePath) ]]; then
    mkdir $workspacePath
fi

cd $workspacePath

if [[ !(-f $confFileName) ]]; then
    touch $confFileName
else
    echo "Server configuration is already exist"

    read -p "Do you want setup mhserver again? (y/n): " user_input
    if [ $user_input != 'y' ]; then
        exit 0
    else
        rm $confFileName
        touch $confFileName
    fi
fi

clear

echo "Hello! Let's setup your home server"

#* --- Server name (mysql user name) ---
server_name=""
while [ -z $server_name ]; do
    read -p "Enter your server name: " server_name 
done

server_name="mhserver_$server_name"
echo "ServerName = $server_name" >> $confFileName

clear

#* --- DB user password ---
user_db_pass=""
echo "Enter password for server databases"

while true; do
    read -p "Password: " -e -s user_db_pass
    read -p "Confirm password: " -e -s confirm_pass

    if [[ $user_db_pass == $confirm_pass ]]; then
        break
    else
        echo -e "Passwords do not match. Try again\n"
    fi
done

clear

#* --- Generate DB server user ---
sql_driver=""
while [ -z $sql_driver ]; do
    read -p "What sql-driver you use? (mysql or mariadb): " sql_driver
done

echo -e "Generating database...\n"
echo "NOTE: By default root pass is empty."
sudo $sql_driver -u root -p -e "CREATE DATABASE IF NOT EXISTS $server_name;
CREATE USER IF NOT EXISTS 'mhserver'@'localhost' IDENTIFIED BY '$user_db_pass';
GRANT ALL PRIVILEGES ON $server_name.* TO 'mhserver'@'localhost';
"

if [ $? -ne 0 ]; then
    echo "Error in generating server databases."
    exit 1
fi
