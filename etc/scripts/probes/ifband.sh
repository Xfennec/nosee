#!/bin/bash

interface=$1
if_dir="/sys/class/net/$interface/statistics"
stat_file="$HOME/.ifband-$interface"
NOW=$(date +%s)

if [ -z "$1" ]; then
    (>&2 echo "USAGE: $0 interface-name")
    exit 1
fi

if [ ! -d $if_dir ]; then
    (>&2 echo "ERROR: unable to find $interface stats")
    exit 1
fi

LAST_CALL=$NOW
LAST_RX=$(cat $if_dir/rx_bytes)
LAST_TX=$(cat $if_dir/tx_bytes)

if [ -f $stat_file ]; then
. $stat_file
fi

RX=$(cat $if_dir/rx_bytes)
TX=$(cat $if_dir/tx_bytes)

time_diff=$(echo $LAST_CALL $NOW | awk '{print ($2 - $1)}')
rx_diff=$(echo $LAST_RX $RX | awk '{print ($2 - $1)}')
tx_diff=$(echo $LAST_TX $TX | awk '{print ($2 - $1)}')

#echo $time_diff $rx_diff $tx_diff
if [ $time_diff -eq 0 ]; then
    RX_KBPS=0
    TX_KBPS=0
else
    RX_KBPS=$(echo $rx_diff $time_diff | awk '{printf ("%i", $1 / $2 / 1024)}')
    TX_KBPS=$(echo $tx_diff $time_diff | awk '{printf ("%i", $1 / $2 / 1024)}')
fi

if [ $RX_KBPS -le 0 ]; then
    RX_KBPS=0
fi
if [ $TX_KBPS -le 0 ]; then
    TX_KBPS=0
fi

echo > $stat_file
echo "LAST_CALL=$NOW" >> $stat_file
echo "LAST_RX=$RX" >> $stat_file
echo "LAST_TX=$TX" >> $stat_file

echo RX_KBPS: $RX_KBPS
echo TX_KBPS: $TX_KBPS

