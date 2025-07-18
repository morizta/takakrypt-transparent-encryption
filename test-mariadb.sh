#!/bin/bash

echo "=== MariaDB/MySQL Transparent Encryption Test ==="
echo "Testing database operations with transparent encryption"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
MOUNT_POINT="/data/sensitive"
DB_DIR="$MOUNT_POINT/mysql"
TEST_DB="takakrypt_test"

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

# Check if MariaDB/MySQL is installed
if ! command -v mysql &> /dev/null; then
    echo -e "${RED}MariaDB/MySQL is not installed${NC}"
    echo "Install with: sudo apt-get install mariadb-server mariadb-client"
    exit 1
fi

# 1. Prepare database directory
test_header "Preparing database directory"
sudo mkdir -p $DB_DIR
sudo chown mysql:mysql $DB_DIR
check_result "Created database directory"

# 2. Create test database
test_header "Creating test database"
sudo mysql -e "CREATE DATABASE IF NOT EXISTS $TEST_DB;"
check_result "Created database"

# 3. Create a table with data in the encrypted mount
test_header "Creating table in encrypted storage"
sudo mysql $TEST_DB -e "
CREATE TABLE IF NOT EXISTS encrypted_data (
    id INT PRIMARY KEY AUTO_INCREMENT,
    sensitive_info VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;
"
check_result "Created table"

# 4. Insert test data
test_header "Inserting sensitive data"
for i in {1..10}; do
    sudo mysql $TEST_DB -e "INSERT INTO encrypted_data (sensitive_info) VALUES ('Secret data $i - SSN: 123-45-678$i');"
done
check_result "Inserted test data"

# 5. Query data to test read operations
test_header "Reading encrypted data"
sudo mysql $TEST_DB -e "SELECT * FROM encrypted_data LIMIT 5;"
check_result "Read data successfully"

# 6. Update operations
test_header "Testing update operations"
sudo mysql $TEST_DB -e "UPDATE encrypted_data SET sensitive_info = 'Updated secret' WHERE id = 1;"
check_result "Updated data"

# 7. Test transactions
test_header "Testing transactions"
sudo mysql $TEST_DB -e "
START TRANSACTION;
INSERT INTO encrypted_data (sensitive_info) VALUES ('Transaction test 1');
INSERT INTO encrypted_data (sensitive_info) VALUES ('Transaction test 2');
COMMIT;
"
check_result "Transaction completed"

# 8. Test concurrent access
test_header "Testing concurrent database access"
for i in {1..3}; do
    (sudo mysql $TEST_DB -e "INSERT INTO encrypted_data (sensitive_info) VALUES ('Concurrent insert $i');") &
done
wait
check_result "Concurrent inserts completed"

# 9. Check FUSE operations during database activity
test_header "Recent FUSE operations for database"
sudo journalctl -u takakrypt -n 30 --no-pager | grep -E "\[FUSE\].*\.(db|ibd|frm)" | tail -10

# 10. Verify encryption is working
test_header "Verifying encryption"
# Try to read raw file (should see encrypted content)
echo "Attempting to read raw database file as unauthorized user..."
sudo -u nobody cat $MOUNT_POINT/mysql/*.ibd 2>&1 | head -c 100 | od -c

# 11. Test database backup/restore
test_header "Testing backup operations"
sudo mysqldump $TEST_DB > $MOUNT_POINT/backup.sql
check_result "Database backup created"

# Check if backup is encrypted
ls -la $MOUNT_POINT/backup.sql

# 12. Cleanup
test_header "Cleanup"
sudo mysql -e "DROP DATABASE IF EXISTS $TEST_DB;"
check_result "Dropped test database"

# Summary
test_header "MariaDB Test Summary"
echo "Database operations tested:"
echo "- Table creation in encrypted storage"
echo "- Insert operations"
echo "- Select/read operations"
echo "- Update operations"
echo "- Transactions"
echo "- Concurrent access"
echo "- Backup operations"
echo ""
echo "All database operations should work transparently with encryption!"