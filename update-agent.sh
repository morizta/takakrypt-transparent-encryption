#!/bin/bash

# Stop the agent service
sudo systemctl stop takakrypt

# Build the agent from source
echo "Building agent..."
go build -o takakrypt-agent cmd/agent/main.go

# Copy the new binary
sudo cp takakrypt-agent /opt/takakrypt/

# Copy updated configuration files
echo "Updating configuration files..."
sudo cp deploy/ubuntu-config/*.json /opt/takakrypt/config/

# Start the agent service
sudo systemctl start takakrypt

# Check status
sudo systemctl status takakrypt