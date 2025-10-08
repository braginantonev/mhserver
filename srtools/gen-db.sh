#!/bin/bash

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

user_db_pass=""
echo "Enter a new password for server database"

while true; do
    read -p "Password: " -e -s user_db_pass
    read -p "Confirm password: " -e -s confirm_pass

    if [[ $user_db_pass == $confirm_pass ]]; then
        break
    else
        echo -e "Passwords do not match. Try again\n"
    fi
done

sql_driver=""
while [ -z $sql_driver ]; do
    read -p "What sql-driver you use? (mysql or mariadb): " sql_driver
done

echo -e "Generating database...\n"
echo "NOTE: By default root pass is empty."
sudo $sql_driver -u root -p -e "CREATE DATABASE IF NOT EXISTS $ServerName;
CREATE USER IF NOT EXISTS 'mhserver'@'localhost' IDENTIFIED BY '$user_db_pass';
GRANT ALL PRIVILEGES ON $ServerName.* TO 'mhserver'@'localhost';
"

if [ $? -ne 0 ]; then
    echo -e "\aError in generating server databases"
    exit 1
fi

#* ---- Table - Users ---- *#

echo -e "\nDatabase has been generated"
echo "Generating users table..."

echo -e "\nNOTE: Use your new password"
$sql_driver -u mhserver -p -e "USE $ServerName;
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user VARCHAR(30) NOT NULL,
    password VARCHAR(256) NOT NULL
);"

if [ $? -ne 0 ]; then
    echo -e "\aError in creating database tables"
    exit 1
fi