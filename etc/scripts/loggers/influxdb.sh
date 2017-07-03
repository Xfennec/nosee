#!/bin/bash

host=${HOST_FILE%.toml}

# input lines looks like:
# df.toml;DISK_FULLEST_PERC;27
res=$(cat | awk -v host=$host -F\; '{
    probe=$1
    key=$2
    val=$3
    sub(/\.toml$/, "", probe)
    measurement=sprintf("%s_%s", probe, key)
    if (val ~ /[0-9.]/)
	printf("%s,host=%s value=%s\n", measurement,host,val)
    else
	printf("%s,host=%s value=\"%s\"\n", measurement,host,val)
}')

curl -i -XPOST 'http://localhost:8086/write?db=nosee' --data-binary "$res"
