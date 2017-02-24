#!/bin/bash

# Test script to show all input channels

echo "stdout test"
(>&2 echo "stderr test")

date > /tmp/remove_me
echo "$0" >> /tmp/remove_me
echo "$1" >> /tmp/remove_me
echo "$2" >> /tmp/remove_me
echo "$3" >> /tmp/remove_me
echo "$4" >> /tmp/remove_me

echo "$SUBJECT" >> /tmp/remove_me
echo "$DETAILS" >> /tmp/remove_me
echo $USER >> /tmp/remove_me
echo $TYPE >> /tmp/remove_me

# stdin is $DETAILS
cat >> //tmp/remove_me
echo $HOME >> /tmp/remove_me
