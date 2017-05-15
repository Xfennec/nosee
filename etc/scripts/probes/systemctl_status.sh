#!/bin/bash

if [ -z "$2" ]; then
    (>&2 echo "ERROR: give unit name (ex: httpd.service) and short name (ex: HTTPD)")
    exit 1
fi


status=$(systemctl is-active "$1")
echo "SYSD_STATUS_$2:" $status
