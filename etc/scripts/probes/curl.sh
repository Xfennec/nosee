#!/bin/bash

# the URL must display usual "KEY: val\nKEY2: val2" format

if [ -z "$1" ]; then
    (>&2 echo "ERROR: give URL")
    exit 1
fi

curl --silent -f "$1"
