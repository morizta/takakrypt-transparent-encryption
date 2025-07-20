#!/bin/bash
# Debug script for Takakrypt mount issues

echo "=== Takakrypt Mount Diagnostics ==="
echo

# Check service status
echo "1. Service Status:"
systemctl status takakrypt --no-pager
echo

# Check if guard points are mounted
echo "2. Mount Status:"
mount | grep -E "(takakrypt|/data)" || echo "No takakrypt mounts found"
echo

# Check if mount points exist
echo "3. Mount Point Directories:"
ls -la /data/ 2>/dev/null || echo "/data/ directory does not exist"
echo

# Check FUSE processes
echo "4. FUSE Processes:"
ps aux | grep -E "(takakrypt|fuse)" | grep -v grep || echo "No FUSE processes found"
echo

# Check logs
echo "5. Recent Takakrypt Logs:"
journalctl -u takakrypt --no-pager -n 20 || echo "No journalctl logs available"
echo

# Check if takakrypt binary exists and is executable
echo "6. Takakrypt Binary:"
which takakrypt || echo "takakrypt command not found"
ls -la /usr/local/bin/takakrypt 2>/dev/null || ls -la /usr/bin/takakrypt 2>/dev/null || echo "takakrypt binary not found in standard locations"
echo

# Check configuration files
echo "7. Configuration Files:"
ls -la /etc/takakrypt/ 2>/dev/null || echo "/etc/takakrypt/ not found"
ls -la deploy/ubuntu-config/ 2>/dev/null || echo "deploy/ubuntu-config/ not found"
echo

# Check if secure storage directories exist
echo "8. Secure Storage:"
ls -la /secure_storage/ 2>/dev/null || echo "/secure_storage/ not found"
echo

# Check for any mount errors
echo "9. Filesystem Errors:"
dmesg | tail -20 | grep -i "fuse\|mount\|takakrypt" || echo "No recent filesystem errors"
echo

echo "=== Recommended Actions ==="
echo "1. If service is not running: sudo systemctl start takakrypt"
echo "2. If mounts are stale: sudo umount /data/* ; sudo systemctl restart takakrypt"
echo "3. If binary missing: check installation"
echo "4. If config missing: verify configuration files are in place"
echo "5. Check logs for specific error messages"