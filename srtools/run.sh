#!/bin/bash

#! ****************************** !#
#! Use -R to re setup server conf !#
#! ****************************** !#

if [[ !(-f "setup.conf") ]]; then
    echo "setup conf file not found"
    exit 1
else
    . setup.conf
fi

./set-cfg.sh

echo "Hello! Let's setup your home server"

./set-name.sh

./gen-db.sh
