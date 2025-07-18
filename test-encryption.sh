#!/bin/bash

echo "=== Testing Transparent Encryption ==="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test mount point
MOUNT_POINT="/data/sensitive"
BACKING_STORE="/secure_storage/sensitive"

# Function to check result
check_result() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $1${NC}"
    else
        echo -e "${RED}✗ $1${NC}"
    fi
}

echo -e "${YELLOW}>>> Creating test file with sensitive data${NC}"
echo "Top Secret: SSN 123-45-6789, Credit Card: 4111-1111-1111-1111" > $MOUNT_POINT/sensitive.txt
check_result "Created sensitive file"

echo ""
echo -e "${YELLOW}>>> Reading through FUSE mount (should show plaintext)${NC}"
cat $MOUNT_POINT/sensitive.txt
check_result "Read through FUSE mount"

echo ""
echo -e "${YELLOW}>>> Checking backing store (should show encrypted data)${NC}"
echo "Raw encrypted file content:"
ls -la $BACKING_STORE/sensitive.txt
echo ""
echo "First 100 bytes of encrypted file:"
xxd $BACKING_STORE/sensitive.txt | head -5

echo ""
echo -e "${YELLOW}>>> Testing file permissions${NC}"
ls -la $MOUNT_POINT/sensitive.txt

echo ""
echo -e "${YELLOW}>>> Testing file encryption detection${NC}"
file $BACKING_STORE/sensitive.txt

echo ""
echo -e "${YELLOW}>>> Testing direct read from backing store (should be encrypted)${NC}"
echo "Attempting to read encrypted file directly:"
head -c 50 $BACKING_STORE/sensitive.txt | od -c

echo ""
echo -e "${YELLOW}>>> Testing file size comparison${NC}"
echo "FUSE mount file size: $(stat -c%s $MOUNT_POINT/sensitive.txt) bytes"
echo "Backing store file size: $(stat -c%s $BACKING_STORE/sensitive.txt) bytes"

echo ""
echo -e "${YELLOW}>>> Testing multiple operations${NC}"
echo "Additional secret data" >> $MOUNT_POINT/sensitive.txt
check_result "Appended to file"

echo "Final content through FUSE:"
cat $MOUNT_POINT/sensitive.txt

echo ""
echo "=== Encryption Test Complete ==="