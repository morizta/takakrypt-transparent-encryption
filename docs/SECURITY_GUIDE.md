# Takakrypt Security Guide

## Overview

This document provides comprehensive security guidance for deploying and operating the Takakrypt transparent encryption system. It covers security architecture, threat model, security controls, and best practices.

## Security Architecture

### Defense in Depth

Takakrypt implements multiple layers of security:

```
┌─────────────────────────────────────────────────────────┐
│                Application Layer                        │
│  • Application-level access controls                   │
│  • User authentication                                 │
└─────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────┐
│                Policy Layer                             │
│  • User-based access control                          │
│  • Process-based access control                       │
│  • Resource-based access control                      │
│  • Action-based permissions                           │
└─────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────┐
│              Encryption Layer                           │
│  • AES-256-GCM encryption                             │
│  • Per-guard-point keys                               │
│  • Authenticated encryption                           │
└─────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────┐
│               System Layer                              │
│  • File system permissions                            │
│  • Process isolation                                  │
│  • Kernel-level controls                              │
└─────────────────────────────────────────────────────────┘
```

### Security Boundaries

1. **User Space Boundary**: Separates applications from system components
2. **Policy Boundary**: Enforces access control decisions
3. **Encryption Boundary**: Protects data at rest
4. **Kernel Boundary**: Provides fundamental OS security

## Threat Model

### Assets Protected

**Primary Assets**:
- Sensitive application data
- Encryption keys
- Configuration files
- Audit logs

**Secondary Assets**:
- System configuration
- User credentials
- Process information

### Threat Actors

**Internal Threats**:
- Malicious insiders with system access
- Compromised user accounts
- Unauthorized applications
- Misconfigured processes

**External Threats**:
- Remote attackers with system access
- Physical access to storage media
- Backup media compromise
- Supply chain attacks

### Attack Vectors

**File System Attacks**:
- Direct access to encrypted storage
- Bypass of FUSE layer
- File system corruption
- Privilege escalation

**Policy Bypass Attacks**:
- Configuration manipulation
- Process impersonation
- User impersonation
- Policy logic flaws

**Cryptographic Attacks**:
- Key extraction
- Weak random number generation
- Side-channel attacks
- Cryptographic implementation flaws

**System-Level Attacks**:
- Kernel exploits
- Container escape
- Memory corruption
- Hardware attacks

## Security Controls

### 1. Access Control

#### User-Based Access Control
```json
{
  "users": [
    {
      "uid": 1000,
      "uname": "appuser",
      "access_level": "application"
    }
  ]
}
```

**Implementation**:
- FUSE extracts real UID/GID from process context
- User identity verified against OS user database
- No password authentication (relies on OS)

**Security Considerations**:
- Trusted OS user authentication
- UID spoofing protection via kernel
- Group membership validation

#### Process-Based Access Control
```json
{
  "processes": [
    {
      "directory": "/usr/bin/",
      "file": "python3",
      "signature": []
    }
  ]
}
```

**Implementation**:
- Process binary identified via `/proc/PID/exe`
- Full path validation against process sets
- Symlink resolution to actual binary

**Security Considerations**:
- Process binary integrity (future: digital signatures)
- Path traversal protection
- Race condition prevention

#### Resource-Based Access Control
- Guard point path matching
- Directory tree protection
- File-level granularity
- Metadata protection

### 2. Encryption Controls

#### Algorithm Selection
- **Primary**: AES-256-GCM
- **Key Size**: 256 bits (32 bytes)
- **Nonce**: 96 bits (12 bytes), cryptographically random
- **Authentication Tag**: 128 bits (16 bytes)

#### Key Management
```go
type Key struct {
    ID           string    // Unique identifier
    Type         string    // "AES256-GCM"
    GuardPointID string    // Associated guard point
    KeyMaterial  string    // Base64-encoded key
    Status       string    // "active", "deprecated", "revoked"
}
```

**Key Generation**:
```bash
# High-entropy key generation
openssl rand -base64 32
```

**Key Storage**:
- File permissions: 600 (owner read/write only)
- JSON format with base64 encoding
- No key escrow (keys stored locally)

**Key Rotation** (Planned):
- Versioned keys per guard point
- Graceful transition period
- Re-encryption of existing data
- Old key retention for recovery

### 3. Audit Controls

#### Audit Events
```go
type AuditEvent struct {
    Timestamp time.Time // Event timestamp
    UserID    int       // User ID
    ProcessID int       // Process ID
    Resource  string    // File path
    Action    string    // "read", "write", etc.
    Result    string    // "permit", "deny"
    PolicyID  string    // Applied policy
    RuleID    string    // Matching rule
}
```

