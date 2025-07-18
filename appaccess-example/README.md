# Application Access Examples

This directory contains examples demonstrating how real applications access MariaDB with transparent encryption enabled.

## Overview

These examples simulate the complete application stack:
```
Application → MariaDB → Transparent Encryption → Encrypted Storage
```

## Files

- **`app.py`** - Python application simulation (customer management system)
- **`test-app-access.sh`** - Shell-based application test (e-commerce simulation)  
- **`node-app.js`** - Node.js API application example
- **`setup.sh`** - Environment setup script
- **`requirements.txt`** - Python dependencies

## Quick Start

1. **Setup environment:**
   ```bash
   cd appaccess-example
   chmod +x setup.sh
   ./setup.sh
   ```

2. **Run Python application test:**
   ```bash
   python3 app.py
   ```

3. **Run shell-based test:**
   ```bash
   ./test-app-access.sh
   ```

## What These Tests Demonstrate

### Python Application (`app.py`)
- **Customer Management System**
- User registration with sensitive data (SSNs, credit cards)
- Order processing and e-commerce transactions
- Data analytics and reporting
- Bulk operations and concurrent access
- Search functionality

### Shell Application (`test-app-access.sh`)  
- **E-commerce Platform Simulation**
- Web session management
- User authentication data
- Transaction processing
- Performance testing
- Data encryption verification

### Node.js API (`node-app.js`)
- **REST API Application**
- Session-based authentication
- Product catalog management
- Transaction processing with payment data
- User analytics
- Concurrent operations

## Key Test Scenarios

1. **Application Registration**: Applications store sensitive user data (SSNs, credit cards, addresses)

2. **Session Management**: Web applications manage user sessions with personal information

3. **Transaction Processing**: E-commerce applications process payments and orders

4. **Data Analytics**: Applications query and analyze sensitive customer data

5. **Bulk Operations**: High-volume applications perform batch operations

6. **Concurrent Access**: Multiple application instances access the database simultaneously

## Verification Points

✅ **Transparent Operation**: Applications work normally without knowing about encryption  
✅ **Data Protection**: Sensitive data is encrypted in the filesystem  
✅ **Performance**: Application performance remains acceptable  
✅ **Concurrency**: Multiple applications can access data concurrently  
✅ **Integrity**: Data remains consistent and accessible  

## Expected Results

- Applications can read/write data normally
- Sensitive data (SSNs, credit cards, addresses) is stored and retrieved correctly
- Raw database files contain encrypted data, not plaintext
- Application performance is maintained
- Multiple applications can operate concurrently

This demonstrates that the transparent encryption system works with real-world application scenarios, not just direct database access.