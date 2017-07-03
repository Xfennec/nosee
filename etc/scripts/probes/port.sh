#!/bin/bash

if [ -z "$1" ]; then
    (>&2 echo "ERROR: give port number (ex: 443)")
    exit 1
fi

nc -z localhost $1 > /dev/null 2>&1
res=$?

open=0
if [ $res -eq 0 ]; then
    open=1
fi

echo "OPEN:" $open
