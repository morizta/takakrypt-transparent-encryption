# Takakrypt Flow Diagrams

## 1. Application Access Flow

```mermaid
sequenceDiagram
    participant App as Application
    participant FUSE as FUSE Layer
    participant Agent as Takakrypt Agent
    participant Policy as Policy Engine
    participant Crypto as Crypto Service
    participant Storage as Encrypted Storage

    App->>FUSE: open("/data/sensitive/file.txt")
    FUSE->>Agent: InterceptOpen()
    Agent->>Policy: EvaluateAccess(user, process, resource)
    
    alt Policy Permits Access
        Policy->>Agent: permit + apply_key=true
        Agent->>Crypto: DecryptFile(file_path, key_id)
        Crypto->>Storage: ReadEncryptedFile()
        Storage->>Crypto: encrypted_data
        Crypto->>Agent: decrypted_data
        Agent->>FUSE: success + decrypted_data
        FUSE->>App: file_descriptor
    else Policy Denies Access
        Policy->>Agent: deny
        Agent->>FUSE: permission_denied
        FUSE->>App: error
    end
```

## 2. File Write Operation Flow

```mermaid
sequenceDiagram
    participant App as Application
    participant FUSE as FUSE Layer
    participant Agent as Takakrypt Agent
    participant Policy as Policy Engine
    participant Crypto as Crypto Service
    participant Storage as Encrypted Storage

    App->>FUSE: write(fd, data)
    FUSE->>Agent: InterceptWrite(data, path)
    Agent->>Policy: EvaluateAccess(user, process, "write")
    
    alt Policy Permits Write
        Policy->>Agent: permit + apply_key=true
        Agent->>Crypto: EncryptData(data, key_id)
        Crypto->>Storage: WriteEncryptedFile(encrypted_data)
        Storage->>Crypto: success
        Crypto->>Agent: success
        Agent->>FUSE: success
        FUSE->>App: bytes_written
    else Policy Denies Write
        Policy->>Agent: deny
        Agent->>FUSE: permission_denied
        FUSE->>App: error
    end
```

## 3. Policy Evaluation Flow

```mermaid
flowchart TD
    A[Access Request] --> B[Extract Context]
    B --> C{User in User Set?}
    C -->|No| D[Next Rule]
    C -->|Yes| E{Process in Process Set?}
    E -->|No| D
    E -->|Yes| F{Resource in Resource Set?}
    F -->|No| D
    F -->|Yes| G{Action Allowed?}
    G -->|No| D
    G -->|Yes| H[Apply Rule Effect]
    H --> I{Effect = Permit?}
    I -->|Yes| J[Allow Access]
    I -->|No| K[Deny Access]
    D --> L{More Rules?}
    L -->|Yes| C
    L -->|No| M[Default Deny]
    M --> K
    J --> N{apply_key = true?}
    N -->|Yes| O[Apply Encryption]
    N -->|No| P[Plain Text Access]
    O --> Q[Log Audit Event]
    P --> Q
    K --> Q
```

## 4. Encryption/Decryption Flow

```mermaid
sequenceDiagram
    participant App as Application
    participant Crypto as Crypto Service
    participant KeyStore as Key Store
    participant Storage as File System

    Note over App,Storage: Encryption Flow
    App->>Crypto: EncryptFile(data, guard_point_id)
    Crypto->>KeyStore: GetKey(guard_point_id)
    KeyStore->>Crypto: encryption_key
    Crypto->>Crypto: Generate IV/Nonce
    Crypto->>Crypto: AES-256-GCM Encrypt
    Crypto->>Storage: WriteFile(iv + ciphertext + tag)
    Storage->>Crypto: success
    Crypto->>App: success

    Note over App,Storage: Decryption Flow
    App->>Crypto: DecryptFile(file_path, guard_point_id)
    Crypto->>Storage: ReadFile(file_path)
    Storage->>Crypto: iv + ciphertext + tag
    Crypto->>KeyStore: GetKey(guard_point_id)
    KeyStore->>Crypto: decryption_key
    Crypto->>Crypto: AES-256-GCM Decrypt
    Crypto->>App: plaintext_data
```

## 5. System Startup Flow

```mermaid
flowchart TD
    A[Start Takakrypt Agent] --> B[Load Configuration]
    B --> C[Validate Config Files]
    C --> D{Config Valid?}
    D -->|No| E[Log Error & Exit]
    D -->|Yes| F[Initialize Components]
    F --> G[Start Policy Engine]
    G --> H[Start Crypto Service]
    H --> I[Load Key Store]
    I --> J[Initialize FUSE Mounts]
    J --> K{Mount Success?}
    K -->|No| L[Log Error & Retry]
    K -->|Yes| M[Start File Interceptor]
    M --> N[Register Signal Handlers]
    N --> O[Agent Ready]
    O --> P[Listen for Requests]
    L --> J
```

## 6. Database Application Flow

