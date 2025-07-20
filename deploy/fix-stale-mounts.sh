#!/bin/bash
# Fix stale FUSE mounts for Takakrypt

echo "=== Fixing Stale FUSE Mounts ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

echo "1. Stopping takakrypt service..."
systemctl stop takakrypt

echo "2. Cleaning up stale mounts..."
# Force unmount all potential stale mounts
fusermount3 -u /data/sensitive 2>/dev/null || umount -f /data/sensitive 2>/dev/null || true
fusermount3 -u /data/database 2>/dev/null || umount -f /data/database 2>/dev/null || true
fusermount3 -u /data/public 2>/dev/null || umount -f /data/public 2>/dev/null || true

echo "3. Recreating mount point directories..."
rm -rf /data/sensitive /data/database /data/public 2>/dev/null || true
mkdir -p /data/{sensitive,database,public}
chmod 755 /data/{sensitive,database,public}

echo "4. Starting takakrypt service..."
systemctl start takakrypt

# Wait a moment for mounts to complete
sleep 3

echo "5. Verifying mounts..."
echo
if mount | grep -q "/data/sensitive" && mount | grep -q "/data/database" && mount | grep -q "/data/public"; then
    echo "✅ SUCCESS: All guard points mounted correctly"
    echo
    mount | grep /data
    echo
    echo "Testing access:"
    ls -la /data/sensitive/ && echo "✅ Sensitive accessible"
    ls -la /data/database/ && echo "✅ Database accessible"  
    ls -la /data/public/ && echo "✅ Public accessible"
else
    echo "❌ FAILED: Some guard points not mounted"
    echo
    echo "Current mounts:"
    mount | grep /data || echo "No /data mounts found"
    echo
    echo "Service status:"
    systemctl status takakrypt --no-pager
    echo
    echo "Recent logs:"
    journalctl -u takakrypt --no-pager -n 10
fi

echo
echo "Fix complete!"