#!/bin/bash
# Test Ctrl+C cancellation
./bantam run <<EOF &
pid=$!
sleep 2
kill $pid 2>/dev/null
wait $pid 2>/dev/null
EOF
