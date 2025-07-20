# Takakrypt Troubleshooting Guide

## Overview

This guide provides comprehensive troubleshooting information for the Takakrypt transparent encryption system. It covers common issues, diagnostic procedures, and resolution steps.

## Quick Diagnostic Commands

### System Status Check
```bash
# Check service status
sudo systemctl status takakrypt

# Check mount points
mount | grep takakrypt

# Check logs
sudo journalctl -u takakrypt -n 50 --no-pager

# Check configuration syntax
for config in /opt/takakrypt/config/*.json; do
    echo "Checking $config"
    jq . "$config" > /dev/null && echo "✓ Valid" || echo "✗ Invalid"
done

# Check file permissions
ls -la /opt/takakrypt/config/
```

### Basic Functionality Test
```bash
# Test basic file operations
echo "test" | sudo tee /data/sensitive/diagnostic-test.txt
cat /data/sensitive/diagnostic-test.txt
rm /data/sensitive/diagnostic-test.txt
```

## Common Issues and Solutions

### 1. Service Won't Start

#### Symptoms
- `systemctl status takakrypt` shows failed state
- Error messages in systemd logs
- FUSE mounts not available

#### Diagnostic Steps
```bash
# Check detailed service logs
sudo journalctl -u takakrypt -n 100 --no-pager

# Check configuration validity
sudo /opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --validate-config

# Check binary permissions
ls -la /opt/takakrypt/takakrypt-agent

# Check FUSE availability
lsmod | grep fuse
```

#### Common Causes and Solutions

**Configuration Syntax Error**:
```bash
# Check for JSON syntax errors
jq . /opt/takakrypt/config/policy.json
# Error: Invalid JSON syntax

# Solution: Fix JSON syntax
sudo vim /opt/takakrypt/config/policy.json
```

**Missing Dependencies**:
```bash
# Check FUSE module
sudo modprobe fuse
lsmod | grep fuse

# Install FUSE if missing
sudo apt-get install fuse libfuse-dev
```

**Permission Issues**:
```bash
# Fix binary permissions
sudo chmod +x /opt/takakrypt/takakrypt-agent
sudo chown root:root /opt/takakrypt/takakrypt-agent

# Fix configuration permissions
sudo chmod 600 /opt/takakrypt/config/keys.json
sudo chmod 644 /opt/takakrypt/config/*.json
sudo chown -R root:root /opt/takakrypt/config/
```

**Configuration Cross-Reference Errors**:
```bash
# Check for invalid references
sudo journalctl -u takakrypt | grep -i "not found\|invalid.*reference"

# Common issues:
# - Guard point references non-existent policy
# - Policy references non-existent user set
# - Guard point references non-existent key
```

### 2. FUSE Mount Issues

#### Symptoms
- Mount points not visible
- "Transport endpoint is not connected" errors
- Permission denied accessing mount points

#### Diagnostic Steps
```bash
# Check mount status
mount | grep takakrypt
df -h | grep takakrypt

# Check for stale mounts
sudo umount /data/sensitive 2>&1 | grep -v "not mounted"

# Check FUSE process
ps aux | grep takakrypt

# Check mount point permissions
ls -ld /data/sensitive /data/database /data/public
```

#### Solutions

**Stale Mount Points**:
```bash
# Clean up stale mounts
sudo umount /data/sensitive /data/database /data/public 2>/dev/null
sudo pkill -f takakrypt-agent
sudo systemctl restart takakrypt
```

**Mount Point Creation**:
```bash
# Create mount directories if missing
sudo mkdir -p /data/sensitive /data/database /data/public
sudo mkdir -p /secure_storage/sensitive /secure_storage/database /secure_storage/public
```

**FUSE Permissions**:
```bash
# Check FUSE user permissions
grep fuse /etc/group
sudo usermod -a -G fuse $(whoami)

# Add FUSE mount options
echo "user_allow_other" | sudo tee -a /etc/fuse.conf
```

### 3. Permission Denied Errors

#### Symptoms
- Applications can't access files in protected directories
- "Permission denied" when reading/writing files
- Files appear but are not accessible

#### Diagnostic Steps
```bash
# Check file ownership and permissions
ls -la /data/sensitive/problematic-file.txt
ls -la /secure_storage/sensitive/problematic-file.txt

# Check policy evaluation logs
sudo journalctl -u takakrypt | grep -E "POLICY.*deny|permission.*denied"

# Check user and process context
id $(whoami)
ps -o pid,user,cmd -p $$
```

#### Solutions

**User Not in User Set**:
```bash
# Check if user is in user sets
sudo jq '.[] | select(.users[].uname == "your-username")' /opt/takakrypt/config/user_set.json

# Add user to appropriate user set
sudo vim /opt/takakrypt/config/user_set.json
sudo systemctl reload takakrypt
```

