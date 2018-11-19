#!/bin/bash

# Test script to show all input channels

file="/tmp/remove_me"

echo "stdout test"
(>&2 echo "stderr test")

date > $file
echo "$0" >> $file
echo "$1" >> $file
echo "$2" >> $file
echo "$3" >> $file
echo "$4" >> $file

echo "$SUBJECT" >> $file
echo $USER >> $file
echo $TYPE >> $file
echo $NOSEE_SRV >> $file

# stdin is $DETAILS
cat >> $file
echo $HOME >> $file
