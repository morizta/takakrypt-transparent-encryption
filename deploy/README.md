# Ubuntu VM User Trial Setup

## Overview
This setup creates a comprehensive test environment for the Takakrypt Transparent Encryption Agent on Ubuntu, simulating real-world scenarios with databases and multiple users.

## Setup Steps

### 1. Initial VM Setup
```bash
# Transfer setup script to Ubuntu VM
scp ubuntu-setup.sh user@vm-ip:/tmp/

# Run setup on Ubuntu VM
ssh user@vm-ip
sudo chmod +x /tmp/ubuntu-setup.sh
sudo /tmp/ubuntu-setup.sh
```

### 2. Transfer Agent Code
```bash
# From your development machine
scp -r ../. user@vm-ip:/tmp/takakrypt-src

# Or clone from git if available
ssh user@vm-ip
cd /tmp
git clone <your-repo> takakrypt-src
```

### 3. Deploy Agent
```bash
# On Ubuntu VM
cd /tmp/takakrypt-src/deploy
sudo chmod +x *.sh
sudo ./deploy-agent.sh
```

### 4. Start and Test
```bash
# Start the agent
sudo /opt/takakrypt/start.sh

# Run comprehensive tests
sudo /opt/takakrypt/test-scenarios.sh
```

## Test Environment

### Users Created
- **testuser1** (UID: 1001) - Database access user
  - Can read sensitive files
  - Can read database files  
  - Cannot write to sensitive areas
  
- **testuser2** (UID: 1002) - Limited access user
  - Can only read public files
  - Denied access to sensitive/database files
  
- **dbadmin** (UID: 1003) - Administrative user
  - Full access to all files
  - Can read/write sensitive data
  - Database administration privileges

### Database Setup
- **MariaDB** installed and configured
- **testapp** database created
- Sample tables: `users`, `sensitive_data`
- Application user: `appuser` / `apppass123`

### Guard Points
1. **Sensitive Data** (`/data/sensitive` → `/secure_storage/sensitive`)
   - Confidential documents
   - Encrypted storage
   - Restricted access
   
2. **Database Files** (`/data/database` → `/secure_storage/database`)
   - Database backups
   - SQL files
   - Admin and authorized user access
   
3. **Public Data** (`/data/public` → `/secure_storage/public`)
   - Open access files
   - No encryption for public data
   - Read access for all users

## Policy Configuration

### Security Rules
1. **Sensitive Data Policy**
   - dbadmin: Full access with encryption
   - testuser1: Read-only with encryption
   - Others: Denied

2. **Database Access Policy**
   - dbadmin: Full access
   - MySQL processes: Read/write access
   - testuser1: Read-only access
   - Others: Denied

3. **Public Access Policy**
   - All users: Read access
   - dbadmin: Write access
   - No encryption for public files

## Test Scenarios

### Scenario 1: Database Admin Workflow
```bash
# Switch to dbadmin
sudo su - dbadmin

# Access sensitive files
cat /data/sensitive/confidential.txt
echo "Admin review $(date)" > /data/sensitive/review.txt

# Manage database backups
cat /data/database/backup.sql
cp /data/database/backup.sql /data/database/backup_$(date +%Y%m%d).sql
```

### Scenario 2: Regular User Access
```bash
# Switch to testuser1
sudo su - testuser1

# Read allowed files
cat /data/sensitive/confidential.txt  # Should work
cat /data/database/backup.sql         # Should work

# Try to write (should fail)
echo "user note" > /data/sensitive/note.txt  # Should be denied
```

### Scenario 3: Restricted User
```bash
# Switch to testuser2
sudo su - testuser2

# Only public access should work
cat /data/public/readme.txt           # Should work
cat /data/sensitive/confidential.txt  # Should be denied
```

### Scenario 4: Database Application Access
```bash
# Test database connectivity
mysql -u appuser -papppass123 testapp -e "SELECT * FROM users;"

# Access sensitive data through application
mysql -u appuser -papppass123 testapp -e "SELECT * FROM sensitive_data;"
```

## Monitoring and Verification

### Check Agent Status
```bash
sudo /opt/takakrypt/status.sh
```

### View Logs
```bash
sudo journalctl -u takakrypt -f
```

### Verify Encryption
```bash
# Check raw encrypted files
sudo ls -la /secure_storage/sensitive/
sudo xxd /secure_storage/sensitive/confidential.txt | head
```

### Test Mount Points
```bash
mount | grep takakrypt
df -h | grep data
```

## Expected Results

### Successful Tests
- ✅ FUSE mounts active on all guard points
- ✅ Policy enforcement working correctly
- ✅ Files encrypted in secure storage
- ✅ User access controls enforced
- ✅ Database applications can access data
- ✅ Audit logging captures all access

### Verification Points
1. **Access Control**: Users can only access files per policy
2. **Encryption**: Files stored encrypted in `/secure_storage/`
3. **Transparency**: Applications work normally through FUSE
4. **Performance**: File operations responsive
5. **Auditability**: All access logged and tracked

## Troubleshooting

### Common Issues
1. **FUSE group doesn't exist**: Fixed in updated setup script (creates group automatically)
2. **FUSE mount fails**: Check FUSE permissions and user_allow_other
3. **Permission denied**: Verify user UIDs match configuration
4. **Service won't start**: Check logs with `journalctl -u takakrypt`
5. **Database connection issues**: Verify MariaDB is running

### FUSE Setup Verification
Before deploying the agent, test FUSE setup:
```bash
sudo ./test-fuse-setup.sh
```

This will check:
- FUSE kernel module loaded
- /dev/fuse device exists  
- fusermount utilities available
- FUSE group and permissions
- Configuration files

### Debug Commands
```bash
# Check FUSE
ls -la /proc/filesystems | grep fuse

# Check service
systemctl status takakrypt

# Check mounts
mount | grep fuse
lsof | grep /data

# Test permissions
sudo -u testuser1 ls -la /data/sensitive/
```

This setup provides a comprehensive test environment for evaluating the Takakrypt Transparent Encryption Agent in realistic scenarios with multiple users, database applications, and varying security requirements.