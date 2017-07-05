#!/bin/bash

# If you are using SSH keys with private passphrase:
# This sample script runs an agent for the current user, creating
# a socket that the nosee service will use.

agent_link="$HOME/.ssh-agent-sock"

if [ -S "$agent_link" ]; then
    echo "Agent is already here."
    exit 0
fi

eval $(ssh-agent -a "$agent_link")
ssh-add "$HOME/keys/id_rsa_xxx"
ssh-add "$HOME/keys/id_rsa_yyy"
# ...
