#!/bin/bash

if [ -z "$1" ]; then
    (>&2 echo "ERROR: give unit name (ex: httpd.service)")
    exit 1
fi


status=$(systemctl is-active "$1")
echo "STATUS:" $status
