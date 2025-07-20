#!/bin/bash
# Test script for testuser2 - Limited access to own folder only
# Expected: Access to /data/sensitive/testuser2/ only, denied elsewhere

set -e

echo "=== Testing testuser2 User Access ==="
echo "User: testuser2 (UID: 1002)"
echo "Expected: Access to /data/sensitive/testuser2/ folder only"
echo "Allowed processes: text-editors only"
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
echo "=== 1. TESTING OWN FOLDER (/data/sensitive/testuser2/) ==="
echo "Expected: SUCCESS with text editors only"

test_with_output "List own folder with ls" "sudo -u testuser2 ls -la /data/sensitive/testuser2/" "SUCCESS"
test_with_output "Write with nano (text editor)" "echo 'testuser2 content' | sudo -u testuser2 tee /data/sensitive/testuser2/nano-test.txt > /dev/null" "SUCCESS"
test_with_output "Read with cat" "sudo -u testuser2 cat /data/sensitive/testuser2/nano-test.txt" "SUCCESS"
test_with_output "Create subdirectory" "sudo -u testuser2 mkdir -p /data/sensitive/testuser2/documents" "SUCCESS"
test_with_output "Write text file" "echo 'my document' | sudo -u testuser2 tee /data/sensitive/testuser2/mydoc.txt > /dev/null" "SUCCESS"

# Test 2: PROCESS RESTRICTIONS - PYTHON SHOULD FAIL
echo -e "\n=== 2. TESTING PROCESS RESTRICTIONS ==="
echo "Expected: text-editors allowed, python DENIED"

test_with_output "Python process (SHOULD FAIL)" "sudo -u testuser2 python3 -c 'open(\"/data/sensitive/testuser2/python-test.txt\", \"w\").write(\"python content\")'" "DENIED"
test_with_output "Text editor simulation (SHOULD WORK)" "echo 'editor content' | sudo -u testuser2 tee /data/sensitive/testuser2/editor.txt > /dev/null" "SUCCESS"

# Test 3: SENSITIVE GUARD POINT - OTHER FOLDERS (SHOULD FAIL)
echo -e "\n=== 3. TESTING OTHER FOLDERS (SHOULD BE DENIED) ==="
echo "Expected: DENIED access to other areas"

test_with_output "List root sensitive directory" "sudo -u testuser2 ls -la /data/sensitive/" "DENIED"
test_with_output "Access testuser1 folder" "sudo -u testuser2 ls -la /data/sensitive/testuser1/" "DENIED"
test_with_output "Write to root sensitive" "echo 'unauthorized' | sudo -u testuser2 tee /data/sensitive/testuser2-hack.txt > /dev/null" "DENIED"
test_with_output "Write to testuser1 folder" "echo 'hack attempt' | sudo -u testuser2 tee /data/sensitive/testuser1/hack.txt > /dev/null" "DENIED"
test_with_output "Access shared folder" "sudo -u testuser2 ls -la /data/sensitive/shared/" "DENIED"

# Test 4: DATABASE GUARD POINT (SHOULD FAIL)
echo -e "\n=== 4. TESTING DATABASE GUARD POINT (SHOULD BE DENIED) ==="
echo "Expected: DENIED - testuser2 has no database access"

test_with_output "List database directory" "sudo -u testuser2 ls -la /data/database/" "DENIED"
test_with_output "Write database file" "echo 'SELECT * FROM hack;' | sudo -u testuser2 tee /data/database/hack.sql > /dev/null" "DENIED"
test_with_output "Python access to database" "sudo -u testuser2 python3 -c 'open(\"/data/database/hack.txt\", \"w\").write(\"hack\")'" "DENIED"

# Test 5: PUBLIC GUARD POINT (SHOULD WORK)
echo -e "\n=== 5. TESTING PUBLIC GUARD POINT (SHOULD WORK) ==="
echo "Expected: SUCCESS - universal access"

test_with_output "List public directory" "sudo -u testuser2 ls -la /data/public/" "SUCCESS"
test_with_output "Write public file" "echo 'public content by user2' | sudo -u testuser2 tee /data/public/testuser2-public.txt > /dev/null" "SUCCESS"
test_with_output "Read public file" "sudo -u testuser2 cat /data/public/testuser2-public.txt" "SUCCESS"
test_with_output "Create public directory" "sudo -u testuser2 mkdir -p /data/public/testuser2-docs" "SUCCESS"

# Test 6: BROWSING BEHAVIOR
echo -e "\n=== 6. TESTING BROWSING BEHAVIOR ==="
echo "Expected: ls should work in allowed folders due to browsing: true"

test_with_output "Browse own folder" "sudo -u testuser2 ls /data/sensitive/testuser2/" "SUCCESS"
test_with_output "Browse public folder" "sudo -u testuser2 ls /data/public/" "SUCCESS"

# Test 7: COMPARISON WITH TESTUSER1 PRIVILEGES
echo -e "\n=== 7. TESTING PRIVILEGE DIFFERENCES ==="
echo "testuser2 has fewer process privileges than testuser1"

test_with_output "Python in own folder (denied for user2)" "sudo -u testuser2 python3 -c 'print(\"test\")' > /dev/null" "DENIED"
test_with_output "Compare: testuser1 CAN use python" "sudo -u testuser1 python3 -c 'print(\"test\")' > /dev/null" "SUCCESS"

# Test 8: EDGE CASES
echo -e "\n=== 8. TESTING EDGE CASES ==="
echo "Testing various access patterns"

test_with_output "Try to create file outside allowed area" "sudo -u testuser2 touch /data/sensitive/hack2.txt" "DENIED"
test_with_output "Try to access via relative path" "cd /data/sensitive/testuser2 && sudo -u testuser2 ls ../testuser1/" "DENIED"
test_with_output "Try to modify permissions in own folder" "sudo -u testuser2 chmod 777 /data/sensitive/testuser2/nano-test.txt" "SUCCESS"

# Test 9: FILE OPERATIONS IN OWN FOLDER
echo -e "\n=== 9. TESTING DETAILED FILE OPERATIONS ==="
echo "Testing comprehensive file operations in allowed folder"

test_with_output "Create multiple files" "sudo -u testuser2 touch /data/sensitive/testuser2/{file1,file2,file3}.txt" "SUCCESS"
test_with_output "Copy files" "sudo -u testuser2 cp /data/sensitive/testuser2/file1.txt /data/sensitive/testuser2/file1-copy.txt" "SUCCESS"
test_with_output "Move files" "sudo -u testuser2 mv /data/sensitive/testuser2/file2.txt /data/sensitive/testuser2/renamed.txt" "SUCCESS"
test_with_output "Delete files" "sudo -u testuser2 rm /data/sensitive/testuser2/file3.txt" "SUCCESS"

# Summary
echo -e "\n================================================"
echo "TEST SUMMARY FOR TESTUSER2:"
echo "✓ ALLOWED: /data/sensitive/testuser2/ (read/write)"
echo "✓ ALLOWED: /data/public/ (read/write)"  
echo "✗ DENIED:  /data/sensitive/ root and other folders"
echo "✗ DENIED:  /data/database/ (all access)"
echo "✓ PROCESS: text-editors only (more restricted than testuser1)"
echo "✗ PROCESS: python NOT allowed (unlike testuser1)"
echo "================================================"