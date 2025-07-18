#!/bin/bash

echo "=== Policy Debug Information ==="
echo ""

echo "Current user information:"
echo "User: $(whoami)"
echo "UID: $(id -u)"
echo "GID: $(id -g)"
echo "Groups: $(id -G)"

echo ""
echo "Mount points:"
mount | grep takakrypt

echo ""
echo "FUSE processes:"
ps aux | grep takakrypt

echo ""
echo "Agent status:"
sudo systemctl status takakrypt --no-pager

echo ""
echo "Recent policy evaluation logs:"
sudo journalctl -u takakrypt -n 20 --no-pager | grep -E "\[POLICY\]" | tail -10

echo ""
echo "Testing simple file operations as different users:"

echo ""
echo "1. Testing as current user (ntoi):"
echo "Test data" > /tmp/test-file.txt
cat /tmp/test-file.txt
rm /tmp/test-file.txt

echo ""
echo "2. Testing directory permissions:"
ls -la /data/
ls -la /data/sensitive/ || echo "Cannot list /data/sensitive/"

echo ""
echo "3. Testing with touch command:"
touch /data/sensitive/debug-test.txt 2>&1 || echo "Touch failed"
ls -la /data/sensitive/debug-test.txt 2>/dev/null || echo "File not created"

echo ""
echo "4. User set configuration check:"
echo "Checking if ntoi (UID 1000) is in user sets..."
grep -A 5 -B 5 '"uid": 1000' /opt/takakrypt/config/user_set.json 2>/dev/null || echo "Cannot read user set config"

echo ""
echo "=== Debug Complete ==="