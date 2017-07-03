#!/bin/bash

if [ -z "$1" ]; then
    (>&2 echo "ERROR: give thermal zone number (ex: 0)")
    exit 1
fi

file="/sys/class/thermal/thermal_zone$1/temp"

if [ ! -f "$file" ]; then
    (>&2 echo "ERROR: invalid path: $file")
    exit 2
fi

val=$(cat "$file")
temp=$(awk "BEGIN {print $val/1000}")
echo "TEMP:" $temp