**Process Not in Process Set**:
```bash
# Check if process is in process sets
sudo jq '.[] | .resource_set_list[] | select(.file == "python3")' /opt/takakrypt/config/process_set.json

# Add process to appropriate process set
sudo vim /opt/takakrypt/config/process_set.json
sudo systemctl reload takakrypt
```

**Policy Rule Order Issues**:
```bash
# Check policy rule evaluation order
sudo jq '.[] | .security_rules[] | {order: .order, id: .id, permission: .effect.permission}' /opt/takakrypt/config/policy.json

# Ensure permit rules have lower order than deny rules
```

### 4. Encryption/Decryption Issues

#### Symptoms
- Files show encrypted content instead of plaintext
- "Ciphertext too short" errors
- Decryption failures in logs

#### Diagnostic Steps
```bash
# Check if encryption is working
echo "test data" > /data/sensitive/encryption-test.txt
ls -la /data/sensitive/encryption-test.txt  # Should show small size
ls -la /secure_storage/sensitive/encryption-test.txt  # Should show larger size

# Check key availability
sudo jq '.[] | {id: .id, guard_point_id: .guard_point_id, status: .status}' /opt/takakrypt/config/keys.json

# Check crypto operation logs
sudo journalctl -u takakrypt | grep -E "CRYPTO|encryption|decryption"
```

#### Solutions

**Key Not Found**:
```bash
# Check guard point to key mapping
sudo jq '.[] | {id: .id, key_id: .key_id}' /opt/takakrypt/config/guard-point.json
sudo jq '.[] | {id: .id, guard_point_id: .guard_point_id}' /opt/takakrypt/config/keys.json

# Ensure key_id in guard point matches key id in keys.json
```

**Apply Key Setting**:
```bash
# Check if apply_key is set to true in policy
sudo jq '.[] | .security_rules[] | {id: .id, apply_key: .effect.option.apply_key}' /opt/takakrypt/config/policy.json

# Set apply_key to true for encryption
"apply_key": true
```

**Empty Files Issue**:
```bash
# Check for empty files (common with shell redirection)
ls -la /secure_storage/sensitive/ | grep " 0 "

# Empty files can't be decrypted - recreate with content
echo "replacement content" > /data/sensitive/fixed-file.txt
```

### 5. Performance Issues

#### Symptoms
- Slow file operations
- High CPU usage
- Application timeouts

#### Diagnostic Steps
```bash
# Check system resources
top -p $(pgrep takakrypt-agent)
iostat -x 1 5

# Check FUSE operation latency
sudo strace -p $(pgrep takakrypt-agent) -c -e trace=file

# Check encryption performance
time dd if=/dev/zero of=/data/sensitive/perf-test bs=1M count=100
time dd if=/data/sensitive/perf-test of=/dev/null bs=1M
rm /data/sensitive/perf-test
```

#### Solutions

**High Policy Evaluation Overhead**:
```bash
# Simplify complex policies
# Reduce number of rules
# Use more specific resource sets
# Optimize rule order (most common rules first)
```

**Large File Handling**:
```bash
# Implement streaming for large files
# Check available memory
free -h

# Adjust buffer sizes if needed (code modification required)
```

**Disk I/O Bottleneck**:
```bash
# Move backing storage to faster storage
# Use SSD for /secure_storage
# Check disk performance
sudo hdparm -Tt /dev/sda
```

### 6. Database Integration Issues

#### Symptoms
- Database can't access data files
- Database corruption errors
- MariaDB/MySQL startup failures

#### Diagnostic Steps
```bash
# Check database file permissions
sudo ls -la /data/database/
sudo ls -la /secure_storage/database/

# Check database error logs
sudo tail -f /var/log/mysql/error.log

# Check if database processes are in process sets
sudo jq '.[] | select(.code == "database-processes")' /opt/takakrypt/config/process_set.json
```

#### Solutions

**Database User Access**:
```bash
# Ensure mysql user is in user sets
sudo jq '.[] | .users[] | select(.uname == "mysql")' /opt/takakrypt/config/user_set.json

# Add mysql user if missing
{
  "uid": 999,
  "uname": "mysql",
  "gname": "mysql",
  "gid": 999
}
```

**Database Process Access**:
```bash
# Ensure mysqld is in process sets
sudo jq '.[] | .resource_set_list[] | select(.file == "mysqld")' /opt/takakrypt/config/process_set.json

# Add mysqld process if missing
{
  "directory": "/usr/sbin/",
  "file": "mysqld"
}
```

**File Locking Issues**:
```bash
# Check if Flock operation is working
sudo journalctl -u takakrypt | grep -i flock

# Database file locking is implemented in file.go:Flock()
```

