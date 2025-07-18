#!/bin/bash

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Stop the agent service and cleanup mounts
sudo systemctl stop takakrypt

# Clean up any stale FUSE mounts
echo "Cleaning up stale FUSE mounts..."
MOUNT_POINTS=("/data/sensitive" "/data/database" "/data/public")
for mount_point in "${MOUNT_POINTS[@]}"; do
    if mountpoint -q "$mount_point" 2>/dev/null; then
        echo "Unmounting $mount_point..."
        sudo umount "$mount_point" 2>/dev/null || sudo umount -l "$mount_point"
    fi
    sudo fusermount3 -u "$mount_point" 2>/dev/null || true
done

# Build the agent from source for Linux
echo "Building agent for Linux..."
cd "$SCRIPT_DIR"
GOOS=linux GOARCH=amd64 go build -o takakrypt-agent cmd/agent/main.go

# Copy the new binary
sudo cp takakrypt-agent /opt/takakrypt/

# Copy updated configuration files
echo "Updating configuration files..."
sudo cp "$SCRIPT_DIR"/deploy/ubuntu-config/*.json /opt/takakrypt/config/

# Start the agent service
sudo systemctl start takakrypt

# Check status
sudo systemctl status takakrypt

echo ""
echo "Agent updated successfully!"
echo "Key store loaded from: /opt/takakrypt/config/keys.json"
echo "To monitor logs: journalctl -u takakrypt -f"