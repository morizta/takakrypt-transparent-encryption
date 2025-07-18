#!/bin/bash

echo "=== Complete FUSE Filesystem Test ==="
echo "Testing enhanced transparent encryption with database operations"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test mount point
MOUNT_POINT="/data/sensitive"

# Function to print test header
test_header() {
    echo ""
    echo -e "${YELLOW}>>> $1${NC}"
    echo "=================================="
}

# Function to check result
check_result() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $1${NC}"
    else
        echo -e "${RED}✗ $1${NC}"
    fi
}

# 1. Test basic file operations
test_header "Testing basic file operations"
echo "Testing file created from scratch" > $MOUNT_POINT/basic-test.txt
check_result "Created file"

cat $MOUNT_POINT/basic-test.txt
check_result "Read file"

# 2. Test text editor operations
test_header "Testing text editor operations (nano)"
echo "Initial content" > $MOUNT_POINT/editor-test.txt

# Use printf to simulate editor-like behavior
printf "Line 1\nLine 2\nLine 3\n" > $MOUNT_POINT/editor-test.txt
sync
check_result "Written multiple lines"

# Append to file
echo "Line 4" >> $MOUNT_POINT/editor-test.txt
check_result "Appended to file"

# 3. Test truncation (important for databases)
test_header "Testing truncation operations"
echo "This is a long line that will be truncated" > $MOUNT_POINT/truncate-test.txt
truncate -s 10 $MOUNT_POINT/truncate-test.txt
check_result "Truncated file to 10 bytes"

ls -la $MOUNT_POINT/truncate-test.txt
wc -c $MOUNT_POINT/truncate-test.txt

# 4. Test database-like operations
test_header "Testing database-like operations"

# Create a simulated database file
DB_FILE="$MOUNT_POINT/test.db"
dd if=/dev/zero of=$DB_FILE bs=1024 count=10 2>/dev/null
check_result "Created database file"

# Write at specific offset (simulate database page write)
echo "PAGE1" | dd of=$DB_FILE bs=1 seek=0 conv=notrunc 2>/dev/null
echo "PAGE2" | dd of=$DB_FILE bs=1 seek=512 conv=notrunc 2>/dev/null
check_result "Written to specific offsets"

# 5. Test file renaming (atomic operations)
test_header "Testing rename operations"
echo "Original file" > $MOUNT_POINT/rename-old.txt
mv $MOUNT_POINT/rename-old.txt $MOUNT_POINT/rename-new.txt
check_result "Renamed file"

# Verify content after rename
cat $MOUNT_POINT/rename-new.txt
check_result "Content preserved after rename"

# 6. Test concurrent operations
test_header "Testing concurrent operations"
# Create multiple files in parallel
for i in {1..5}; do
    (echo "Concurrent write $i" > $MOUNT_POINT/concurrent-$i.txt) &
done
wait
check_result "Created files concurrently"

ls -la $MOUNT_POINT/concurrent-*.txt

# 7. Test sync operations
test_header "Testing sync operations"
echo "Testing sync" > $MOUNT_POINT/sync-test.txt
sync $MOUNT_POINT/sync-test.txt
check_result "Synced file"

# 8. Test with actual database commands (if available)
test_header "Testing with SQLite (if available)"
if command -v sqlite3 &> /dev/null; then
    sqlite3 $MOUNT_POINT/test.sqlite "CREATE TABLE test (id INTEGER PRIMARY KEY, data TEXT);"
    sqlite3 $MOUNT_POINT/test.sqlite "INSERT INTO test (data) VALUES ('encrypted data');"
    sqlite3 $MOUNT_POINT/test.sqlite "SELECT * FROM test;"
    check_result "SQLite operations"
else
    echo "SQLite not installed, skipping database test"
fi

# 9. Test file locking
test_header "Testing file locking"
# Create a lock file
exec 200>$MOUNT_POINT/test.lock
flock -n 200
check_result "Acquired file lock"
flock -u 200
exec 200>&-

# 10. Monitor FUSE logs during tests
test_header "Recent FUSE operation logs"
echo "Checking last 20 lines of agent logs..."
sudo journalctl -u takakrypt -n 20 --no-pager | grep -E "\[FUSE\]|\[CRYPTO\]" | tail -10

# Summary
test_header "Test Summary"
echo "Enhanced FUSE filesystem test completed!"
echo ""
echo "Key operations tested:"
echo "- Basic read/write operations"
echo "- Text editor compatibility"
echo "- File truncation (database requirement)"
echo "- Database-like operations with offsets"
echo "- Atomic rename operations"
echo "- Concurrent file operations"
echo "- File synchronization"
echo "- File locking mechanisms"
echo ""
echo "Check the logs for detailed FUSE operation traces"