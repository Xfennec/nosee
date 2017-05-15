#!/bin/bash

if [ -z "$2" ]; then
    (>&2 echo "ERROR: give certificate path, short name and 'days to expire date'")
    (>&2 echo "ERROR: Usage: $0 /etc/pki/tls/certs/myweb.crt MYWEB 15")
    exit 1
fi

cert_path=$1
short_name=$2
days_to_expire=$3

timestamp=$(echo $(($days_to_expire*24*60*60)))

openssl x509 -checkend $timestamp -noout -in "$1"
res=$?

echo "CERT_WILL_EXPIRE_${short_name}:" $res
