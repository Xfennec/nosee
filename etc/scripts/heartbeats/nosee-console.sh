#!/bin/bash

# nosee console heartbeat URL
url="http://localhost:8080/heartbeat"

# NOSEE_SRV, VERSION, DATETIME, STARTTIME, UPTIME

curl -s -f -w "HTTP Code %{http_code}\n" \
    --form-string "uptime=$UPTIME" \
    --form-string "server=$NOSEE_SRV" \
    --form-string "version=$VERSION" \
    "$url"

if [ $? -ne 0 ]; then
    exit 1
fi
