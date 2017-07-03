#!/bin/bash

# ex: backup.sh /tmp/backup.start /tmp/backup.ok

if [ -z "$2" ]; then
    (>&2 echo "ERROR: give 'start' flag file and 'ok' flag file")
    exit 1
fi

start_file="$1"
ok_file="$2"

if [ ! -f "$start_file" ]; then
    (>&2 echo "ERROR: can't read start file '$start_file'")
    exit 1
fi
if [ ! -f "$ok_file" ]; then
    (>&2 echo "ERROR: can't read ok file '$ok_file'")
    exit 1
fi

ok_tmsp=$(date +%s -r "$ok_file")
start_tmsp=$(date +%s -r "$start_file")
now=$(date +%s)

last_ok_hours=$(echo $ok_tmsp $now | awk '{ diff=$2-$1; print diff/60/60 }')
last_duration=$(echo $start_tmsp $ok_tmsp | awk '{
    diff=$2-$1;
    if (diff > 0)
	print diff/60/60
    else
	print 0
}')

echo "LAST_OK_HOURS:" $last_ok_hours
echo "LAST_DURATION_HOURS:" $last_duration
