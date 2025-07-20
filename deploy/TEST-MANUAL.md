# Takakrypt Transparent Encryption - Manual Testing Guide

## Overview
This document provides comprehensive testing procedures for the Takakrypt transparent encryption system with the new policy configuration featuring subfolder-specific permissions.

## Test Environment Setup

### Prerequisites
1. Takakrypt system installed and running
2. Users created: `ntoi`, `testuser1`, `testuser2`, `dbadmin`
3. Guard points mounted: `/data/sensitive`, `/data/database`, `/data/public`
4. MariaDB/MySQL installed and configured

### Guard Point Configuration
- **Sensitive Guard Point**: `/data/sensitive` → `/secure_storage/sensitive`
- **Database Guard Point**: `/data/database` → `/secure_storage/database`  
- **Public Guard Point**: `/data/public` → `/secure_storage/public`

## Policy Summary

| User | Sensitive GP | Database GP | Public GP | Allowed Processes |
|------|-------------|-------------|-----------|-------------------|
| `ntoi` | Full access (admin) | Full access | Full access | Any process |
| `testuser1` | Own folder only (`/testuser1/`) | No access | Full access | text-editors, python-apps |
| `testuser2` | Own folder only (`/testuser2/`) | No access | Full access | text-editors only |
| `mysql` | No access | Process-based access | No special access | mysql processes |
| Others | No access | No access | Full access | Any process |

## Test Scripts Available

1. **`test-cleanup.sh`** - Cleanup and reset guard points
2. **`test-ntoi-user.sh`** - Test ntoi admin user
3. **`test-testuser1.sh`** - Test testuser1 limited access
4. **`test-testuser2.sh`** - Test testuser2 limited access
5. **`run-all-tests.sh`** - Execute all tests sequentially

## Manual Testing Procedures

### 1. Pre-Test Setup

```bash
# 1. Stop and clean existing setup
sudo ./test-cleanup.sh

# 2. Start takakrypt service
sudo systemctl start takakrypt

# 3. Verify guard points are mounted
mount | grep /data
```

Expected output:
```
takakrypt-fuse on /data/sensitive type fuse
takakrypt-fuse on /data/database type fuse  
takakrypt-fuse on /data/public type fuse
```

### 2. Test ntoi User (Admin)

**Expected Behavior**: Full access to all guard points with any process

```bash
# Run automated test
sudo ./test-ntoi-user.sh

# Manual verification
sudo -u ntoi ls -la /data/sensitive/
sudo -u ntoi ls -la /data/database/
sudo -u ntoi ls -la /data/public/

# Test write access
echo "admin test" | sudo -u ntoi tee /data/sensitive/admin.txt
echo "admin db" | sudo -u ntoi tee /data/database/admin.sql
echo "admin public" | sudo -u ntoi tee /data/public/admin.txt

# Test cross-folder access
sudo -u ntoi mkdir -p /data/sensitive/{testuser1,testuser2,shared}
sudo -u ntoi ls -la /data/sensitive/testuser1/
```

**Expected Results**: All operations should succeed.

### 3. Test testuser1 User

**Expected Behavior**: Access only to `/data/sensitive/testuser1/` and `/data/public/`

```bash
# Run automated test
sudo ./test-testuser1.sh

# Manual verification - SHOULD WORK
sudo -u testuser1 ls -la /data/sensitive/testuser1/
echo "user1 content" | sudo -u testuser1 tee /data/sensitive/testuser1/test.txt
sudo -u testuser1 python3 -c "open('/data/sensitive/testuser1/python.txt', 'w').write('python test')"
sudo -u testuser1 ls -la /data/public/

# Manual verification - SHOULD FAIL
sudo -u testuser1 ls -la /data/sensitive/
sudo -u testuser1 ls -la /data/sensitive/testuser2/
sudo -u testuser1 ls -la /data/database/
echo "hack" | sudo -u testuser1 tee /data/sensitive/hack.txt
```

**Expected Results**: 
- ✅ Own folder operations succeed
- ✅ Public folder operations succeed  
- ❌ Root sensitive folder access denied
- ❌ Other user folders denied
- ❌ Database folder denied

### 4. Test testuser2 User

**Expected Behavior**: Access only to `/data/sensitive/testuser2/` and `/data/public/`, text-editors only

```bash
# Run automated test
sudo ./test-testuser2.sh

# Manual verification - SHOULD WORK
sudo -u testuser2 ls -la /data/sensitive/testuser2/
echo "user2 content" | sudo -u testuser2 tee /data/sensitive/testuser2/test.txt
sudo -u testuser2 ls -la /data/public/

# Manual verification - SHOULD FAIL
sudo -u testuser2 python3 -c "print('test')"  # Process restriction
sudo -u testuser2 ls -la /data/sensitive/
sudo -u testuser2 ls -la /data/sensitive/testuser1/
sudo -u testuser2 ls -la /data/database/
```

**Expected Results**:
- ✅ Own folder operations succeed
- ✅ Public folder operations succeed
- ❌ Python process denied (unlike testuser1)
- ❌ Root sensitive folder access denied
- ❌ Other user folders denied
- ❌ Database folder denied

