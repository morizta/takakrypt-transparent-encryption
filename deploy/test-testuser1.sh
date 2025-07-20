#!/bin/bash
# Test script for testuser1 - Limited access to own folder only
# Expected: Access to /data/sensitive/testuser1/ only, denied elsewhere

set -e

echo "=== Testing testuser1 User Access ==="
echo "User: testuser1 (UID: 1001)"
echo "Expected: Access to /data/sensitive/testuser1/ folder only"
echo "Allowed processes: text-editors, python-apps"
echo "================================================"
echo

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test function with detailed output
test_with_output() {
    local description=$1
    local command=$2
    local expected=$3
    
    echo -e "\n${YELLOW}Testing: $description${NC}"
    echo "Command: $command"
    
    if [ "$expected" = "SUCCESS" ]; then
        if eval "$command" 2>/dev/null; then
            echo -e "${GREEN}✓ Result: SUCCESS - As expected${NC}"
        else
            echo -e "${RED}✗ Result: FAILED - Expected SUCCESS${NC}"
        fi
    else
        if eval "$command" 2>&1 | grep -q "Permission denied\|denied\|No such file"; then
            echo -e "${GREEN}✓ Result: DENIED - As expected${NC}"
        else
            echo -e "${RED}✗ Result: SUCCESS - Expected DENIED${NC}"
        fi
    fi
}

# Ensure test directories exist
sudo -u ntoi mkdir -p /data/sensitive/testuser1 2>/dev/null || true
sudo -u ntoi mkdir -p /data/sensitive/testuser2 2>/dev/null || true
sudo -u ntoi mkdir -p /data/sensitive/shared 2>/dev/null || true

# Test 1: SENSITIVE GUARD POINT - OWN FOLDER (SHOULD WORK)
echo "=== 1. TESTING OWN FOLDER (/data/sensitive/testuser1/) ==="
echo "Expected: SUCCESS with text editors and python"

test_with_output "List own folder with ls" "sudo -u testuser1 ls -la /data/sensitive/testuser1/" "SUCCESS"
test_with_output "Write with nano (text editor)" "echo 'test content' | sudo -u testuser1 tee /data/sensitive/testuser1/nano-test.txt > /dev/null" "SUCCESS"
test_with_output "Read with cat" "sudo -u testuser1 cat /data/sensitive/testuser1/nano-test.txt" "SUCCESS"
test_with_output "Write with python" "sudo -u testuser1 python3 -c 'open(\"/data/sensitive/testuser1/python-test.txt\", \"w\").write(\"python content\")'" "SUCCESS"
test_with_output "Create subdirectory" "sudo -u testuser1 mkdir -p /data/sensitive/testuser1/subdir" "SUCCESS"

# Test 2: SENSITIVE GUARD POINT - OTHER FOLDERS (SHOULD FAIL)
echo -e "\n=== 2. TESTING OTHER FOLDERS (SHOULD BE DENIED) ==="
echo "Expected: DENIED access to other areas"

test_with_output "List root sensitive directory" "sudo -u testuser1 ls -la /data/sensitive/" "DENIED"
test_with_output "Access testuser2 folder" "sudo -u testuser1 ls -la /data/sensitive/testuser2/" "DENIED"
test_with_output "Write to root sensitive" "echo 'unauthorized' | sudo -u testuser1 tee /data/sensitive/testuser1-hack.txt > /dev/null" "DENIED"
test_with_output "Write to testuser2 folder" "echo 'hack attempt' | sudo -u testuser1 tee /data/sensitive/testuser2/hack.txt > /dev/null" "DENIED"
test_with_output "Access shared folder" "sudo -u testuser1 ls -la /data/sensitive/shared/" "DENIED"

# Test 3: DATABASE GUARD POINT (SHOULD FAIL)
echo -e "\n=== 3. TESTING DATABASE GUARD POINT (SHOULD BE DENIED) ==="
echo "Expected: DENIED - testuser1 has no database access"

test_with_output "List database directory" "sudo -u testuser1 ls -la /data/database/" "DENIED"
test_with_output "Write database file" "echo 'SELECT * FROM hack;' | sudo -u testuser1 tee /data/database/hack.sql > /dev/null" "DENIED"
test_with_output "Python access to database" "sudo -u testuser1 python3 -c 'open(\"/data/database/hack.txt\", \"w\").write(\"hack\")'" "DENIED"

# Test 4: PUBLIC GUARD POINT (SHOULD WORK)
echo -e "\n=== 4. TESTING PUBLIC GUARD POINT (SHOULD WORK) ==="
echo "Expected: SUCCESS - universal access"

test_with_output "List public directory" "sudo -u testuser1 ls -la /data/public/" "SUCCESS"
test_with_output "Write public file" "echo 'public content' | sudo -u testuser1 tee /data/public/testuser1-public.txt > /dev/null" "SUCCESS"
test_with_output "Read public file" "sudo -u testuser1 cat /data/public/testuser1-public.txt" "SUCCESS"
test_with_output "Python write to public" "sudo -u testuser1 python3 -c 'open(\"/data/public/testuser1-python.txt\", \"w\").write(\"public python\")'" "SUCCESS"

# Test 5: PROCESS RESTRICTIONS
echo -e "\n=== 5. TESTING PROCESS RESTRICTIONS ==="
echo "Expected: SUCCESS with allowed processes, DENIED with others"

# Test allowed processes (should work in own folder)
test_with_output "Text editor process (nano simulation)" "echo 'nano test' | sudo -u testuser1 tee /data/sensitive/testuser1/nano-process.txt > /dev/null" "SUCCESS"
test_with_output "Python process" "sudo -u testuser1 python3 -c 'print(\"python works\")' > /dev/null" "SUCCESS"

# Test browsing behavior (ls should work with browsing: true)
echo -e "\n=== 6. TESTING BROWSING BEHAVIOR ==="
echo "Expected: ls should work in allowed folders due to browsing: true"

test_with_output "Browse own folder" "sudo -u testuser1 ls /data/sensitive/testuser1/" "SUCCESS"

# Test edge cases
echo -e "\n=== 7. TESTING EDGE CASES ==="
echo "Testing various access patterns"

test_with_output "Try to create file outside allowed area" "sudo -u testuser1 touch /data/sensitive/hack.txt" "DENIED"
test_with_output "Try to modify permissions" "sudo -u testuser1 chmod 777 /data/sensitive/testuser1/nano-test.txt" "SUCCESS"
test_with_output "Try to access via symlink" "sudo -u testuser1 ln -s /data/sensitive/testuser2 /data/sensitive/testuser1/link 2>/dev/null && sudo -u testuser1 ls /data/sensitive/testuser1/link/" "DENIED"

# Summary
echo -e "\n================================================"
echo "TEST SUMMARY FOR TESTUSER1:"
echo "✓ ALLOWED: /data/sensitive/testuser1/ (read/write)"
echo "✓ ALLOWED: /data/public/ (read/write)"
echo "✗ DENIED:  /data/sensitive/ root and other folders"
echo "✗ DENIED:  /data/database/ (all access)"
echo "✓ PROCESS: text-editors, python-apps allowed"
echo "================================================"