#!/bin/bash

EXECUTABLE_NAME=mhserver
INSTALL_PATH=/opt/mhserver
USPACE_DIR=uspace

# $1 - prompt; Use $(yn_input) to get value
yn_input() {
    local user_input=""
    while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
        read -e -p "$1 (y/n): " user_input
    done
    echo $user_input
}

if [[ !(-e $INSTALL_PATH) ]]; then 
    echo "MHServer not installed"
    exit
fi


echo "This action will be delete all mhserver files (even user files)!"
echo "Make a backup of user files from all disks which have a symlink in "uspace" dir, if you want to save their!"

echo
if [[ $(yn_input "Are you sure you want to uninstall MHServer?") == "n" ]]; then
    exit
fi

sudo systemctl disable --now mhserver

USPACE_DISKS=($INSTALL_PATH/$USPACE_DIR/*)
for sml in "${USPACE_DISKS[@]}"; do
    sudo rm -rf $(readlink -f $sml) 
done

sudo rm -rf $INSTALL_PATH

sudo mariadb -u root <<EOF
DROP DATABASE IF EXISTS mhs_main;
DROP USER IF EXISTS mhserver;
EOF

sudo rm /etc/systemd/system/mhserver.service
sudo systemctl daemon-reload
sudo systemctl reset-failed

sudo userdel mhserver

echo "MHServer and user files was deleted."