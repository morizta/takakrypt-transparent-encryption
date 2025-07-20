# Takakrypt Configuration Reference

## Overview

This document provides detailed reference information for all Takakrypt configuration files. Each configuration file uses JSON format and follows a specific schema for validation.

## Configuration File Locations

- **Guard Points**: `/opt/takakrypt/config/guard-point.json`
- **Policies**: `/opt/takakrypt/config/policy.json`
- **User Sets**: `/opt/takakrypt/config/user_set.json`
- **Process Sets**: `/opt/takakrypt/config/process_set.json`
- **Keys**: `/opt/takakrypt/config/keys.json`

## 1. Guard Points Configuration (`guard-point.json`)

### Purpose
Define protected directories that will be transparently encrypted.

### Schema
```json
[
  {
    "id": "string",
    "name": "string", 
    "protected_path": "string",
    "secure_storage_path": "string",
    "policy_id": "string",
    "key_id": "string",
    "status": "string"
  }
]
```

### Field Descriptions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier for the guard point |
| `name` | string | Yes | Human-readable name |
| `protected_path` | string | Yes | FUSE mount point (e.g., `/data/sensitive`) |
| `secure_storage_path` | string | Yes | Encrypted storage location |
| `policy_id` | string | Yes | Associated policy ID |
| `key_id` | string | Yes | Encryption key ID |
| `status` | string | Yes | `active`, `inactive`, or `maintenance` |

### Example Configuration
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
    "name": "Database Files Guard Point",
    "protected_path": "/data/database",
    "secure_storage_path": "/secure_storage/database",
    "policy_id": "policy-database-access",
    "key_id": "key-database-001",
    "status": "active"
  }
]
```

### Validation Rules
- `id` must be unique across all guard points
- `protected_path` must be an absolute path
- `secure_storage_path` must be an absolute path
- `policy_id` must reference an existing policy
- `key_id` must reference an existing key

## 2. Policies Configuration (`policy.json`)

### Purpose
Define access control rules for guard points.

### Schema
```json
[
  {
    "id": "string",
    "code": "string",
    "name": "string",
    "policy_type": "string",
    "description": "string",
    "security_rules": [
      {
        "id": "string",
        "order": "integer",
        "resource_set": ["string"],
        "user_set": ["string"],
        "process_set": ["string"],
        "action": ["string"],
        "browsing": "boolean",
        "effect": {
          "permission": "string",
          "option": {
            "apply_key": "boolean",
            "audit": "boolean"
          }
        }
      }
    ]
  }
]
```

### Field Descriptions

#### Policy Level
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique policy identifier |
| `code` | string | Yes | Policy code for internal reference |
| `name` | string | Yes | Human-readable policy name |
| `policy_type` | string | Yes | Always `"life_data_transformation"` |
| `description` | string | No | Policy description |

#### Security Rule Level
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique rule identifier |
| `order` | integer | Yes | Rule evaluation order (ascending) |
| `resource_set` | array | Yes | Resource set IDs (empty = all resources) |
| `user_set` | array | Yes | User set IDs (empty = all users) |
| `process_set` | array | Yes | Process set IDs (empty = all processes) |
| `action` | array | Yes | Allowed actions: `read`, `write`, `all_ops` |
| `browsing` | boolean | Yes | Allow directory browsing |
| `effect.permission` | string | Yes | `permit` or `deny` |
| `effect.option.apply_key` | boolean | Yes | Apply encryption/decryption |
| `effect.option.audit` | boolean | Yes | Log access attempts |

### Example Configuration
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
      }
    ]
  }
]
```

### Best Practices
- Use ascending order numbers (1, 2, 3, ...)
- More specific rules should have lower order numbers
- Always include a default deny rule with high order number
- Empty arrays match all items (wildcard behavior)

## 3. User Sets Configuration (`user_set.json`)

### Purpose
Define groups of users for policy application.

### Schema
```json
[
  {
    "id": "string",
    "code": "string", 
    "name": "string",
    "created_at": "integer",
    "modified_at": "integer",
    "description": "string",
    "users": [
      {
        "index": "integer",
        "id": "string",
        "uid": "integer",
        "uname": "string",
        "fname": "string",
        "gname": "string",
        "gid": "integer",
        "os": "string",
        "type": "string",
        "email": "string",
        "os_domain": "string",
        "os_user": "string",
        "created_at": "integer",
        "modified_at": "integer"
      }
    ]
  }
]
```

### Field Descriptions

#### User Set Level
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique user set identifier |
| `code` | string | Yes | User set code for policy reference |
| `name` | string | Yes | Human-readable name |
| `description` | string | No | User set description |
| `created_at` | integer | No | Creation timestamp (nanoseconds) |
| `modified_at` | integer | No | Modification timestamp (nanoseconds) |

#### User Level
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `index` | integer | Yes | User index within set |
| `id` | string | Yes | Unique user identifier |
| `uid` | integer | Yes | Unix user ID |
| `uname` | string | Yes | Unix username |
| `fname` | string | Yes | Full name |
| `gname` | string | Yes | Group name |
| `gid` | integer | Yes | Unix group ID |
| `os` | string | Yes | Operating system (`linux`) |
| `type` | string | Yes | User type (`system`, `application`) |
| `email` | string | No | Email address |
| `os_domain` | string | No | OS domain |
| `os_user` | string | No | OS username |