### 5. Test Database Access

**Expected Behavior**: Only MySQL/MariaDB processes can access database files

```bash
# Test MariaDB access (should work)
sudo systemctl start mariadb
sudo mysql -e "CREATE DATABASE testdb;"
sudo mysql -e "USE testdb; CREATE TABLE test(id INT);"

# Check if database files are created in encrypted storage
sudo ls -la /data/database/
sudo ls -la /secure_storage/database/

# Test direct access (should fail for regular users)
sudo -u testuser1 ls -la /data/database/
echo "hack" | sudo -u testuser1 tee /data/database/hack.sql
```

**Expected Results**:
- ✅ MariaDB can create and access database files
- ✅ Files are encrypted in secure storage
- ❌ Regular users cannot access database files directly

### 6. Test Public Access

**Expected Behavior**: Universal access for all users and processes

```bash
# Test with different users
sudo -u ntoi echo "ntoi public" | tee /data/public/ntoi.txt
sudo -u testuser1 echo "user1 public" | tee /data/public/user1.txt  
sudo -u testuser2 echo "user2 public" | tee /data/public/user2.txt

# Test different processes
sudo -u testuser1 python3 -c "open('/data/public/python-public.txt', 'w').write('python in public')"
sudo -u testuser2 ls -la /data/public/

# Verify no encryption (optional)
sudo ls -la /secure_storage/public/
sudo cat /secure_storage/public/ntoi.txt  # Should be readable
```

**Expected Results**: All operations succeed for all users.

### 7. Test Browsing Permissions

**Expected Behavior**: `ls` should work when `browsing: true` regardless of process restrictions

```bash
# Test ls command specifically
sudo -u ntoi ls /data/sensitive/
sudo -u testuser1 ls /data/sensitive/testuser1/
sudo -u testuser2 ls /data/sensitive/testuser2/

# Test browsing in restricted areas (should fail)
sudo -u testuser1 ls /data/sensitive/testuser2/
sudo -u testuser2 ls /data/sensitive/testuser1/
```

**Expected Results**: 
- ✅ ls works in allowed folders
- ❌ ls fails in restricted folders

## Test Results Matrix

| Operation | ntoi | testuser1 (own) | testuser1 (other) | testuser2 (own) | testuser2 (other) |
|-----------|------|------------------|-------------------|------------------|-------------------|
| `ls /data/sensitive/` | ✅ | ❌ | ❌ | ❌ | ❌ |
| `ls /data/sensitive/testuser1/` | ✅ | ✅ | N/A | ❌ | N/A |
| `ls /data/sensitive/testuser2/` | ✅ | ❌ | N/A | ✅ | N/A |
| `write to own folder` | ✅ | ✅ | N/A | ✅ | N/A |
| `write to other folder` | ✅ | ❌ | ❌ | ❌ | ❌ |
| `ls /data/database/` | ✅ | ❌ | ❌ | ❌ | ❌ |
| `ls /data/public/` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `python process` | ✅ | ✅ | ✅ | ❌ | ❌ |

## Troubleshooting

### Common Issues

1. **Guard points not mounted**
   ```bash
   sudo systemctl restart takakrypt
   mount | grep /data
   ```

2. **Permission denied for ls**
   - Check if browsing policy implementation is correct
   - Verify process set restrictions

3. **MariaDB cannot access files**
   ```bash
   sudo systemctl status mariadb
   sudo journalctl -u mariadb -f
   ```

4. **Policy not taking effect**
   ```bash
   sudo systemctl restart takakrypt
   sudo journalctl -u takakrypt -f
   ```

### Log Analysis

```bash
# Check takakrypt logs
sudo journalctl -u takakrypt -f

# Check for policy evaluation
sudo grep -i "policy" /var/log/takakrypt/takakrypt.log

# Check for access denials
sudo grep -i "denied" /var/log/takakrypt/takakrypt.log
```

## Security Verification

### Expected Security Behaviors

1. **Isolation**: Users cannot access each other's folders
2. **Process Restriction**: testuser2 cannot use python in sensitive areas
3. **Database Protection**: Only MySQL processes access database files
4. **Encryption**: Sensitive and database files are encrypted at rest
5. **Audit Trail**: All access attempts are logged

### Verification Commands

```bash
# Check encrypted storage
sudo ls -la /secure_storage/sensitive/
sudo hexdump -C /secure_storage/sensitive/testuser1/test.txt | head

# Check audit logs
sudo grep "testuser1" /var/log/takakrypt/audit.log
sudo grep "DENIED" /var/log/takakrypt/audit.log

# Verify file encryption
sudo file /secure_storage/sensitive/testuser1/*
```

## Cleanup After Testing

```bash
# Run cleanup script
sudo ./test-cleanup.sh

# Restart fresh
sudo systemctl restart takakrypt
```

## Expected vs Actual Results Template

Use this template to document test results:

```
Test: [Test Description]
User: [username]
Command: [command executed]
Expected: [SUCCESS/DENIED]
Actual: [SUCCESS/DENIED]
Status: [PASS/FAIL]
Notes: [any additional observations]
```