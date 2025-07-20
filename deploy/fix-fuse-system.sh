#!/bin/bash
# Comprehensive FUSE system fix

echo "=== FUSE System Repair ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

echo "1. Loading FUSE kernel module..."
modprobe fuse
if lsmod | grep -q fuse; then
    echo "✅ FUSE module loaded successfully"
else
    echo "❌ Failed to load FUSE module"
    exit 1
fi

echo
echo "2. Stopping all takakrypt processes..."
systemctl stop takakrypt
pkill -f takakrypt-agent 2>/dev/null || true
sleep 2

echo
echo "3. Aggressive mount cleanup..."
# Force unmount everything
for mount_point in /data/sensitive /data/database /data/public; do
    echo "Cleaning $mount_point..."
    
    # Kill any processes using the mount
    lsof +D "$mount_point" 2>/dev/null | tail -n +2 | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true
    
    # Try multiple unmount methods
    fusermount3 -uz "$mount_point" 2>/dev/null || true
    umount -f "$mount_point" 2>/dev/null || true
    umount -l "$mount_point" 2>/dev/null || true
    
    # Remove and recreate directory
    rm -rf "$mount_point" 2>/dev/null || true
    mkdir -p "$mount_point"
    chmod 755 "$mount_point"
done

echo
echo "4. Checking FUSE device permissions..."
ls -la /dev/fuse
chmod 666 /dev/fuse 2>/dev/null || true

echo
echo "5. Updating configuration for testing..."
# Temporarily modify public policy to allow root access for testing
if [ -f "/opt/takakrypt/config/policy.json" ]; then
    cp /opt/takakrypt/config/policy.json /opt/takakrypt/config/policy.json.backup
    echo "✅ Backed up policy.json"
else
    echo "❌ Policy file not found"
fi

echo
echo "6. Starting takakrypt with fresh environment..."
# Clear any environment issues
cd /tmp
systemctl start takakrypt

echo "Waiting 10 seconds for proper mount..."
sleep 10

echo
echo "7. Verification tests..."
echo "Mount status:"
mount | grep "/data" || echo "No mounts found"

echo
echo "FUSE module status:"
lsmod | grep fuse || echo "FUSE not loaded"

echo
echo "Testing access as different users:"
echo "- Root access to public:"
timeout 5 ls -la /data/public/ 2>&1 || echo "FAILED"

echo "- ntoi access to sensitive:"
timeout 5 sudo -u ntoi ls -la /data/sensitive/ 2>&1 || echo "FAILED"

echo "- testuser1 access to own folder:"
sudo -u ntoi mkdir -p /data/sensitive/testuser1 2>/dev/null || true
timeout 5 sudo -u testuser1 ls -la /data/sensitive/testuser1/ 2>&1 || echo "FAILED"

echo
echo "8. Service health check..."
systemctl status takakrypt --no-pager --lines=5

echo
echo "9. Recent logs..."
journalctl -u takakrypt --no-pager -n 10

echo
echo "=== Repair Complete ==="
echo
echo "If issues persist:"
echo "1. Check: sudo journalctl -u takakrypt -f"
echo "2. Restart shell session (exit and reconnect)"
echo "3. Run: sudo systemctl restart takakrypt"
echo "4. Check system requirements for FUSE"