#!/bin/bash

echo "=== Setting up Application Access Test Environment ==="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to check result
check_result() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $1${NC}"
    else
        echo -e "${RED}✗ $1${NC}"
        return 1
    fi
}

echo -e "${YELLOW}>>> Checking Python installation${NC}"
python3 --version
check_result "Python 3 is available"

echo ""
echo -e "${YELLOW}>>> Installing Python dependencies${NC}"
pip3 install mysql-connector-python
check_result "Installed mysql-connector-python"

echo ""
echo -e "${YELLOW}>>> Making scripts executable${NC}"
chmod +x app.py
chmod +x test-app-access.sh
check_result "Made scripts executable"

echo ""
echo -e "${YELLOW}>>> Testing database connection${NC}"
python3 -c "
import mysql.connector
try:
    conn = mysql.connector.connect(host='localhost', user='root', password='')
    print('✓ Database connection successful')
    conn.close()
except Exception as e:
    print(f'✗ Database connection failed: {e}')
    exit(1)
"
check_result "Database connection test"

echo ""
echo -e "${GREEN}Setup complete! You can now run:${NC}"
echo "  ./app.py                    - Python application simulation"
echo "  ./test-app-access.sh        - Shell-based application test"
echo ""
echo -e "${YELLOW}Both tests simulate applications accessing MariaDB with transparent encryption${NC}"