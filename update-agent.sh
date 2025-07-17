#!/bin/bash

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Stop the agent service
sudo systemctl stop takakrypt

# Build the agent from source for Linux
echo "Building agent for Linux..."
cd "$SCRIPT_DIR"
GOOS=linux GOARCH=amd64 go build -o takakrypt-agent cmd/agent/main.go

# Copy the new binary
sudo cp takakrypt-agent /opt/takakrypt/

# Copy updated configuration files
echo "Updating configuration files..."
sudo cp "$SCRIPT_DIR/deploy/ubuntu-config/*.json" /opt/takakrypt/config/

# Start the agent service
sudo systemctl start takakrypt

# Check status
sudo systemctl status takakrypt

echo ""
echo "Agent updated successfully!"
echo "Key store loaded from: /opt/takakrypt/config/keys.json"
echo "To monitor logs: journalctl -u takakrypt -f"