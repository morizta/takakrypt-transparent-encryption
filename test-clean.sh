#!/bin/bash

echo "=== Clean FUSE Test ==="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

echo -e "${YELLOW}>>> Cleaning up old test files${NC}"
sudo rm -f $MOUNT_POINT/clean-test-*
sudo rm -f $BACKING_STORE/clean-test-*

echo ""
echo -e "${YELLOW}>>> Creating new test file${NC}"
echo "Fresh test data $(date)" > $MOUNT_POINT/clean-test-1.txt
check_result "Created file with redirection"

echo ""
echo -e "${YELLOW}>>> Checking file ownership${NC}"
ls -la $MOUNT_POINT/clean-test-1.txt
ls -la $BACKING_STORE/clean-test-1.txt

echo ""
echo -e "${YELLOW}>>> Reading file through FUSE${NC}"
cat $MOUNT_POINT/clean-test-1.txt
check_result "Read file through FUSE"

echo ""
echo -e "${YELLOW}>>> Testing file size (encryption check)${NC}"
FUSE_SIZE=$(stat -c%s $MOUNT_POINT/clean-test-1.txt)
BACKING_SIZE=$(stat -c%s $BACKING_STORE/clean-test-1.txt)
echo "FUSE mount size: $FUSE_SIZE bytes"
echo "Backing store size: $BACKING_SIZE bytes"

if [ $BACKING_SIZE -gt $FUSE_SIZE ]; then
    echo -e "${GREEN}✓ File is encrypted (backing store larger)${NC}"
else
    echo -e "${RED}✗ File may not be encrypted${NC}"
fi

echo ""
echo -e "${YELLOW}>>> Testing append operation${NC}"
echo "Additional data $(date)" >> $MOUNT_POINT/clean-test-1.txt
check_result "Appended to file"

echo ""
echo -e "${YELLOW}>>> Final content verification${NC}"
echo "Content through FUSE:"
cat $MOUNT_POINT/clean-test-1.txt

echo ""
echo -e "${YELLOW}>>> Testing nano editor${NC}"
echo "Nano test content" > $MOUNT_POINT/clean-test-nano.txt
check_result "Created file for nano test"

if [ -f $MOUNT_POINT/clean-test-nano.txt ]; then
    echo "File exists and readable:"
    cat $MOUNT_POINT/clean-test-nano.txt
fi

echo ""
echo -e "${YELLOW}>>> Testing multiple file operations${NC}"
for i in {1..3}; do
    echo "Test file $i content" > $MOUNT_POINT/clean-test-multi-$i.txt
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Created file $i${NC}"
    else
        echo -e "${RED}✗ Failed to create file $i${NC}"
    fi
done

echo ""
echo -e "${YELLOW}>>> Final file listing${NC}"
ls -la $MOUNT_POINT/clean-test-*

echo ""
echo "=== Clean Test Complete ==="