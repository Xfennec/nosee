#!/bin/bash

if [ -z "$1" ]; then
    (>&2 echo "ERROR: give IP to test")
    exit 1
fi
dest=$1

res=$(ping -qAc5 "$dest")

loss=$(echo "$res" | grep "packets transmitted" | sed -r 's/.* ([0-9]+)%.*/\1/g')
avg=$(echo "$res" | grep "^rtt" | awk -F/ '{print $5}')

echo PING_LOSS_PERC: $loss
echo PING_AVG_MS: $avg
