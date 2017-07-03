#!/bin/bash

lines=$(df -kP | grep '^/dev/' | grep -v '[[:space:]]/mnt/')
fullest=$(echo "$lines" | awk '{print $5}' | cut -d% -f1 | sort -n | tail -n1)

echo "FULLEST_PERC:" $fullest

all=$(echo "$lines" | awk '{print $5,$6}')
while read -r line; do
    dfree=$(echo "$line" | awk '{print $1}' | cut -d% -f1)
    name=$(echo "$line" | awk '{print $2}')
    name=$(echo "$name" | sed 's#/#_#g' | sed 's/^_//')
    if [ -z "$name" ]; then
	name="ROOT"
    fi
    echo "DF_${name^^}_PERC:" $dfree
done <<< "$all"