**Audit Trail Requirements**:
- Tamper-evident logging
- Non-repudiation
- Complete operation coverage
- Secure log storage

#### Log Security
```bash
# Secure log file permissions
chmod 640 /var/log/takakrypt.log
chown root:adm /var/log/takakrypt.log

# Log rotation with integrity
/var/log/takakrypt.log {
    daily
    rotate 365
    compress
    delaycompress
    notifempty
    create 640 root adm
    postrotate
        /usr/bin/systemctl reload takakrypt
    endscript
}
```

### 4. Configuration Security

#### File Permissions
```bash
# Configuration directory
chmod 700 /opt/takakrypt/config
chown root:root /opt/takakrypt/config

# Key file (most sensitive)
chmod 600 /opt/takakrypt/config/keys.json
chown root:root /opt/takakrypt/config/keys.json

# Other configuration files
chmod 644 /opt/takakrypt/config/*.json
chown root:root /opt/takakrypt/config/*.json
```

#### Configuration Validation
- JSON schema validation
- Cross-reference integrity checks
- Range and format validation
- Circular reference detection

#### Configuration Backup
```bash
#!/bin/bash
# Secure configuration backup

BACKUP_DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="/opt/takakrypt/backup"
CONFIG_DIR="/opt/takakrypt/config"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup configuration (excluding keys)
tar -czf "$BACKUP_DIR/config-${BACKUP_DATE}.tar.gz" \
    --exclude="keys.json" \
    -C "$CONFIG_DIR" .

# Backup keys separately (encrypted)
gpg --cipher-algo AES256 --compress-algo 1 --symmetric \
    --output "$BACKUP_DIR/keys-${BACKUP_DATE}.gpg" \
    "$CONFIG_DIR/keys.json"

# Set secure permissions
chmod 600 "$BACKUP_DIR"/*
```

## Security Best Practices

### 1. Deployment Security

#### System Hardening
```bash
# Disable unnecessary services
systemctl disable bluetooth
systemctl disable avahi-daemon

# Configure firewall (if needed)
ufw enable
ufw default deny incoming
ufw default allow outgoing

# Secure shared memory
echo "tmpfs /run/shm tmpfs defaults,nodev,nosuid,noexec 0 0" >> /etc/fstab

# Kernel parameters
echo "kernel.dmesg_restrict=1" >> /etc/sysctl.conf
echo "kernel.kptr_restrict=2" >> /etc/sysctl.conf
echo "net.ipv4.conf.all.send_redirects=0" >> /etc/sysctl.conf
```

#### User Account Security
```bash
# Create dedicated service account (if not running as root)
useradd -r -s /bin/false -d /opt/takakrypt takakrypt

# Lock unnecessary accounts
passwd -l games
passwd -l ftp
passwd -l mail
```

#### File System Security
```bash
# Mount options for sensitive data
/secure_storage ext4 defaults,nodev,nosuid,noexec 0 2

# Set up file system encryption (optional additional layer)
cryptsetup luksFormat /dev/sdb1
cryptsetup luksOpen /dev/sdb1 secure_storage
mkfs.ext4 /dev/mapper/secure_storage
```

### 2. Operational Security

#### Key Management Best Practices

**Key Generation**:
```bash
# Use hardware random number generator if available
if [ -c /dev/hwrng ]; then
    rng-tools --daemon
fi

# Generate high-quality entropy
openssl rand -base64 32 > /dev/null  # Warm up
```

**Key Backup Strategy**:
1. Create encrypted backup of keys
2. Store backup in separate physical location
3. Use strong passphrase for backup encryption
4. Test recovery procedures regularly
5. Document key recovery process

**Key Rotation Planning**:
1. Schedule regular key rotation (annually)
2. Plan for emergency key rotation
3. Test rotation procedures in staging
4. Maintain old keys for data recovery
5. Document rotation procedures

#### Access Control Best Practices

**Principle of Least Privilege**:
- Grant minimum necessary permissions
- Use specific process sets, not wildcards
- Regularly review and audit permissions
- Remove unused user/process entries

**Process Whitelisting**:
```json
{
  "process_set": {
    "database_processes": [
      "/usr/sbin/mysqld",      // Specific binary paths
      "/usr/bin/mysql"         // No wildcards
    ]
  }
}
```

**User Access Reviews**:
- Monthly review of user sets
- Quarterly access certification
- Automated monitoring of access patterns
- Alert on unusual access attempts

### 3. Monitoring and Detection