### Example Configuration
```json
[
  {
    "id": "user-set-admin",
    "code": "admin-set",
    "name": "System Administrators",
    "description": "System administrators with full access",
    "users": [
      {
        "index": 0,
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
  }
]
```

## 4. Process Sets Configuration (`process_set.json`)

### Purpose
Define groups of authorized processes.

### Schema
```json
[
  {
    "id": "string",
    "code": "string",
    "name": "string",
    "created_at": "integer",
    "modified_at": "integer", 
    "description": "string",
    "resource_set_list": [
      {
        "index": "integer",
        "id": "string",
        "directory": "string",
        "file": "string",
        "signature": ["string"],
        "rwp_exempted_resources": ["string"],
        "created_at": "integer",
        "modified_at": "integer"
      }
    ]
  }
]
```

### Field Descriptions

#### Process Set Level
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique process set identifier |
| `code` | string | Yes | Process set code for policy reference |
| `name` | string | Yes | Human-readable name |
| `description` | string | No | Process set description |

#### Process Level
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `index` | integer | Yes | Process index within set |
| `id` | string | Yes | Unique process identifier |
| `directory` | string | Yes | Process binary directory |
| `file` | string | Yes | Process binary filename |
| `signature` | array | No | Digital signatures (future) |
| `rwp_exempted_resources` | array | No | Exempted resources (future) |

### Example Configuration
```json
[
  {
    "id": "process-set-database",
    "code": "database-processes",
    "name": "Database Processes",
    "description": "Authorized database server processes",
    "resource_set_list": [
      {
        "index": 0,
        "id": "mysql-server",
        "directory": "/usr/sbin/",
        "file": "mysqld",
        "signature": [],
        "rwp_exempted_resources": []
      },
      {
        "index": 1,
        "id": "mysql-client",
        "directory": "/usr/bin/",
        "file": "mysql",
        "signature": [],
        "rwp_exempted_resources": []
      }
    ]
  }
]
```

### Process Identification
Processes are identified by the full path: `directory + file`
- Example: `/usr/sbin/mysqld`
- Symlinks are resolved to actual binary paths
- Process validation happens on every file access

## 5. Keys Configuration (`keys.json`)

### Purpose
Store encryption keys for guard points.

### Schema
```json
[
  {
    "id": "string",
    "name": "string",
    "type": "string", 
    "guard_point_id": "string",
    "key_material": "string",
    "created_at": "string",
    "status": "string"
  }
]
```

### Field Descriptions
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique key identifier |
| `name` | string | Yes | Human-readable key name |
| `type` | string | Yes | Encryption algorithm (`AES256-GCM`) |
| `guard_point_id` | string | Yes | Associated guard point ID |
| `key_material` | string | Yes | Base64-encoded encryption key |
| `created_at` | string | No | Creation timestamp |
| `status` | string | Yes | `active`, `deprecated`, or `revoked` |

### Example Configuration
```json
[
  {
    "id": "key-sensitive-data-001",
    "name": "Sensitive Data Encryption Key",
    "type": "AES256-GCM",
    "guard_point_id": "gp-sensitive",
    "key_material": "base64-encoded-256-bit-key",
    "created_at": "2024-01-01T00:00:00Z",
    "status": "active"
  }
]
```

### Security Considerations
- File permissions must be 600 (owner read/write only)
- Keys should be backed up securely (encrypted)
- Key rotation should be planned (not yet implemented)
- Key material is base64-encoded 256-bit (32 byte) keys

## 6. Configuration Validation

### Validation Tools
```bash
# Validate JSON syntax
jq . /opt/takakrypt/config/policy.json

# Validate configuration
/opt/takakrypt/takakrypt-agent --config /opt/takakrypt/config --validate-config
```

### Cross-Reference Validation
The system validates:
- Guard point policy_id references exist in policies
- Guard point key_id references exist in keys
- Policy resource_set references exist in resource sets
- Policy user_set references exist in user sets
- Policy process_set references exist in process sets

### Common Validation Errors
- **JSON Syntax Error**: Invalid JSON format
- **Missing Required Field**: Required field is null or missing
- **Invalid Reference**: Referenced ID doesn't exist
- **Duplicate ID**: ID is used multiple times
- **Invalid Path**: Path is not absolute or doesn't exist
- **Invalid Type**: Field has wrong data type

## 7. Configuration Management

### Configuration Reload
```bash
# Reload without restart
sudo systemctl reload takakrypt

# Or send signal
sudo pkill -HUP takakrypt-agent
```

### Backup Configuration
```bash
# Create backup
sudo tar -czf takakrypt-config-$(date +%Y%m%d).tar.gz -C /opt/takakrypt/config .

# Restore backup
sudo tar -xzf takakrypt-config-backup.tar.gz -C /opt/takakrypt/config/
sudo systemctl reload takakrypt
```

### Version Control
Consider using Git for configuration management:
```bash
cd /opt/takakrypt/config
sudo git init
sudo git add *.json
sudo git commit -m "Initial configuration"
```

## 8. Environment-Specific Configurations

### Development Environment
- Relaxed policies for testing
- Debug logging enabled
- Test users and processes
- Non-sensitive test data

### Production Environment
- Strict access controls
- Minimal logging
- Real users and applications
- Regular key rotation planned

### Staging Environment
- Production-like policies
- Extended logging for validation
- Synthetic test data
- Performance testing enabled

This configuration reference provides comprehensive details for all Takakrypt configuration files and management procedures.