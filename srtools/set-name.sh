#!/bin/bash

#! ****************************** !#
#!  Use -R to rename your server  !#
#! ****************************** !#

if [[ !(-f "setup.conf") ]]; then
    echo "setup conf file not found"
    exit 1
else
    . setup.conf
fi

conf_path=$workspacePath$confFileName 
if [[ !(-f $conf_path) ]]; then
    echo "server conf file not found"
    exit 1
else
    . $conf_path
fi

new_server_name=""
while [ -z $new_server_name ]; do
    read -p "Enter your server name: " new_server_name 
done

new_server_name="mhserver_$new_server_name"
if [[ $1 == "-R" ]]; then
    sed -e "s/ServerName=$ServerName/ServerName=$new_server_name/" $conf_path > /tmp/$confFileName
    mv /tmp/$confFileName $conf_path
else
    echo "ServerName=$new_server_name" >> $workspacePath$confFileName
fi