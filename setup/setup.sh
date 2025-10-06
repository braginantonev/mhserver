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

echo -e "\nHello! Let's setup your home server\n"

#* --- Server name (mysql user name) ---
server_name=""
while [ -z $server_name ]; do
    read -p "Enter your server name: " server_name 
done

echo "ServerName = mhserver-$server_name" >> $confFileName

#* --- DB user password ---
user_db_pass=""
echo -e "\nEnter password for server databases"

while true; do
    read -p "Password: " -e -s user_db_pass
    read -p "Confirm password: " -e -s confirm_pass

    if [[ $user_db_pass == $confirm_pass ]]; then
        break
    else
        echo -e "Passwords do not match. Try again\n"
    fi
done

    