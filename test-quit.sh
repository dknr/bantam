#!/bin/bash
# Test script to verify Ctrl+C behavior
# Sends Ctrl+C three times and expects exit on first press

# Use 'expect' or send Ctrl+C via /dev/tty
# For now, just manually verify with: timeout 3 ./bantam run < /dev/stdin

echo "Testing: Press Ctrl+C once to exit (empty prompt)"
timeout 3 ./bantam run 2>&1
