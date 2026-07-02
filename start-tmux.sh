#!/bin/bash

# Ensure the 'prod' session exists
if ! tmux has-session -t rose 2>/dev/null; then
  # Create a new session named 'prod', detached
  cd /workspaces/rose
  tmux new-session -d -s rose
fi
