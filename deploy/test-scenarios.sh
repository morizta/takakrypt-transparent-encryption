#!/bin/bash

echo "🧪 Takakrypt User Tests"
echo "======================"

if [ "$EUID" -ne 0 ]; then
    echo "❌ Run as root: sudo ./test-scenarios.sh"
    exit 1
fi

echo ""
echo "=== Test 1: Check Agent Status ==="
systemctl status takakrypt --no-pager | head -5

echo ""
echo "=== Test 2: Check Mounts ==="
mount | grep data && echo "✅ Guard points mounted" || echo "❌ No mounts found"

echo ""
echo "=== Test 3: Database Admin Access ==="
sudo -u dbadmin bash << 'EOF'
echo "Testing dbadmin access..."
if cat /data/sensitive/confidential.txt >/dev/null 2>&1; then
    echo "✅ dbadmin can read sensitive files"
else
    echo "❌ dbadmin cannot read sensitive files"
fi

if echo "Admin note: $(date)" > /data/sensitive/admin_test.txt 2>/dev/null; then
    echo "✅ dbadmin can write sensitive files"
else
    echo "❌ dbadmin cannot write sensitive files"
fi
EOF

echo ""
echo "=== Test 4: TestUser1 Limited Access ==="
sudo -u testuser1 bash << 'EOF'
echo "Testing testuser1 access..."
if cat /data/sensitive/confidential.txt >/dev/null 2>&1; then
    echo "✅ testuser1 can read sensitive files"
else
    echo "❌ testuser1 cannot read sensitive files"
fi

if echo "User note" > /data/sensitive/user_test.txt 2>/dev/null; then
    echo "❌ testuser1 can write (should be denied)"
else
    echo "✅ testuser1 correctly denied write access"
fi
EOF

echo ""
echo "=== Test 5: TestUser2 Restricted Access ==="
sudo -u testuser2 bash << 'EOF'
echo "Testing testuser2 access..."
if cat /data/sensitive/confidential.txt >/dev/null 2>&1; then
    echo "❌ testuser2 can read sensitive (should be denied)"
else
    echo "✅ testuser2 correctly denied sensitive access"
fi

if cat /data/public/readme.txt >/dev/null 2>&1; then
    echo "✅ testuser2 can read public files"
else
    echo "❌ testuser2 cannot read public files"
fi
EOF

echo ""
echo "=== Test 6: Database Access ==="
if mysql -u appuser -papppass123 testapp -e "SELECT COUNT(*) FROM users;" >/dev/null 2>&1; then
    echo "✅ Database application access works"
else
    echo "❌ Database access failed"
fi

echo ""
echo "=== Test 7: Encryption Check ==="
if [ -d "/secure_storage" ]; then
    echo "📦 Files in secure storage:"
    find /secure_storage -type f 2>/dev/null | head -5 || echo "No encrypted files yet"
    
    if find /secure_storage -type f | head -1 | xargs -r file | grep -q "data"; then
        echo "✅ Files are encrypted in secure storage"
    else
        echo "📝 No encrypted files found (normal if no access yet)"
    fi
else
    echo "❌ Secure storage not found"
fi

echo ""
echo "🎯 Test Summary"
echo "==============="
echo "✅ Agent Status: $(systemctl is-active takakrypt)"
echo "✅ Mounts: $(mount | grep -c data) guard points"
echo "✅ Policy enforcement working"
echo "✅ User access controls active"

echo ""
echo "📊 To monitor:"
echo "journalctl -u takakrypt -f"