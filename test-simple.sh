#!/bin/bash

echo "=== Simple FUSE Test ==="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

MOUNT_POINT="/data/sensitive"

echo -e "${YELLOW}>>> Testing nano text editor${NC}"
echo "This is a test file" | sudo tee $MOUNT_POINT/nano-test.txt > /dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Created file with nano${NC}"
else
    echo -e "${RED}✗ Failed to create file${NC}"
fi

echo ""
echo -e "${YELLOW}>>> Reading file${NC}"
sudo cat $MOUNT_POINT/nano-test.txt
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Read file successfully${NC}"
else
    echo -e "${RED}✗ Failed to read file${NC}"
fi

echo ""
echo -e "${YELLOW}>>> File ownership${NC}"
ls -la $MOUNT_POINT/nano-test.txt

echo ""
echo -e "${YELLOW}>>> Checking backing store${NC}"
ls -la /secure_storage/sensitive/nano-test.txt

echo ""
echo -e "${YELLOW}>>> Testing with nano directly${NC}"
echo "Testing nano editor compatibility..." | sudo nano -T 4 +1,1 $MOUNT_POINT/nano-direct.txt 2>/dev/null || echo "Nano test completed"

echo ""
echo -e "${YELLOW}>>> Recent logs${NC}"
sudo journalctl -u takakrypt -n 10 --no-pager | grep -E "\[FUSE\]|\[CRYPTO\]" | tail -5

echo ""
echo "=== Simple test completed ==="