## Advanced Diagnostics

### Debug Mode Operation

**Enable Debug Logging**:
```bash
# Stop service
sudo systemctl stop takakrypt

# Run in debug mode
sudo /opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --log-level debug

# Or modify service file for persistent debug
sudo sed -i 's/--log-level info/--log-level debug/' /etc/systemd/system/takakrypt.service
sudo systemctl daemon-reload
sudo systemctl start takakrypt
```

**Debug Log Analysis**:
```bash
# Filter by component
sudo journalctl -u takakrypt | grep "\[FUSE\]"
sudo journalctl -u takakrypt | grep "\[POLICY\]"
sudo journalctl -u takakrypt | grep "\[CRYPTO\]"

# Filter by operation
sudo journalctl -u takakrypt | grep "InterceptOpen"
sudo journalctl -u takakrypt | grep "InterceptWrite"
sudo journalctl -u takakrypt | grep "EvaluateAccess"
```

### Configuration Validation

**Comprehensive Configuration Check**:
```bash
#!/bin/bash
# comprehensive-config-check.sh

CONFIG_DIR="/opt/takakrypt/config"
ERRORS=0

echo "=== Takakrypt Configuration Validation ==="

# Check JSON syntax
for file in "$CONFIG_DIR"/*.json; do
    echo "Checking JSON syntax: $(basename "$file")"
    if ! jq . "$file" > /dev/null 2>&1; then
        echo "ERROR: Invalid JSON in $file"
        ((ERRORS++))
    fi
done

# Check file permissions
echo "Checking file permissions..."
if [[ $(stat -c "%a" "$CONFIG_DIR/keys.json") != "600" ]]; then
    echo "ERROR: keys.json should have 600 permissions"
    ((ERRORS++))
fi

# Check cross-references
echo "Checking cross-references..."

# Guard points reference valid policies
jq -r '.[].policy_id' "$CONFIG_DIR/guard-point.json" | while read policy_id; do
    if ! jq -e ".[] | select(.id == \"$policy_id\")" "$CONFIG_DIR/policy.json" > /dev/null; then
        echo "ERROR: Guard point references non-existent policy: $policy_id"
        ((ERRORS++))
    fi
done

# Guard points reference valid keys
jq -r '.[].key_id' "$CONFIG_DIR/guard-point.json" | while read key_id; do
    if ! jq -e ".[] | select(.id == \"$key_id\")" "$CONFIG_DIR/keys.json" > /dev/null; then
        echo "ERROR: Guard point references non-existent key: $key_id"
        ((ERRORS++))
    fi
done

echo "Validation complete. Errors found: $ERRORS"
exit $ERRORS
```

### Performance Monitoring

**Performance Metrics Collection**:
```bash
#!/bin/bash
# performance-monitor.sh

DURATION=60  # Monitor for 60 seconds

echo "=== Takakrypt Performance Monitoring ==="
echo "Monitoring for $DURATION seconds..."

# Start monitoring
{
    # CPU and memory usage
    top -b -n $((DURATION/5)) -d 5 -p $(pgrep takakrypt-agent) | \
    grep takakrypt-agent | \
    awk '{print strftime("%Y-%m-%d %H:%M:%S"), "CPU:", $9"%, MEM:", $10"%"}'
} &

{
    # File operation statistics
    for i in $(seq 1 $((DURATION/10))); do
        echo "$(date): File operations in last 10 seconds:"
        sudo journalctl -u takakrypt --since "10 seconds ago" | \
        grep -E "InterceptOpen|InterceptWrite|InterceptRead" | \
        wc -l
        sleep 10
    done
} &

wait
echo "Monitoring complete."
```

### Network Diagnostics (Future)

When network features are added:
```bash
# Check network connectivity
netstat -tlnp | grep takakrypt

# Check firewall rules
iptables -L | grep takakrypt

# Monitor network traffic
tcpdump -i any -n port 8080  # If management interface is added
```

## Emergency Recovery Procedures

### Service Recovery

