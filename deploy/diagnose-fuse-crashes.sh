#!/bin/bash
# Diagnose why FUSE filesystem keeps crashing

echo "=== FUSE Crash Diagnostics ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

echo "1. Checking system FUSE support:"
lsmod | grep fuse || echo "FUSE module not loaded"
which fusermount3 || echo "fusermount3 not found"
ls -la /dev/fuse 2>/dev/null || echo "/dev/fuse not found"

echo
echo "2. Checking takakrypt agent logs in detail:"
journalctl -u takakrypt --no-pager -n 50 | grep -E "(ERROR|FATAL|panic|crash|failed)"

echo
echo "3. Checking system logs for FUSE errors:"
dmesg | tail -50 | grep -i "fuse\|takakrypt\|segfault\|killed"

echo
echo "4. Testing basic FUSE operations:"
echo "Stopping takakrypt..."
systemctl stop takakrypt

echo "Cleaning up mounts..."
fusermount3 -u /data/sensitive 2>/dev/null || true
fusermount3 -u /data/database 2>/dev/null || true
fusermount3 -u /data/public 2>/dev/null || true
rm -rf /data/{sensitive,database,public}
mkdir -p /data/{sensitive,database,public}

echo "Starting takakrypt with verbose logging..."
systemctl start takakrypt

echo "Waiting 5 seconds for mount..."
sleep 5

echo "Testing immediate access:"
echo "- ls /data/public/ (should work):"
timeout 3 ls /data/public/ 2>&1 || echo "TIMEOUT or ERROR"

echo "- ls /data/sensitive/ (permission test):"
timeout 3 ls /data/sensitive/ 2>&1 || echo "TIMEOUT or ERROR"

echo
echo "5. Process analysis:"
ps aux | grep takakrypt
lsof | grep "/data" 2>/dev/null || echo "No /data files open"

echo
echo "6. Mount analysis:"
mount | grep "/data"
findmnt /data/sensitive /data/database /data/public 2>/dev/null || echo "Mounts not found in findmnt"

echo
echo "7. Configuration analysis:"
echo "Checking takakrypt config:"
ls -la /opt/takakrypt/config/ 2>/dev/null || echo "Config directory not found"

echo "Checking guard point config:"
if [ -f "/opt/takakrypt/config/guard-point.json" ]; then
    echo "Guard points configured:"
    cat /opt/takakrypt/config/guard-point.json | grep -E "(protected_path|policy)" || echo "Cannot parse guard points"
else
    echo "Guard point config not found at /opt/takakrypt/config/"
fi

echo
echo "8. Memory and resource check:"
free -h
df -h /secure_storage 2>/dev/null || echo "Secure storage not accessible"

echo
echo "9. Test minimal operations:"
echo "Testing if filesystem survives minimal operations..."
timeout 5 sudo -u root touch /data/public/test-file.txt 2>&1 && echo "✓ Root can write to public" || echo "✗ Root cannot write to public"
timeout 5 ls /data/public/ 2>&1 > /dev/null && echo "✓ Can list public after write" || echo "✗ Cannot list public after write"

echo
echo "=== Diagnostic Complete ==="
echo "Look for:"
echo "- FUSE module issues (section 1)"
echo "- Takakrypt agent errors (section 2)"
echo "- System crashes/segfaults (section 3)"
echo "- Mount timeouts (section 4)"
echo "- Configuration problems (section 7)"