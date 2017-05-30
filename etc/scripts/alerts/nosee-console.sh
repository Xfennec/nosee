#!/bin/bash

if [ -z "$1" ]; then
    (>&2 echo "ERROR: give nosee console URL (ex: http://localhost:8080/alerts)")
    exit 1
fi

DETAILS=$(cat)

curl -s -f -w "HTTP Code %{http_code}\n" \
    --form-string "type=$TYPE" \
    --form-string "subject=$SUBJECT" \
    --form-string "details=$DETAILS" \
    --form-string "classes=$CLASSES" \
    --form-string "hostname=$HOST_NAME" \
    --form-string "nosee_srv=$NOSEE_SRV" \
    --form-string "uniqueid=$UNIQUEID" \
    --form-string "datetime=$DATETIME" \
    "$1"

if [ $? -ne 0 ]; then
    exit 1
fi
