#!/bin/bash

if [ -z "$3" ]; then
    (>&2 echo "ERROR: give URL, short name and an expected string")
    (>&2 echo "Usage example: $0 'http://www.perdu.com/' PERDU 'Pas de panique'")
    exit 1
fi

url=$1
short=$2
expected=$3

status=0

page=$(curl --silent -f "$url")
if [ $? -eq 0 ]; then
    n=$(echo "$page" | grep "$expected" | wc -l)
    if [ $n -gt 0 ]; then
	status=1
    fi
fi

echo "CURL_OK_$short:" $status
