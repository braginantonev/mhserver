#!/bin/bash

#! *********************************** !#
#! Use -R to recreate server conf file !#
#! *********************************** !#

if [[ !(-f "setup.conf") ]]; then
    echo "setup conf file not found"
    exit 1
else
    . setup.conf
fi

if [[ !(-e $workspacePath) ]]; then
    mkdir $workspacePath
fi

cd $workspacePath

if [[ !(-f $confFileName) ]]; then
    touch $confFileName
else
    if [[ $1 == "-R" ]]; then
        rm $confFileName
        touch $confFileName
    fi
fi