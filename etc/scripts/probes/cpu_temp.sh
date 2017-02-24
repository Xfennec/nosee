#!/bin/bash

if [ -z "$1" ]; then
    (>&2 echo "ERROR: give thermal zone number (ex: 0)")
    (>&2 echo "second error line")
    exit 1
fi

if [ "$1" = "stop" ]; then
    echo "# Ok then… bye!"
    exit 0
fi

file="/sys/class/thermal/thermal_zone$1/temp"

if [ ! -f "$file" ]; then
    (>&2 echo "ERROR: invalid path: $file")
    exit 2
fi

val=$(cat "$file")
temp=$(awk "BEGIN {print $val/1000}")
echo "CPU${1}_TEMP:" $temp
