#!/bin/bash

# Server must have mod_status loaded and configured with something like:
#<Location /server-status>
#    SetHandler server-status
#    Order deny,allow
#    Deny from all
#    Allow from 127.0.0.1 ::1
#</Location>

# ExtendedStatus must be set to On (default since Apache 2.3.6)

stat_file="$HOME/.apache-modstatus"
NOW=$(date +%s)

page=$(curl --silent -f "http://localhost/server-status?auto")
if [ $? -ne 0 ]; then
    (>&2 echo "ERROR: unable to get status (mod_status OK on localhost?)")
    exit 1
fi

requests=$(echo "$page" | grep '^Total Accesses' | awk -F ': ' '{print $2}')
kbytes=$(echo "$page" | grep '^Total kBytes' | awk -F ': ' '{print $2}')

LAST_CALL=$NOW
LAST_REQUESTS=$requests
LAST_KBYTES=$kbytes
if [ -f $stat_file ]; then
. $stat_file
fi

REQUESTS=$requests
KBYTES=$kbytes

time_diff=$(echo $LAST_CALL $NOW | awk '{print ($2 - $1)}')
requests_diff=$(echo $LAST_REQUESTS $REQUESTS | awk '{print ($2 - $1)}')
kbytes_diff=$(echo $LAST_KBYTES $KBYTES | awk '{print ($2 - $1)}')

if [ $time_diff -eq 0 ]; then
    RPS=0
    KBPS=0
else
    RPS=$(echo $requests_diff $time_diff | awk '{t=$1/$2; printf ("%f", (t>0?t:0))}')
    KBPS=$(echo $kbytes_diff $time_diff | awk '{t=$1/$2; printf ("%f", (t>0?t:0))}')
fi


echo > $stat_file
echo "LAST_CALL=$NOW" >> $stat_file
echo "LAST_REQUESTS=$REQUESTS" >> $stat_file
echo "LAST_KBYTES=$KBYTES" >> $stat_file

echo RPS: $RPS
echo KBPS: $KBPS
