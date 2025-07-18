#!/bin/bash

echo "=== Cleaning up FUSE mount points ==="

# Stop the agent service first
echo "Stopping takakrypt service..."
sudo systemctl stop takakrypt

# Wait a moment for clean shutdown
sleep 2

# List of mount points to clean up
MOUNT_POINTS=("/data/sensitive" "/data/database" "/data/public")

# Force unmount any stale FUSE mounts
for mount_point in "${MOUNT_POINTS[@]}"; do
    echo "Checking mount point: $mount_point"
    
    # Check if it's mounted
    if mountpoint -q "$mount_point" 2>/dev/null; then
        echo "  Unmounting $mount_point..."
        sudo umount "$mount_point" 2>/dev/null || sudo umount -l "$mount_point"
    fi
    
    # Force unmount with fusermount3 if still connected
    if [ -d "$mount_point" ]; then
        echo "  Force cleanup with fusermount3..."
        sudo fusermount3 -u "$mount_point" 2>/dev/null || true
        sudo fusermount3 -uz "$mount_point" 2>/dev/null || true
    fi
    
    # Remove and recreate directory to ensure clean state
    if [ -d "$mount_point" ]; then
        echo "  Removing and recreating directory..."
        sudo rm -rf "$mount_point"
        sudo mkdir -p "$mount_point"
        sudo chown ntoi:ntoi "$mount_point"
        sudo chmod 755 "$mount_point"
    fi
done

# Clean up any remaining fuse processes
echo "Cleaning up any remaining FUSE processes..."
sudo pkill -f takakrypt-agent 2>/dev/null || true

# Check for any remaining mounts
echo "Checking for remaining FUSE mounts..."
mount | grep -i fuse | grep -E "(sensitive|database|public)" || echo "No remaining FUSE mounts found"

echo "Cleanup completed. You can now restart the agent."