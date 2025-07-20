# Takakrypt Deployment Guide

## 1. Pre-Deployment Planning

### 1.1 System Assessment

**Hardware Requirements Check:**
```bash
# Check CPU architecture
uname -m  # Should show x86_64

# Check available memory
free -h  # Minimum 512MB, recommended 2GB

# Check disk space
df -h    # Minimum 1GB available

# Check for AES-NI support (recommended)
grep -m1 -o aes /proc/cpuinfo
```

**Operating System Compatibility:**
```bash
# Check Linux version
uname -r  # Should be 2.6.14 or higher

# Check FUSE support
modprobe fuse
lsmod | grep fuse

# Check systemd availability
systemctl --version
```

### 1.2 Network and Security Planning

**Firewall Configuration:**
- No specific ports required for standalone deployment
- Consider monitoring ports for future management interface
- Ensure local file system access is not restricted

**Security Considerations:**
- Plan encrypted storage location
- Consider backup strategy for keys
- Plan user/process access patterns
- Document security policies

## 2. Installation Process

### 2.1 Package Installation

**Install Dependencies:**
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y libfuse-dev build-essential

# CentOS/RHEL
sudo yum install -y fuse-devel gcc make
# or for newer versions:
sudo dnf install -y fuse-devel gcc make
```

**Install Go (for building from source):**
```bash
# Download and install Go 1.19+
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### 2.2 Build and Install Takakrypt

**Build from Source:**
```bash
# Clone repository
git clone https://github.com/morizta/takakrypt-transparent-encryption.git
cd takakrypt-transparent-encryption

# Build the agent
go build -o takakrypt-agent cmd/agent/main.go

# Install binary
sudo mkdir -p /opt/takakrypt
sudo cp takakrypt-agent /opt/takakrypt/
sudo chmod +x /opt/takakrypt/takakrypt-agent
```

**Install Configuration:**
```bash
# Create configuration directory
sudo mkdir -p /opt/takakrypt/config

# Copy configuration files
sudo cp deploy/ubuntu-config/*.json /opt/takakrypt/config/
sudo chown -R root:root /opt/takakrypt/config
sudo chmod 600 /opt/takakrypt/config/keys.json
sudo chmod 644 /opt/takakrypt/config/*.json
```

### 2.3 Create systemd Service

**Create Service File:**
```bash
sudo tee /etc/systemd/system/takakrypt.service > /dev/null <<EOF
[Unit]
Description=Takakrypt Transparent Encryption Agent
After=network.target
Requires=network.target

[Service]
Type=simple
ExecStart=/opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --log-level info
Restart=always
RestartSec=5
User=root
Group=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

**Enable and Start Service:**
```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable takakrypt

# Start service
sudo systemctl start takakrypt

# Check status
sudo systemctl status takakrypt
```

## 3. Configuration Setup

### 3.1 Guard Point Configuration

**Edit Guard Points (`/opt/takakrypt/config/guard-point.json`):**
```json
[
  {
    "id": "gp-sensitive",
    "name": "Sensitive Data Guard Point",
    "protected_path": "/data/sensitive",
    "secure_storage_path": "/secure_storage/sensitive",
    "policy_id": "policy-sensitive-data",
    "key_id": "key-sensitive-data-001",
    "status": "active"
  },
  {
    "id": "gp-database",
    "name": "Database Guard Point",
    "protected_path": "/data/database",
    "secure_storage_path": "/secure_storage/database",
    "policy_id": "policy-database-access",
    "key_id": "key-database-001",
    "status": "active"
  }
]
```

**Create Mount Directories:**
```bash
# Create protected directories
sudo mkdir -p /data/sensitive
sudo mkdir -p /data/database
sudo mkdir -p /data/public

# Create backing storage directories
sudo mkdir -p /secure_storage/sensitive
sudo mkdir -p /secure_storage/database
sudo mkdir -p /secure_storage/public

