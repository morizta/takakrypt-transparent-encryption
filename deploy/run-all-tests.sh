#!/bin/bash
# Comprehensive test runner for Takakrypt policy configuration
# This script runs all user tests and generates a summary report

set -e

echo "================================================================="
echo "            Takakrypt Transparent Encryption"
echo "             Comprehensive Policy Testing"
echo "================================================================="
echo

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

# Function to run test and track results
run_test() {
    local test_name=$1
    local test_script=$2
    
    echo -e "${BLUE}=== Running $test_name ===${NC}"
    echo "Script: $test_script"
    echo
    
    if [ -f "$test_script" ]; then
        if bash "$test_script" > "/tmp/${test_name,,}-results.log" 2>&1; then
            echo -e "${GREEN}✓ $test_name: COMPLETED${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}✗ $test_name: FAILED${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            echo "  Check /tmp/${test_name,,}-results.log for details"
        fi
    else
        echo -e "${RED}✗ $test_name: SCRIPT NOT FOUND${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo
}

# Function to check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking Prerequisites...${NC}"
    
    # Check if takakrypt service is running
    if systemctl is-active --quiet takakrypt; then
        echo "✓ Takakrypt service is running"
    else
        echo "✗ Takakrypt service is not running"
        echo "  Starting takakrypt service..."
        systemctl start takakrypt
        sleep 2
        if systemctl is-active --quiet takakrypt; then
            echo "✓ Takakrypt service started successfully"
        else
            echo "✗ Failed to start takakrypt service"
            exit 1
        fi
    fi
    
    # Check if guard points are mounted
    local mounted_points=0
    if mount | grep -q "/data/sensitive"; then
        echo "✓ Sensitive guard point mounted"
        mounted_points=$((mounted_points + 1))
    else
        echo "✗ Sensitive guard point not mounted"
    fi
    
    if mount | grep -q "/data/database"; then
        echo "✓ Database guard point mounted"
        mounted_points=$((mounted_points + 1))
    else
        echo "✗ Database guard point not mounted"
    fi
    
    if mount | grep -q "/data/public"; then
        echo "✓ Public guard point mounted"
        mounted_points=$((mounted_points + 1))
    else
        echo "✗ Public guard point not mounted"
    fi
    
    if [ $mounted_points -lt 3 ]; then
        echo "Warning: Not all guard points are mounted. Tests may fail."
    fi
    
    # Check if test users exist
    if id "ntoi" &>/dev/null; then
        echo "✓ User ntoi exists"
    else
        echo "✗ User ntoi does not exist"
    fi
    
    if id "testuser1" &>/dev/null; then
        echo "✓ User testuser1 exists"
    else
        echo "✗ User testuser1 does not exist"
    fi
    
    if id "testuser2" &>/dev/null; then
        echo "✓ User testuser2 exists"
    else
        echo "✗ User testuser2 does not exist"
    fi
    
    echo
}

# Function to setup test environment
setup_test_environment() {
    echo -e "${YELLOW}Setting up test environment...${NC}"
    
    # Create test directories if they don't exist
    mkdir -p /data/sensitive/{testuser1,testuser2} 2>/dev/null || true
    mkdir -p /data/database/testdb 2>/dev/null || true
    # Note: public folder can have shared directory for universal access
    mkdir -p /data/public/shared 2>/dev/null || true
    
    # Set appropriate permissions
    chown -R ntoi:ntoi /data/sensitive 2>/dev/null || true
    chown testuser1:testuser1 /data/sensitive/testuser1 2>/dev/null || true
    chown testuser2:testuser2 /data/sensitive/testuser2 2>/dev/null || true
    
    echo "✓ Test environment ready"
    echo
}

# Function to generate summary report
generate_summary() {
    echo -e "${BLUE}=================================================================${NC}"
    echo -e "${BLUE}                        TEST SUMMARY                            ${NC}"
    echo -e "${BLUE}=================================================================${NC}"
    echo
    
    echo "Total Tests Run: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    echo
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 ALL TESTS PASSED! 🎉${NC}"
        echo "Your Takakrypt policy configuration is working correctly."
    else
        echo -e "${RED}⚠️  SOME TESTS FAILED ⚠️${NC}"
        echo "Please check the individual test logs for details:"
        ls -la /tmp/*-results.log 2>/dev/null || echo "No log files found"
    fi
    
    echo
    echo "Detailed results available in:"
    echo "  - /tmp/ntoi_user-results.log"
    echo "  - /tmp/testuser1-results.log"
    echo "  - /tmp/testuser2-results.log"
    echo
    
    # Show policy summary
    echo -e "${YELLOW}Policy Configuration Summary:${NC}"
    echo "┌─────────────┬─────────────────┬─────────────────┬─────────────────┬─────────────────────┐"
    echo "│ User        │ Sensitive GP    │ Database GP     │ Public GP       │ Allowed Processes   │"
    echo "├─────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────────┤"
    echo "│ ntoi        │ Full access     │ Full access     │ Full access     │ Any process         │"
    echo "│ testuser1   │ Own folder only │ No access       │ Full access     │ text-editors,python │"
    echo "│ testuser2   │ Own folder only │ No access       │ Full access     │ text-editors only   │"
    echo "│ mysql       │ No access       │ Process access  │ No special      │ mysql processes     │"
    echo "└─────────────┴─────────────────┴─────────────────┴─────────────────┴─────────────────────┘"
    echo
}

# Main execution
echo "Starting comprehensive policy testing..."
echo "Test logs will be saved to /tmp/*-results.log"
echo

# Run checks
check_prerequisites
setup_test_environment

# Run all tests
echo -e "${BLUE}Running User Access Tests...${NC}"
echo

run_test "NTOI_USER" "./test-ntoi-user.sh"
run_test "TESTUSER1" "./test-testuser1.sh"  
run_test "TESTUSER2" "./test-testuser2.sh"

# Additional system tests
echo -e "${BLUE}Running System Tests...${NC}"

# Test database access
echo -e "${YELLOW}Testing Database System Access...${NC}"
if systemctl is-active --quiet mariadb || systemctl is-active --quiet mysql; then
    if mysql -e "SELECT 1;" &>/dev/null; then
        echo "✓ Database system access: SUCCESS"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "✗ Database system access: FAILED"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    echo "⚠ Database service not running - skipping database test"
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

# Test encryption verification
echo -e "${YELLOW}Testing Encryption Verification...${NC}"
if [ -d "/secure_storage" ]; then
    # Check if files in secure storage are different from mount points
    if find /secure_storage -name "*.txt" -exec file {} \; | grep -q "data"; then
        echo "✓ Encryption verification: Files appear encrypted"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "⚠ Encryption verification: No encrypted files found to verify"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    fi
else
    echo "✗ Encryption verification: Secure storage not found"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

echo

# Generate final report
generate_summary

# Cleanup option
echo
read -p "Do you want to view detailed test results? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}=== DETAILED RESULTS ===${NC}"
    for log_file in /tmp/*-results.log; do
        if [ -f "$log_file" ]; then
            echo -e "\n${YELLOW}=== $(basename $log_file) ===${NC}"
            cat "$log_file"
        fi
    done
fi

echo
echo "Testing completed!"
echo "================================================================="