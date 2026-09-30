#!/bin/bash

for arg in "$@"; do
  if [[ $arg == --db_pass=* ]]; then
    db_pass="${arg#*=}"
  fi
done

mariadb -u mhserver --password=$db_pass <<-SQL
exit
SQL

if [ $? -ne 0 ]; then
    echo "wrong database password"
    exit 1
fi

echo -e "Generate register secrets...\n"

echo \
"#######################################################################
#                        Register secret keys                         #
#######################################################################
#                                                                     #"

for (( i = 0; i < 5; i ++))
do
    key=$(openssl rand -hex 32)
    mariadb -u mhserver --password=$db_pass -D mhs_main <<-SQL
    INSERT INTO register_secret_keys (secret_key) VALUES ('$key');
SQL

    if [ $? -eq 0 ]; then
        echo "# $i. $key #"
    fi
done

echo \
"#                                                                     #
#######################################################################"

echo -e "\nUse this secrets to register on server"