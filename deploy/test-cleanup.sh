#!/bin/bash
# Cleanup script for Takakrypt guard points
# This script unmounts guard points and cleans up test data

set -e

echo "=== Takakrypt Guard Point Cleanup Script ==="
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

# Function to unmount guard point
unmount_guard_point() {
    local mount_point=$1
    local mount_name=$(basename $mount_point)
    
    echo "Checking guard point: $mount_point"
    
    if mount | grep -q "$mount_point"; then
        echo "  - Unmounting $mount_point..."
        fusermount -u "$mount_point" 2>/dev/null || umount "$mount_point" 2>/dev/null || {
            echo "  - Force unmounting $mount_point..."
            umount -f "$mount_point" 2>/dev/null || true
        }
        echo "  - Unmounted successfully"
    else
        echo "  - Not mounted"
    fi
}

# Function to clean directory
clean_directory() {
    local dir=$1
    
    if [ -d "$dir" ]; then
        echo "Cleaning directory: $dir"
        rm -rf "$dir"/*
        echo "  - Cleaned"
    else
        echo "Directory $dir does not exist"
    fi
}

# Stop takakrypt service if running
echo "Stopping takakrypt service..."
systemctl stop takakrypt 2>/dev/null || echo "  - Service not running"
echo

# Unmount all guard points
echo "Unmounting guard points..."
unmount_guard_point "/data/sensitive"
unmount_guard_point "/data/database"
unmount_guard_point "/data/public"
echo

# Clean secure storage
echo "Cleaning secure storage..."
clean_directory "/secure_storage/sensitive"
clean_directory "/secure_storage/database"
clean_directory "/secure_storage/public"
echo

# Clean mount points (optional - comment out if you want to keep the structure)
read -p "Do you want to clean mount point directories? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleaning mount points..."
    clean_directory "/data/sensitive"
    clean_directory "/data/database"
    clean_directory "/data/public"
    echo
fi

# Create test folder structure if requested
read -p "Do you want to create test folder structure? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Creating test folder structure..."
    
    # Sensitive folders
    mkdir -p /secure_storage/sensitive/testuser1
    mkdir -p /secure_storage/sensitive/testuser2
    mkdir -p /secure_storage/sensitive/shared
    
    # Database folders
    mkdir -p /secure_storage/database/myapp
    
    # Public folders
    mkdir -p /secure_storage/public/documents
    
    # Set permissions
    chown -R ntoi:ntoi /secure_storage/sensitive
    chown testuser1:testuser1 /secure_storage/sensitive/testuser1
    chown testuser2:testuser2 /secure_storage/sensitive/testuser2
    chmod -R 755 /secure_storage/sensitive
    
    chown -R mysql:mysql /secure_storage/database
    chmod -R 750 /secure_storage/database
    
    chmod -R 755 /secure_storage/public
    
    echo "  - Test folder structure created"
fi

echo
echo "Cleanup complete!"
echo
echo "To restart takakrypt:"
echo "  sudo systemctl start takakrypt"
echo
echo "To check status:"
echo "  sudo systemctl status takakrypt"
echo "  mount | grep /data"