#### Security Monitoring
```bash
# Monitor configuration changes
auditctl -w /opt/takakrypt/config -p rwxa -k takakrypt_config

# Monitor key file access
auditctl -w /opt/takakrypt/config/keys.json -p rwxa -k takakrypt_keys

# Monitor binary execution
auditctl -w /opt/takakrypt/takakrypt-agent -p x -k takakrypt_exec
```

#### Log Analysis
```bash
# Monitor for access denials
journalctl -u takakrypt | grep "permission.*deny"

# Monitor for encryption errors
journalctl -u takakrypt | grep "encryption.*failed"

# Monitor for configuration errors
journalctl -u takakrypt | grep "configuration.*error"
```

#### Alerting Rules
1. **High Priority**:
   - Service failures
   - Configuration tampering
   - Key access violations
   - Mount point failures

2. **Medium Priority**:
   - Unusual access patterns
   - Policy violations
   - Performance degradation
   - Log rotation issues

3. **Low Priority**:
   - Normal access denials
   - Debug messages
   - Routine operations

### 4. Incident Response

#### Security Incident Classification

**Critical Incidents**:
- Key compromise
- Configuration tampering
- Service bypass
- Data exfiltration

**High Priority Incidents**:
- Policy violations
- Unauthorized access attempts
- Performance attacks
- Configuration errors

**Medium Priority Incidents**:
- Unusual access patterns
- Process violations
- Log anomalies
- Resource exhaustion

#### Incident Response Procedures

**Immediate Response** (0-1 hour):
1. Assess incident severity
2. Isolate affected systems
3. Preserve evidence
4. Notify stakeholders
5. Begin containment

**Short-term Response** (1-24 hours):
1. Complete containment
2. Eradicate threats
3. Begin recovery
4. Communication updates
5. Evidence collection

**Long-term Response** (1-30 days):
1. Complete recovery
2. Lessons learned
3. Process improvements
4. Documentation updates
5. Training updates

#### Emergency Procedures

**Key Compromise Response**:
```bash
# Immediate key rotation
/opt/takakrypt/scripts/emergency-key-rotation.sh

# Audit all access during compromise window
journalctl -u takakrypt --since "YYYY-MM-DD HH:MM:SS" | grep ACCESS

# Re-encrypt all data with new keys
/opt/takakrypt/scripts/re-encrypt-all.sh
```

**Service Compromise Response**:
```bash
# Stop service immediately
systemctl stop takakrypt

# Unmount FUSE filesystems
umount /data/sensitive /data/database /data/public

# Investigate and remediate
# Restore from known-good backup
# Restart with new configuration
```

## Compliance and Certification

### Security Standards Alignment

**Common Criteria**:
- Protection Profile: File Encryption
- Evaluation Assurance Level: EAL2 (planned)
- Security Functional Requirements coverage

**FIPS 140-2**:
- Level 1: Cryptographic module requirements
- Approved algorithms: AES-256-GCM
- Key management requirements

**NIST Cybersecurity Framework**:
- Identify: Asset management, risk assessment
- Protect: Access control, data security
- Detect: Continuous monitoring
- Respond: Incident response procedures
- Recover: Recovery planning, improvements

### Audit Requirements

**Internal Audits**:
- Quarterly configuration reviews
- Annual security assessments
- Penetration testing (annual)
- Vulnerability assessments (quarterly)

**External Audits**:
- Third-party security assessment
- Compliance certification
- Vendor security reviews
- Regulatory examinations

### Documentation Requirements

**Security Documentation**:
- Security architecture document
- Risk assessment and mitigation
- Incident response procedures
- Change management procedures
- User access procedures

**Compliance Documentation**:
- Policy and procedure documents
- Training records
- Audit trail reports
- Incident reports
- Change logs

## Security Testing

### Security Test Categories

**Vulnerability Testing**:
- Configuration security scanning
- Dependency vulnerability scanning
- Static code analysis
- Dynamic application security testing

**Penetration Testing**:
- External penetration testing
- Internal penetration testing
- Social engineering testing
- Physical security testing

**Red Team Exercises**:
- Full-scope security testing
- Advanced persistent threat simulation
- Security control bypass testing
- Detection capability testing

### Security Test Automation

```bash
#!/bin/bash
# Automated security testing

# Configuration security scan
/opt/takakrypt/security/config-scan.sh

# Dependency vulnerability scan
npm audit --audit-level high

# File permission check
find /opt/takakrypt -type f -perm /o+w -exec ls -l {} \;

# Service security check
systemctl show takakrypt | grep -E "User|Group|PrivateNetwork"
```

This security guide provides comprehensive guidance for securing the Takakrypt transparent encryption system in production environments.