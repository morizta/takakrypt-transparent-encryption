#!/bin/bash

echo "🚀 Manual FUSE Test"

# Start agent in background
echo "Starting agent..."
go run ./cmd/agent --config ./ --log-level info &
AGENT_PID=$!

# Wait for startup
sleep 3

echo "Testing with Go program..."
go run ./cmd/simple-test

# Clean up
echo "Stopping agent..."
kill $AGENT_PID
sleep 1