#!/bin/bash

if [[ "$1" == "" ]]; then
    echo -e "\aYou must enter the target dir"
    exit 1
fi

TARGET=$1
SERVER_USPACE_DIR=/opt/mhserver/uspace

if [[ !(-e $TARGET) ]]; then
    echo -e "\aTarget dir not found"
    exit 1
fi

TARGET_USPACE=$TARGET/.mhserver

mkdir $TARGET_USPACE
sudo chown mhserver:mhserver $TARGET_USPACE
sudo chmod 770 $TARGET_USPACE

AVAILABLE_SPACES=($SERVER_USPACE_DIR/*)
if [[ "${AVAILABLE_SPACES[0]}" == "$SERVER_USPACE_DIR/*" ]]; then
    sudo ln -s $TARGET_USPACE $SERVER_USPACE_DIR/0
else
    sudo ln -s $TARGET_USPACE $SERVER_USPACE_DIR/${#AVAILABLE_SPACES[@]}
fi
