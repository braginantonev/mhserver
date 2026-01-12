#!/bin/bash

write_org_info() {
    echo -e "\nWrite organization info\n"
    sudo openssl req -new -key rootCA.key -out org.csr
}

create_root_cert() {
    echo -e "\nCreate root certificate\n"
    sudo openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.pem
}

create_TLS_cert() {
    sudo openssl x509 -req -in org.csr -CA rootCA.pem -CAkey rootCA.key -CAcreateserial -out org.crt -days 365 -sha256
}

# Функция, которая показывает, сколько дней осталось до конца действия сертификата
# Первый параметр - имя файла (сертификата)
get_expires_cert(){
    EXP_DATE=$( 
    echo "" | openssl x509 -noout -in $1 -enddate \
    | sed 's/notAfter=//' 
)
    echo $(( ($(date -d "$EXP_DATE" +%s) - $(date -d "now" +%s) ) / 86400 ))
}

CONFIG_PATH=/usr/share/mhserver/

if [[ !(-e $CONFIG_PATH) ]]; then
    echo "error: mhserver not configured. Use setup script first"
    exit 1
fi

echo "SSL Certificate generation..."

cd $CONFIG_PATH

if [[ !(-e ssl) ]]; then
    sudo mkdir ssl
fi

cd ssl

if [[ !(-e rootCA.key) ]]; then
    sudo openssl genrsa -out rootCA.key
fi

if [[ -e rootCA.pem ]]; then
    echo -n "Root certificate expiration check... "
    
    exp=$(get_expires_cert rootCA.pem)
    if [[ $exp -le 30 ]]; then
        echo -e "Certificate will be expired soon.\nGenerate new root certificate"

        sudo rm rootCA.pem
        create_root_cert
    else
        echo "Certificate will be expired in $exp days"
    fi
else
    create_root_cert
fi

if [[ -e org.csr ]]; then
    user_input=""
    while !([ "$user_input" == 'y' ] || [ "$user_input" == 'n' ]); do
        read -p "Would you rewrite your organization info? (y/n): " user_input
    done

    if [[ $user_input == "y" ]]; then
        sudo rm org.csr 
        write_org_info
    fi
else
    write_org_info
fi

if [[ -e org.crt ]]; then
    echo -n "TLS certificate expiration check... "
    
    exp=$(get_expires_cert org.crt)
    if [[ $exp -le 30 ]]; then
        echo -e "Certificate will be expired soon.\nGenerate new TLS certificate"

        sudo rm org.crt
        create_TLS_cert
    else
        echo "Certificate will be expired in $exp days"
    fi
else
    create_TLS_cert
fi