# Set permissions
sudo chmod 755 /data/*
sudo chmod 755 /secure_storage/*
```

### 3.2 Key Management Setup

**Generate Encryption Keys:**
```bash
# Generate keys for each guard point
cd takakrypt-transparent-encryption
go run cmd/keygen/main.go -output /opt/takakrypt/config/keys.json

# Secure key file permissions
sudo chmod 600 /opt/takakrypt/config/keys.json
```

**Verify Key Configuration:**
```bash
# Check key file syntax
sudo cat /opt/takakrypt/config/keys.json | jq '.'

# Verify key mappings
sudo grep -A 5 -B 5 "guard_point_id" /opt/takakrypt/config/keys.json
```

### 3.3 User and Process Set Configuration

**Configure User Sets (`/opt/takakrypt/config/user_set.json`):**
```json
[
  {
    "id": "user-set-admin",
    "code": "admin-set",
    "name": "System Administrators",
    "description": "System administrators with full access",
    "users": [
      {
        "id": "admin-user",
        "uid": 0,
        "uname": "root",
        "fname": "Root User",
        "gname": "root",
        "gid": 0,
        "os": "linux",
        "type": "system"
      }
    ]
  },
  {
    "id": "user-set-appuser",
    "code": "appuser-set",
    "name": "Application Users",
    "description": "Users running applications",
    "users": [
      {
        "id": "appuser-id",
        "uid": 1000,
        "uname": "appuser",
        "fname": "Application User",
        "gname": "appuser",
        "gid": 1000,
        "os": "linux",
        "type": "application"
      }
    ]
  }
]
```

**Configure Process Sets (`/opt/takakrypt/config/process_set.json`):**
```json
[
  {
    "id": "process-set-database",
    "code": "database-processes",
    "name": "Database Processes",
    "description": "Authorized database processes",
    "resource_set_list": [
      {
        "id": "mysql-server",
        "directory": "/usr/sbin/",
        "file": "mysqld"
      },
      {
        "id": "postgres-server",
        "directory": "/usr/lib/postgresql/",
        "file": "postgres"
      }
    ]
  },
  {
    "id": "process-set-apps",
    "code": "application-processes",
    "name": "Application Processes",
    "description": "Authorized application processes",
    "resource_set_list": [
      {
        "id": "python-app",
        "directory": "/usr/bin/",
        "file": "python3"
      },
      {
        "id": "java-app",
        "directory": "/usr/bin/",
        "file": "java"
      }
    ]
  }
]
```

### 3.4 Policy Configuration

**Configure Security Policies (`/opt/takakrypt/config/policy.json`):**
```json
[
  {
    "id": "policy-sensitive-data",
    "code": "policy-sensitive-data",
    "name": "Sensitive Data Access Policy",
    "policy_type": "life_data_transformation",
    "description": "Controls access to sensitive files with encryption",
    "security_rules": [
      {
        "id": "rule-admin-full",
        "order": 1,
        "resource_set": ["sensitive-data"],
        "user_set": ["admin-set"],
        "process_set": [],
        "action": ["read", "write", "all_ops"],
        "browsing": true,
        "effect": {
          "permission": "permit",
          "option": {
            "apply_key": true,
            "audit": true
          }
        }
      },
      {
        "id": "rule-app-controlled",
        "order": 2,
        "resource_set": ["sensitive-data"],
        "user_set": ["appuser-set"],
        "process_set": ["application-processes"],
        "action": ["read", "write"],
        "browsing": true,
        "effect": {
          "permission": "permit",
          "option": {
            "apply_key": true,
            "audit": true
          }
        }
      },
      {
        "id": "rule-deny-all",
        "order": 99,
        "resource_set": [],
        "user_set": [],
        "process_set": [],
        "action": ["read", "write", "all_ops"],
        "browsing": true,
        "effect": {
          "permission": "deny",
          "option": {
            "apply_key": false,
            "audit": true
          }
        }
      }
    ]
  }
]
```

## 4. Service Management

### 4.1 Service Operations

**Start/Stop Service:**
```bash
# Start service
sudo systemctl start takakrypt

# Stop service
sudo systemctl stop takakrypt

# Restart service
sudo systemctl restart takakrypt

# Check status
sudo systemctl status takakrypt
```

**Configuration Reload:**
```bash
# Reload configuration without restart
sudo systemctl reload takakrypt

# Or send SIGHUP signal
sudo pkill -HUP takakrypt-agent
```

### 4.2 Log Management

**View Logs:**
```bash
# Follow logs in real-time
sudo journalctl -u takakrypt -f

# View recent logs
sudo journalctl -u takakrypt -n 100

# View logs for specific time period
sudo journalctl -u takakrypt --since "2024-01-01" --until "2024-01-02"
```

**Log Rotation:**
```bash
# Configure log rotation
sudo tee /etc/logrotate.d/takakrypt > /dev/null <<EOF
/var/log/takakrypt.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 root root
    postrotate
        systemctl reload takakrypt
    endscript
}
EOF
```

### 4.3 Monitoring and Health Checks

**Health Check Script:**
```bash
#!/bin/bash
# health-check.sh

# Check service status
if ! systemctl is-active --quiet takakrypt; then
    echo "ERROR: Takakrypt service is not running"
    exit 1
fi

# Check mount points
for mount in /data/sensitive /data/database /data/public; do
    if ! mountpoint -q "$mount"; then
        echo "ERROR: $mount is not mounted"
        exit 1
    fi
done

# Check configuration files
for config in /opt/takakrypt/config/*.json; do
    if ! jq . "$config" >/dev/null 2>&1; then
        echo "ERROR: Invalid JSON in $config"
        exit 1
    fi
done

echo "OK: All health checks passed"
```

## 5. Testing and Validation

### 5.1 Basic Functionality Tests

**Test Mount Points:**
```bash
# Check mount points
mount | grep takakrypt

# Test basic file operations
echo "test data" | sudo tee /data/sensitive/test.txt
sudo cat /data/sensitive/test.txt
sudo rm /data/sensitive/test.txt
```

**Test Encryption:**
```bash
# Create test file
echo "sensitive data" > /data/sensitive/encryption-test.txt

# Check encrypted storage
ls -la /secure_storage/sensitive/
sudo hexdump -C /secure_storage/sensitive/encryption-test.txt | head -5

# Verify plaintext access
cat /data/sensitive/encryption-test.txt
```

### 5.2 Access Control Tests

**Test User Access:**
```bash
# Test as authorized user
sudo -u appuser echo "app data" > /data/sensitive/app-test.txt

# Test as unauthorized user (should fail)
sudo -u nobody echo "unauthorized" > /data/sensitive/unauthorized.txt
```

**Test Process Restrictions:**
```bash
# Test authorized process
sudo -u appuser python3 -c "
with open('/data/sensitive/python-test.txt', 'w') as f:
    f.write('python access test')
"

# Test unauthorized process (should fail)
sudo -u appuser cat /data/sensitive/python-test.txt
```

### 5.3 Performance Tests

**Basic Performance Test:**
```bash
#!/bin/bash
# performance-test.sh

# Create test file
TEST_FILE="/data/sensitive/perf-test.txt"
TEST_SIZE="100M"

echo "Creating ${TEST_SIZE} test file..."
sudo dd if=/dev/zero of="$TEST_FILE" bs=1M count=100

echo "Testing read performance..."
time sudo dd if="$TEST_FILE" of=/dev/null bs=1M

echo "Testing write performance..."
time sudo dd if=/dev/zero of="$TEST_FILE" bs=1M count=100

sudo rm "$TEST_FILE"
```

## 6. Maintenance and Updates

### 6.1 Backup Procedures

**Configuration Backup:**
```bash
#!/bin/bash
# backup-config.sh

BACKUP_DIR="/opt/takakrypt/backup/$(date +%Y%m%d-%H%M%S)"
sudo mkdir -p "$BACKUP_DIR"

# Backup configuration files
sudo cp -r /opt/takakrypt/config/* "$BACKUP_DIR/"

# Backup service file
sudo cp /etc/systemd/system/takakrypt.service "$BACKUP_DIR/"

# Create archive
sudo tar -czf "$BACKUP_DIR.tar.gz" -C /opt/takakrypt/backup "$(basename "$BACKUP_DIR")"
sudo rm -rf "$BACKUP_DIR"

echo "Backup created: $BACKUP_DIR.tar.gz"
```

**Key Backup (Encrypted):**
```bash
# Encrypt and backup keys
sudo gpg --cipher-algo AES256 --compress-algo 1 --symmetric \
    --output /opt/takakrypt/backup/keys-$(date +%Y%m%d).gpg \
    /opt/takakrypt/config/keys.json
```

### 6.2 Update Procedures

**Update Agent:**
```bash
# Stop service
sudo systemctl stop takakrypt

# Backup current version
sudo cp /opt/takakrypt/takakrypt-agent /opt/takakrypt/takakrypt-agent.backup

# Install new version
sudo cp new-takakrypt-agent /opt/takakrypt/takakrypt-agent
sudo chmod +x /opt/takakrypt/takakrypt-agent

# Start service
sudo systemctl start takakrypt

# Verify operation
sudo systemctl status takakrypt
```

**Configuration Updates:**
```bash
# Backup current configuration
sudo cp /opt/takakrypt/config/policy.json /opt/takakrypt/config/policy.json.backup

# Update configuration
sudo vim /opt/takakrypt/config/policy.json

# Validate configuration
sudo jq . /opt/takakrypt/config/policy.json

# Reload configuration
sudo systemctl reload takakrypt
```

## 7. Troubleshooting

### 7.1 Common Issues

**Service Won't Start:**
```bash
# Check logs
sudo journalctl -u takakrypt -n 50

# Check configuration
sudo /opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --validate-config

# Check permissions
sudo ls -la /opt/takakrypt/config/
```

**Mount Points Not Working:**
```bash
# Check FUSE module
lsmod | grep fuse
sudo modprobe fuse

# Check mount permissions
sudo ls -la /data/
sudo ls -la /secure_storage/

# Unmount and remount
sudo umount /data/sensitive
sudo systemctl restart takakrypt
```

**Permission Denied Errors:**
```bash
# Check policy configuration
sudo jq '.[] | .security_rules[] | select(.user_set[] == "your-user-set")' /opt/takakrypt/config/policy.json

# Check user set configuration
sudo jq '.[] | select(.users[].uname == "your-username")' /opt/takakrypt/config/user_set.json

# Check process set configuration
sudo jq '.[] | .resource_set_list[] | select(.file == "your-process")' /opt/takakrypt/config/process_set.json
```

### 7.2 Debug Mode

**Enable Debug Logging:**
```bash
# Stop service
sudo systemctl stop takakrypt

# Run in debug mode
sudo /opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --log-level debug

# Or modify service file
sudo sed -i 's/--log-level info/--log-level debug/' /etc/systemd/system/takakrypt.service
sudo systemctl daemon-reload
sudo systemctl start takakrypt
```

### 7.3 Recovery Procedures

**Emergency Recovery:**
```bash
# Unmount all FUSE mounts
sudo umount /data/sensitive /data/database /data/public

# Stop service
sudo systemctl stop takakrypt

# Access backing storage directly
sudo ls -la /secure_storage/sensitive/

# Restore from backup
sudo tar -xzf /opt/takakrypt/backup/backup-date.tar.gz -C /opt/takakrypt/config/

# Restart service
sudo systemctl start takakrypt
```

## 8. Security Hardening

### 8.1 System Hardening

**File Permissions:**
```bash
# Secure configuration directory
sudo chmod 700 /opt/takakrypt/config
sudo chmod 600 /opt/takakrypt/config/keys.json
sudo chmod 644 /opt/takakrypt/config/*.json

# Secure binary
sudo chmod 755 /opt/takakrypt/takakrypt-agent
sudo chown root:root /opt/takakrypt/takakrypt-agent
```

**SELinux Configuration (if applicable):**
```bash
# Create SELinux policy for Takakrypt
sudo setsebool -P use_fusefs_home_dirs 1
sudo setsebool -P fuse_use_fusefs 1
```

### 8.2 Network Security

**Firewall Configuration:**
```bash
# No incoming ports need to be opened for basic operation
# If management interface is added, configure appropriate rules
```

**Audit Configuration:**
```bash
# Enable file access auditing
sudo auditctl -w /data/sensitive -p rwxa -k takakrypt-sensitive
sudo auditctl -w /opt/takakrypt/config -p rwxa -k takakrypt-config
```

This deployment guide provides comprehensive instructions for installing, configuring, and maintaining the Takakrypt transparent encryption system in production environments.