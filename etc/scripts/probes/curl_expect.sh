#!/bin/bash

if [ -z "$2" ]; then
    (>&2 echo "ERROR: give URL and an expected string")
    (>&2 echo "Usage example: $0 'http://www.perdu.com/' 'Pas de panique'")
    exit 1
fi

url=$1
expected=$2

status=0

page=$(curl --silent -f "$url")
if [ $? -eq 0 ]; then
    n=$(echo "$page" | grep "$expected" | wc -l)
    if [ $n -gt 0 ]; then
	status=1
    fi
fi

echo "FOUND_EXPECTED:" $status