**Complete Service Recovery**:
```bash
#!/bin/bash
# emergency-recovery.sh

echo "=== Emergency Takakrypt Recovery ==="

# 1. Stop service
echo "Stopping service..."
sudo systemctl stop takakrypt

# 2. Unmount FUSE mounts
echo "Unmounting FUSE filesystems..."
sudo umount /data/sensitive /data/database /data/public 2>/dev/null

# 3. Kill any hanging processes
echo "Cleaning up processes..."
sudo pkill -f takakrypt-agent

# 4. Check and fix permissions
echo "Fixing permissions..."
sudo chmod 700 /opt/takakrypt/config
sudo chmod 600 /opt/takakrypt/config/keys.json
sudo chmod 644 /opt/takakrypt/config/*.json
sudo chown -R root:root /opt/takakrypt/config

# 5. Validate configuration
echo "Validating configuration..."
for config in /opt/takakrypt/config/*.json; do
    if ! jq . "$config" > /dev/null 2>&1; then
        echo "ERROR: Invalid JSON in $config"
        echo "Please fix configuration before continuing."
        exit 1
    fi
done

# 6. Restart service
echo "Restarting service..."
sudo systemctl start takakrypt

# 7. Verify mounts
echo "Verifying mounts..."
sleep 5
if mount | grep -q takakrypt; then
    echo "SUCCESS: FUSE mounts are active"
else
    echo "ERROR: FUSE mounts failed"
    sudo journalctl -u takakrypt -n 20
    exit 1
fi

echo "Recovery complete."
```

### Configuration Recovery

**Restore from Backup**:
```bash
#!/bin/bash
# restore-config.sh

BACKUP_FILE="$1"

if [[ -z "$BACKUP_FILE" ]]; then
    echo "Usage: $0 <backup-file>"
    echo "Available backups:"
    ls -la /opt/takakrypt/backup/
    exit 1
fi

echo "=== Configuration Restore ==="
echo "Restoring from: $BACKUP_FILE"

# Stop service
sudo systemctl stop takakrypt

# Backup current configuration
sudo mv /opt/takakrypt/config /opt/takakrypt/config.backup.$(date +%Y%m%d-%H%M%S)

# Restore configuration
sudo mkdir -p /opt/takakrypt/config
sudo tar -xzf "$BACKUP_FILE" -C /opt/takakrypt/config

# Fix permissions
sudo chmod 700 /opt/takakrypt/config
sudo chmod 600 /opt/takakrypt/config/keys.json
sudo chmod 644 /opt/takakrypt/config/*.json
sudo chown -R root:root /opt/takakrypt/config

# Validate and restart
sudo /opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --validate-config
sudo systemctl start takakrypt

echo "Configuration restored and service restarted."
```

## Support and Escalation

### Information Collection

When escalating issues, collect:

1. **System Information**:
   ```bash
   uname -a
   lsb_release -a
   df -h
   free -h
   ```

2. **Service Information**:
   ```bash
   sudo systemctl status takakrypt
   sudo journalctl -u takakrypt -n 100 --no-pager
   ```

3. **Configuration Information**:
   ```bash
   # Sanitized configuration (remove sensitive data)
   sudo jq 'del(.[] | .key_material)' /opt/takakrypt/config/keys.json
   sudo cat /opt/takakrypt/config/policy.json
   ```

4. **Error Reproduction**:
   - Exact commands that reproduce the issue
   - Expected vs actual behavior
   - Error messages and logs
   - Timeline of events

### Log Collection Script

```bash
#!/bin/bash
# collect-support-info.sh

SUPPORT_DIR="/tmp/takakrypt-support-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$SUPPORT_DIR"

echo "Collecting Takakrypt support information..."

# System information
uname -a > "$SUPPORT_DIR/system-info.txt"
lsb_release -a >> "$SUPPORT_DIR/system-info.txt" 2>/dev/null
df -h > "$SUPPORT_DIR/disk-usage.txt"
free -h > "$SUPPORT_DIR/memory-usage.txt"

# Service information
sudo systemctl status takakrypt > "$SUPPORT_DIR/service-status.txt"
sudo journalctl -u takakrypt -n 500 --no-pager > "$SUPPORT_DIR/service-logs.txt"

# Configuration (sanitized)
sudo jq 'del(.[] | .key_material)' /opt/takakrypt/config/keys.json > "$SUPPORT_DIR/keys-sanitized.json"
sudo cp /opt/takakrypt/config/*.json "$SUPPORT_DIR/" 2>/dev/null

# Remove actual key material for security
sed -i 's/"key_material": "[^"]*"/"key_material": "[REDACTED]"/g' "$SUPPORT_DIR"/*.json

# Mount information
mount | grep takakrypt > "$SUPPORT_DIR/mount-info.txt"

# Process information
ps aux | grep takakrypt > "$SUPPORT_DIR/process-info.txt"

# Create archive
tar -czf "/tmp/takakrypt-support-$(date +%Y%m%d-%H%M%S).tar.gz" -C /tmp "$(basename "$SUPPORT_DIR")"
rm -rf "$SUPPORT_DIR"

echo "Support information collected: /tmp/takakrypt-support-$(date +%Y%m%d-%H%M%S).tar.gz"
echo "Please provide this file when requesting support."
```

This troubleshooting guide provides comprehensive diagnostic and resolution procedures for the Takakrypt system.