#!/bin/bash

# load.sh [prog1] [prog2] [script3] [...]
# will return "LOAD_PROG_DETECTED: 1" if any
# of the prog/script is found ("my load is high
# but my backup is running, so it's ok")

if [ -f /proc/loadavg ]; then
    load=$(awk '{print $1}' /proc/loadavg)
else
    load_field=$(LANG=C uptime | awk -F, '{print $(NF-2)}')
    load=$(echo "$load_field" | awk -F: '{print $2}')
fi

detected=0
if [ -n $2 ]; then
    while [ ${#} -gt 0 ]; do
        pidof -x "$1" > /dev/null
        if [ $? -eq 0 ]; then
            detected=1
        fi
        shift
    done
fi

echo "LOAD:" $load
echo "CPU_COUNT:" $(nproc)
echo "LOAD_PROG_DETECTED:" $detected
