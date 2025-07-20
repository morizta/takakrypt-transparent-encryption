#!/bin/bash
# Test script for ntoi user - Full admin access
# Expected: Full access to all guard points

set -e

echo "=== Testing ntoi User Access (Admin) ==="
echo "User: ntoi (UID: 1000)"
echo "Expected: Full access to all guard points"
echo "================================================"
echo

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test and report
test_operation() {
    local operation=$1
    local expected=$2
    local actual_result
    
    if eval "$operation" 2>/dev/null; then
        actual_result="SUCCESS"
    else
        actual_result="FAILED"
    fi
    
    if [ "$expected" = "$actual_result" ]; then
        echo -e "${GREEN}✓ PASS${NC} - $operation"
    else
        echo -e "${RED}✗ FAIL${NC} - $operation (Expected: $expected, Got: $actual_result)"
    fi
}

# Test function with detailed output
test_with_output() {
    local description=$1
    local command=$2
    local expected=$3
    
    echo -e "\n${YELLOW}Testing: $description${NC}"
    echo "Command: $command"
    
    if [ "$expected" = "SUCCESS" ]; then
        if eval "$command"; then
            echo -e "${GREEN}✓ Result: SUCCESS - As expected${NC}"
        else
            echo -e "${RED}✗ Result: FAILED - Expected SUCCESS${NC}"
        fi
    else
        if eval "$command" 2>&1 | grep -q "Permission denied\|denied"; then
            echo -e "${GREEN}✓ Result: DENIED - As expected${NC}"
        else
            echo -e "${RED}✗ Result: SUCCESS - Expected DENIED${NC}"
        fi
    fi
}

# Switch to ntoi user
echo "Switching to ntoi user..."
echo

# Test 1: SENSITIVE GUARD POINT
echo "=== 1. TESTING SENSITIVE GUARD POINT (/data/sensitive) ==="
echo "Expected: Full access (read, write, ls, mkdir, rm)"

# Create test structure
sudo -u ntoi mkdir -p /data/sensitive/ntoi-test 2>/dev/null || true
sudo -u ntoi mkdir -p /data/sensitive/testuser1 2>/dev/null || true
sudo -u ntoi mkdir -p /data/sensitive/testuser2 2>/dev/null || true
sudo -u ntoi mkdir -p /data/sensitive/shared 2>/dev/null || true

# Test operations
test_with_output "List root directory" "sudo -u ntoi ls -la /data/sensitive/" "SUCCESS"
test_with_output "Write to root" "echo 'ntoi root test' | sudo -u ntoi tee /data/sensitive/ntoi-root.txt > /dev/null" "SUCCESS"
test_with_output "Read from root" "sudo -u ntoi cat /data/sensitive/ntoi-root.txt" "SUCCESS"
test_with_output "Write to testuser1 folder" "echo 'ntoi in user1' | sudo -u ntoi tee /data/sensitive/testuser1/ntoi-test.txt > /dev/null" "SUCCESS"
test_with_output "Write to testuser2 folder" "echo 'ntoi in user2' | sudo -u ntoi tee /data/sensitive/testuser2/ntoi-test.txt > /dev/null" "SUCCESS"
test_with_output "Create directory" "sudo -u ntoi mkdir -p /data/sensitive/ntoi-newdir" "SUCCESS"
test_with_output "Remove file" "sudo -u ntoi rm -f /data/sensitive/ntoi-root.txt" "SUCCESS"

# Test 2: DATABASE GUARD POINT
echo -e "\n=== 2. TESTING DATABASE GUARD POINT (/data/database) ==="
echo "Expected: Full access with any process"

test_with_output "List database directory" "sudo -u ntoi ls -la /data/database/" "SUCCESS"
test_with_output "Write database file" "echo 'CREATE TABLE test;' | sudo -u ntoi tee /data/database/test.sql > /dev/null" "SUCCESS"
test_with_output "Read database file" "sudo -u ntoi cat /data/database/test.sql" "SUCCESS"
test_with_output "Create DB directory" "sudo -u ntoi mkdir -p /data/database/ntoi-db" "SUCCESS"
test_with_output "Python access" "sudo -u ntoi python3 -c 'open(\"/data/database/python-test.txt\", \"w\").write(\"python test\")'" "SUCCESS"

# Test 3: PUBLIC GUARD POINT
echo -e "\n=== 3. TESTING PUBLIC GUARD POINT (/data/public) ==="
echo "Expected: Full access (no encryption)"

test_with_output "List public directory" "sudo -u ntoi ls -la /data/public/" "SUCCESS"
test_with_output "Write public file" "echo 'public data' | sudo -u ntoi tee /data/public/ntoi-public.txt > /dev/null" "SUCCESS"
test_with_output "Read public file" "sudo -u ntoi cat /data/public/ntoi-public.txt" "SUCCESS"
test_with_output "Create public directory" "sudo -u ntoi mkdir -p /data/public/ntoi-docs" "SUCCESS"

# Test 4: Cross-user access verification
echo -e "\n=== 4. TESTING CROSS-USER ACCESS ==="
echo "Testing ntoi's ability to access other users' folders"

test_with_output "Access testuser1's files" "sudo -u ntoi ls -la /data/sensitive/testuser1/" "SUCCESS"
test_with_output "Access testuser2's files" "sudo -u ntoi ls -la /data/sensitive/testuser2/" "SUCCESS"
test_with_output "Modify testuser1's files" "echo 'admin override' | sudo -u ntoi tee /data/sensitive/testuser1/admin-note.txt > /dev/null" "SUCCESS"

# Summary
echo -e "\n================================================"
echo "TEST SUMMARY FOR NTOI USER:"
echo "- Should have FULL access to ALL guard points"
echo "- Should be able to use ANY process"
echo "- Should be able to access other users' folders"
echo "================================================"