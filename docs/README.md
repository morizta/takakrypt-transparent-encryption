# Takakrypt Transparent Encryption System

## Overview

Takakrypt is a transparent encryption system that provides real-time file encryption and decryption using FUSE (Filesystem in Userspace). It implements enterprise-grade security controls similar to Thales CTE (Clear Text Encryption) and LDT (Live Data Transformation) systems.

## Key Features

- **Transparent Encryption**: Files are encrypted at rest but appear as plaintext to authorized applications
- **Policy-Based Access Control**: Fine-grained access control based on users, processes, and resources
- **Guard Point Protection**: Directory-level encryption with individual key management
- **Process Set Enforcement**: Only authorized processes can access encrypted data
- **Real-time Encryption/Decryption**: No performance impact on application workflows
- **Audit Trail**: Complete audit logging of all access attempts
- **Key Management**: Centralized key store with guard point-based key mapping

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Applications  │    │   Applications  │    │   Applications  │
│   (Python)      │    │   (MariaDB)     │    │  (Text Editors) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   FUSE Layer    │
                    │ (Transparent)   │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Takakrypt     │
                    │   Agent         │
                    └─────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Policy Engine   │    │ Crypto Service  │    │  Key Store      │
│ (Access Control)│    │ (AES256-GCM)    │    │ (Guard Points)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Encrypted      │
                    │  Storage        │
                    └─────────────────┘
```

## Core Components

### 1. FUSE Filesystem Layer
- **Purpose**: Provides transparent file system interface
- **Location**: `/internal/fuse/`
- **Key Files**:
  - `filesystem.go` - Main FUSE operations
  - `file.go` - File-level operations
  - `helpers.go` - Utility functions

### 2. Takakrypt Agent
- **Purpose**: Central orchestrator for encryption/decryption operations
- **Location**: `/cmd/agent/`
- **Key Files**:
  - `main.go` - Agent entry point
  - `agent.go` - Core agent logic

### 3. Policy Engine
- **Purpose**: Access control and permission management
- **Location**: `/internal/policy/`
- **Key Files**:
  - `engine.go` - Policy evaluation engine
  - `types.go` - Policy data structures

### 4. Crypto Service
- **Purpose**: Encryption/decryption operations
- **Location**: `/internal/crypto/`
- **Key Files**:
  - `service.go` - Crypto operations
  - `keystore.go` - Key management

### 5. File System Interceptor
- **Purpose**: Intercepts file operations and applies policies
- **Location**: `/internal/filesystem/`
- **Key Files**:
  - `interceptor.go` - Main interception logic

## Quick Start

### Prerequisites
- Linux operating system
- Go 1.19+ for building
- FUSE library (`libfuse-dev`)
- MariaDB/MySQL for database testing

### Installation

```bash
# Clone repository
git clone https://github.com/morizta/takakrypt-transparent-encryption.git
cd takakrypt-transparent-encryption

# Build and install
make build
sudo make install

# Start service
sudo systemctl enable takakrypt
sudo systemctl start takakrypt
```

### Configuration

1. **Guard Points**: Define protected directories in `guard-point.json`
2. **Policies**: Configure access rules in `policy.json`
3. **User Sets**: Define user groups in `user_set.json`
4. **Process Sets**: Define allowed processes in `process_set.json`
5. **Keys**: Configure encryption keys in `keys.json`

### Testing

```bash
# Run comprehensive tests
cd appaccess-example
./setup.sh
python3 app.py

# Test process restrictions
cat /data/sensitive/test.txt    # Should be denied
nano /data/sensitive/test.txt   # Should work
```

## Security Model

### Guard Points
- **Definition**: Protected directories with individual encryption keys
- **Example**: `/data/sensitive` → encrypted to `/secure_storage/sensitive`
- **Key Mapping**: Each guard point has a unique encryption key

### Policy Evaluation
1. **User Identification**: Extract UID/GID from FUSE context
2. **Process Identification**: Determine process binary from PID
3. **Resource Matching**: Match file path to resource sets
4. **Rule Evaluation**: Apply security rules in order
5. **Permission Decision**: Grant/deny access with encryption

### Process Set Enforcement
- **Authorized Processes**: Only processes in process sets can access encrypted data
- **Examples**:
  - Python applications: `python3`, `python3.10`
  - Database processes: `mysqld`, `mysql`
  - Text editors: `nano`, `vim`

## Performance

- **Encryption**: AES-256-GCM with hardware acceleration
- **Overhead**: <5% performance impact on file operations
- **Caching**: Intelligent caching of decrypted data
- **Scalability**: Supports concurrent access from multiple applications

## Monitoring

### Logs
```bash
# View agent logs
journalctl -u takakrypt -f

# View policy decisions
journalctl -u takakrypt | grep POLICY

# View crypto operations
journalctl -u takakrypt | grep CRYPTO
```

### Audit Trail
- All access attempts are logged with timestamps
- User, process, and resource information captured
- Success/failure status recorded

## Troubleshooting

### Common Issues

1. **Permission Denied**
   - Check user is in appropriate user set
   - Verify process is in process set
   - Confirm policy allows the operation

2. **File Shows as Encrypted**
   - Check `apply_key` setting in policy
   - Verify encryption key is available
   - Check file ownership matches policy

3. **FUSE Mount Issues**
   - Restart takakrypt service
   - Check for stale mount points
   - Verify FUSE permissions

### Debug Mode
```bash
# Run agent in debug mode
sudo systemctl stop takakrypt
sudo /opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --log-level debug
```

## Development

### Building from Source
```bash
# Install dependencies
go mod download

# Build agent
go build -o takakrypt-agent cmd/agent/main.go

# Run tests
go test ./...
```

### Contributing
1. Fork the repository
2. Create feature branch
3. Add tests for new functionality
4. Submit pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
- GitHub Issues: https://github.com/morizta/takakrypt-transparent-encryption/issues
- Documentation: `/docs/` directory
- Examples: `/appaccess-example/` directory