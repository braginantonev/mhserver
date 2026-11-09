#!/bin/bash

db_pass=""

# Available modes:
# 0 - show help
# 1 - show available keys
# 2 - generate new keys
MODE=0

for arg in "$@"; do
    case $arg in
        "--db_pass="* )
        db_pass="${arg#*=}"
        ;;
        "-S" )
        MODE=1
        ;;
        "--show" )
        MODE=1
        ;;
        "-G" )
        MODE=2
        ;;
        "--generate" )
        MODE=2
        ;;
    esac
done

show_help() {
    echo \
    "register_keys: register_keys.sh [option] [--db_pass=* database_password]
Script to generate or check register keys.

Options:
    -G --generate generate, save and show new register secret keys
    -S --show show unused register keys
"
}

show_keys() {
    row=$(mariadb -u mhserver -D mhs_main -N --password=$db_pass <<-SQL
SELECT secret_key FROM register_secret_keys;
SQL
)

    mapfile -t keys <<< "$row"
    echo -e "Register secret keys:\n"

    for i in ${!keys[@]}; do
        echo "$i. ${keys[i]}"
    done
}

generate_keys() {
    echo -e "Generate register secrets...\n"

    for (( i = 0; i < 5; i ++))
    do
        key=$(openssl rand -hex 32)
        mariadb -u mhserver --password=$db_pass -D mhs_main <<-SQL
INSERT INTO register_secret_keys (secret_key) VALUES ('$key');
SQL
    done
}

if [ $MODE == 0 ]; then
    show_help
    exit 0
fi

mariadb -u mhserver --password=$db_pass <<-SQL
exit
SQL

if [ $? -ne 0 ]; then
    echo "wrong database password"
    exit 1
fi

case $MODE in
    1 )
    show_keys
    ;;
    2 )
    generate_keys
    show_keys
    ;;
esac

echo -e "\nUse this keys to register on server"
