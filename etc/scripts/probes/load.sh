#!/bin/bash

if [ -f /proc/loadavg ]; then
    load=$(awk '{print $1}' /proc/loadavg)
else
    load_field=$(LANG=C uptime | awk -F, '{print $(NF-2)}')
    load=$(echo "$load_field" | awk -F: '{print $2}')
fi

echo "LOAD:" $load
