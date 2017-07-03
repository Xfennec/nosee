#!/bin/bash

mdstat="/proc/mdstat"

if [ ! -f "$mdstat" ]; then
    (>&2 echo "ERROR: cant find md RAID support ($mdstat)")
    exit 1
fi

fcount=$(grep -c "\[.*_.*\]" $mdstat)

echo "ERR_ARRAYS:" $fcount