```mermaid
sequenceDiagram
    participant PyApp as Python Application
    participant MariaDB as MariaDB Server
    participant FUSE as FUSE Mount
    participant Agent as Takakrypt Agent
    participant Storage as Encrypted Storage

    Note over PyApp,Storage: Database Transaction Flow
    PyApp->>MariaDB: INSERT sensitive_data
    MariaDB->>FUSE: write(/data/database/table.ibd)
    FUSE->>Agent: InterceptWrite(mysqld, data)
    Agent->>Agent: Check Policy (mysqld in mysql-processes)
    Agent->>Agent: Encrypt data (apply_key=true)
    Agent->>Storage: Write encrypted data
    Storage->>Agent: success
    Agent->>FUSE: success
    FUSE->>MariaDB: write success
    MariaDB->>PyApp: transaction complete

    Note over PyApp,Storage: Database Query Flow
    PyApp->>MariaDB: SELECT * FROM sensitive_data
    MariaDB->>FUSE: read(/data/database/table.ibd)
    FUSE->>Agent: InterceptRead(mysqld, file_path)
    Agent->>Agent: Check Policy (mysqld authorized)
    Agent->>Storage: Read encrypted data
    Storage->>Agent: encrypted_data
    Agent->>Agent: Decrypt data (apply_key=true)
    Agent->>FUSE: decrypted_data
    FUSE->>MariaDB: plaintext data
    MariaDB->>PyApp: query results
```

## 7. Process Set Enforcement Flow

```mermaid
flowchart TD
    A[File Access Request] --> B[Extract Process Binary]
    B --> C[Get Process Path from PID]
    C --> D{Process in Process Set?}
    D -->|No| E[Access Denied]
    D -->|Yes| F[Continue Policy Evaluation]
    F --> G{User Authorized?}
    G -->|No| E
    G -->|Yes| H{Resource Matches?}
    H -->|No| E
    H -->|Yes| I[Grant Access]
    I --> J[Apply Encryption if Required]
    E --> K[Log Audit Event - Denied]
    J --> L[Log Audit Event - Granted]
```

## 8. Key Management Flow

```mermaid
sequenceDiagram
    participant Admin as Administrator
    participant KeyStore as Key Store
    participant Agent as Takakrypt Agent
    participant GP as Guard Point

    Note over Admin,GP: Key Generation
    Admin->>KeyStore: Generate new key for guard point
    KeyStore->>KeyStore: Create AES-256 key
    KeyStore->>KeyStore: Store key with guard point mapping
    KeyStore->>Agent: Key ready for use

    Note over Admin,GP: Key Rotation (Future)
    Admin->>KeyStore: Rotate key for guard point
    KeyStore->>KeyStore: Generate new key version
    KeyStore->>Agent: Signal key rotation
    Agent->>GP: Re-encrypt with new key
    GP->>Agent: Re-encryption complete
    Agent->>KeyStore: Confirm key rotation
    KeyStore->>KeyStore: Mark old key for deletion
```

## 9. Audit and Logging Flow

```mermaid
sequenceDiagram
    participant User as User Process
    participant Agent as Takakrypt Agent
    participant Audit as Audit Logger
    participant Syslog as System Log

    User->>Agent: File access request
    Agent->>Agent: Process request
    Agent->>Audit: Create audit event
    Audit->>Audit: Format audit message
    Audit->>Syslog: Write to system log
    
    Note over User,Syslog: Audit Event Structure
    Note right of Audit: timestamp, user_id, process, resource, action, result
    
    Agent->>User: Return access result
```

## 10. Error Handling Flow

```mermaid
flowchart TD
    A[Error Occurred] --> B{Error Type?}
    B -->|Configuration Error| C[Log Error]
    B -->|Key Not Found| D[Log Error & Deny Access]
    B -->|Crypto Error| E[Log Error & Deny Access]
    B -->|FUSE Error| F[Log Error & Return EIO]
    B -->|Policy Error| G[Log Error & Default Deny]
    
    C --> H[Attempt Recovery]
    D --> I[Audit Event]
    E --> I
    F --> I
    G --> I
    
    H --> J{Recovery Success?}
    J -->|Yes| K[Continue Operation]
    J -->|No| L[Service Degradation]
    
    I --> M[Return Error to Application]
    K --> N[Normal Operation]
    L --> O[Alert Administrator]
    M --> O
```

## 11. Configuration Reload Flow

```mermaid
sequenceDiagram
    participant Admin as Administrator
    participant Agent as Takakrypt Agent
    participant Config as Config Manager
    participant FUSE as FUSE Layer

    Admin->>Agent: Send SIGHUP
    Agent->>Config: Reload configuration
    Config->>Config: Validate new config
    
    alt Config Valid
        Config->>Agent: New configuration loaded
        Agent->>FUSE: Update mount points if needed
        FUSE->>Agent: Mount update complete
        Agent->>Agent: Configuration reloaded successfully
    else Config Invalid
        Config->>Agent: Configuration error
        Agent->>Agent: Keep current configuration
        Agent->>Admin: Log configuration error
    end
```

## 12. Multi-User Access Flow

```mermaid
sequenceDiagram
    participant User1 as User 1 (ntoi)
    participant User2 as User 2 (testuser1)
    participant FUSE as FUSE Layer
    participant Agent as Takakrypt Agent
    participant Policy as Policy Engine

    par User 1 Access
        User1->>FUSE: access /data/sensitive/file.txt
        FUSE->>Agent: InterceptOpen(uid=1000, process=python3)
        Agent->>Policy: Evaluate(ntoi, python3, sensitive-data)
        Policy->>Agent: permit + apply_key=true
        Agent->>FUSE: success + decrypted_data
        FUSE->>User1: file access granted
    and User 2 Access
        User2->>FUSE: access /data/sensitive/file.txt
        FUSE->>Agent: InterceptOpen(uid=1001, process=cat)
        Agent->>Policy: Evaluate(testuser1, cat, sensitive-data)
        Policy->>Agent: deny (cat not in process set)
        Agent->>FUSE: permission denied
        FUSE->>User2: access denied
    end
```

These flow diagrams illustrate the complete operational flow of the Takakrypt transparent encryption system, from application access through policy evaluation to encryption/decryption operations.