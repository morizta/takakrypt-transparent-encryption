#!/bin/bash
# Quick policy test that monitors mount health

echo "=== Quick Policy Test with Mount Monitoring ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

# Function to check mount health
check_mounts() {
    echo "Checking mount health..."
    if mount | grep -q "/data/sensitive" && mount | grep -q "/data/database" && mount | grep -q "/data/public"; then
        echo "✅ Mounts appear healthy"
        # Test actual access
        if timeout 2 ls /data/public/ > /dev/null 2>&1; then
            echo "✅ Public mount accessible"
        else
            echo "❌ Public mount not accessible - restarting..."
            ./fix-stale-mounts.sh
            return 1
        fi
    else
        echo "❌ Some mounts missing - restarting..."
        ./fix-stale-mounts.sh
        return 1
    fi
}

# Initial mount check
check_mounts || exit 1

echo
echo "=== Testing Policy Enforcement ==="

echo "1. Testing ntoi (admin) access:"
sudo -u ntoi ls -la /data/sensitive/ 2>&1 | head -3
if [ $? -eq 0 ]; then
    echo "✅ ntoi can access sensitive root"
else
    echo "❌ ntoi cannot access sensitive root"
fi

echo
echo "2. Testing testuser1 access to own folder:"
# First ensure the folder exists
sudo -u ntoi mkdir -p /data/sensitive/testuser1 2>/dev/null
sudo -u testuser1 ls -la /data/sensitive/testuser1/ 2>&1 | head -3
if [ $? -eq 0 ]; then
    echo "✅ testuser1 can access own folder"
else
    echo "❌ testuser1 cannot access own folder"
fi

echo
echo "3. Testing testuser1 access to other areas (should fail):"
sudo -u testuser1 ls -la /data/sensitive/ 2>&1 | head -3
if [ $? -ne 0 ]; then
    echo "✅ testuser1 properly denied access to root"
else
    echo "❌ testuser1 should not access root"
fi

echo
echo "4. Testing file operations (ntoi):"
if sudo -u ntoi echo "test content" > /data/sensitive/test-file.txt 2>/dev/null; then
    echo "✅ ntoi can write files"
    if sudo -u ntoi cat /data/sensitive/test-file.txt 2>/dev/null; then
        echo "✅ ntoi can read files"
    else
        echo "❌ ntoi cannot read files"
    fi
    sudo -u ntoi rm /data/sensitive/test-file.txt 2>/dev/null
else
    echo "❌ ntoi cannot write files"
fi

echo
echo "5. Testing process restrictions:"
echo "Python process outside guard points:"
sudo -u testuser2 python3 -c "print('Python works outside')" 2>&1
echo "Python process in sensitive (testuser2 should be denied):"
sudo -u testuser2 python3 -c "open('/data/sensitive/testuser2/test.py', 'w').write('test')" 2>&1

echo
echo "6. Final mount check:"
check_mounts

echo
echo "=== Test Complete ==="
echo "If mounts keep failing, there may be a deeper FUSE/policy issue."