#!/bin/bash

if [ -z "$2" ]; then
    (>&2 echo "ERROR: give certificate path and 'days to expire'")
    (>&2 echo "ERROR: Usage: $0 /etc/pki/tls/certs/myweb.crt 15")
    exit 1
fi

cert_path=$1
short_name=$2
days_to_expire=$3

timestamp=$(echo $(($days_to_expire*24*60*60)))

openssl x509 -checkend $timestamp -noout -in "$1"
res=$?

echo "WILL_EXPIRE:" $res
