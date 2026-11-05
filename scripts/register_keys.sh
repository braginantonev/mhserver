#!/bin/bash

db_pass=""

# Available modes:
# 0 - show available keys
# 1 - generate new keys
mode=0

for arg in "$@"; do
    case $arg in
        --db_pass=*)
        db_pass="${arg#*=}"
        ;;
        -G)
        mode=1
        ;;
    esac
done

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

mariadb -u mhserver --password=$db_pass <<-SQL
exit
SQL

if [ $? -ne 0 ]; then
    echo "wrong database password"
    exit 1
fi

case $mode in
    0)
    show_keys
    ;;
    1)
    generate_keys
    show_keys
    ;;
esac

echo -e "\nUse this keys to register on server"
