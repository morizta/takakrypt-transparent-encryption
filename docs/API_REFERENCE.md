# Takakrypt API Reference

## Overview

This document describes the internal APIs and interfaces used within the Takakrypt system. These APIs are designed for internal component communication and future extensibility.

## Core Interfaces

### 1. Policy Engine Interface

```go
// PolicyEngine provides policy evaluation and management functionality
type PolicyEngine interface {
    // EvaluateAccess evaluates an access request against configured policies
    EvaluateAccess(ctx context.Context, request *AccessRequest) (*AccessResult, error)
    
    // LoadPolicies loads policy configuration from the specified path
    LoadPolicies(configPath string) error
    
    // ReloadPolicies reloads policy configuration without restart
    ReloadPolicies() error
    
    // ValidatePolicy validates a policy configuration
    ValidatePolicy(policy *Policy) error
    
    // GetPolicyStats returns policy evaluation statistics
    GetPolicyStats() *PolicyStats
}
```

#### AccessRequest Structure
```go
type AccessRequest struct {
    Path      string    // File path being accessed
    Action    string    // "read", "write", "all_ops"
    UID       int       // User ID
    GID       int       // Group ID
    ProcessID int       // Process ID
    Binary    string    // Process binary path
    Timestamp time.Time // Request timestamp
}
```

#### AccessResult Structure
```go
type AccessResult struct {
    Allowed    bool      // Access allowed/denied
    Permission string    // "permit" or "deny"
    ApplyKey   bool      // Apply encryption/decryption
    Audit      bool      // Log this access
    RuleID     string    // Matching rule ID
    PolicyID   string    // Applied policy ID
    Reason     string    // Decision reason
}
```

### 2. Crypto Service Interface

```go
// CryptoService provides encryption and decryption operations
type CryptoService interface {
    // Encrypt encrypts data using the specified key
    Encrypt(data []byte, keyID string) ([]byte, error)
    
    // Decrypt decrypts data using the specified key
    Decrypt(data []byte, keyID string) ([]byte, error)
    
    // EncryptFile encrypts an entire file
    EncryptFile(inputPath, outputPath, keyID string) error
    
    // DecryptFile decrypts an entire file
    DecryptFile(inputPath, outputPath, keyID string) error
    
    // EncryptForGuardPoint encrypts data for a specific guard point
    EncryptForGuardPoint(data []byte, guardPointID string) ([]byte, error)
    
    // DecryptForGuardPoint decrypts data for a specific guard point
    DecryptForGuardPoint(data []byte, guardPointID string) ([]byte, error)
    
    // GenerateKey generates a new encryption key
    GenerateKey() ([]byte, error)
    
    // ValidateKey validates a key format and strength
    ValidateKey(key []byte) error
}
```

#### Encryption Parameters
```go
type EncryptionParams struct {
    Algorithm string // "AES256-GCM"
    KeySize   int    // 256 bits
    NonceSize int    // 96 bits
    TagSize   int    // 128 bits
}
```

### 3. Key Store Interface

```go
// KeyStore manages encryption keys
type KeyStore interface {
    // GetKey retrieves a key by ID
    GetKey(keyID string) (*Key, error)
    
    // GetKeyForGuardPoint retrieves key for a guard point
    GetKeyForGuardPoint(guardPointID string) (*Key, error)
    
    // StoreKey stores a new key
    StoreKey(key *Key) error
    
    // UpdateKey updates an existing key
    UpdateKey(key *Key) error
    
    // DeleteKey deletes a key (marks as revoked)
    DeleteKey(keyID string) error
    
    // ListKeys lists all keys with optional filter
    ListKeys(filter *KeyFilter) ([]*Key, error)
    
    // LoadKeys loads keys from configuration
    LoadKeys(configPath string) error
    
    // ReloadKeys reloads key configuration
    ReloadKeys() error
}
```

#### Key Structure
```go
type Key struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`
    Type          string    `json:"type"`
    GuardPointID  string    `json:"guard_point_id"`
    KeyMaterial   string    `json:"key_material"`
    CreatedAt     time.Time `json:"created_at"`
    Status        string    `json:"status"`
    Version       int       `json:"version,omitempty"`
}
```

### 4. File System Interceptor Interface

```go
// Interceptor handles file system operation interception
type Interceptor interface {
    // InterceptOpen handles file open operations
    InterceptOpen(ctx context.Context, op *FileOperation) (*OperationResult, error)
    
    // InterceptWrite handles file write operations  
    InterceptWrite(ctx context.Context, op *FileOperation) (*OperationResult, error)
    
    // InterceptRead handles file read operations
    InterceptRead(ctx context.Context, op *FileOperation) (*OperationResult, error)
    
    // InterceptList handles directory listing operations
    InterceptList(ctx context.Context, op *FileOperation) (*OperationResult, error)
    
    // InterceptDelete handles file deletion operations
    InterceptDelete(ctx context.Context, op *FileOperation) (*OperationResult, error)
}
```

#### FileOperation Structure
```go
type FileOperation struct {
    Type    string      // "open", "read", "write", "list", "delete"
    Path    string      // File path
    Data    []byte      // Data for write operations
    Mode    os.FileMode // File mode for create operations
    Flags   int         // Open flags
    UID     int         // User ID
    GID     int         // Group ID
    PID     int         // Process ID
    Binary  string      // Process binary path
}
```

#### OperationResult Structure
```go
type OperationResult struct {
    Allowed    bool         // Operation allowed
    Data       []byte       // Data for read operations
    Encrypted  bool         // Data is encrypted
    Error      error        // Operation error
    AuditEvent *AuditEvent  // Audit event data
}
```

### 5. Configuration Manager Interface

```go
// ConfigManager handles configuration management
type ConfigManager interface {
    // LoadConfiguration loads all configuration files
    LoadConfiguration(configPath string) error
    
    // ReloadConfiguration reloads configuration
    ReloadConfiguration() error
    
    // ValidateConfiguration validates all configuration
    ValidateConfiguration() error
    
    // GetGuardPoints returns guard point configuration
    GetGuardPoints() ([]*GuardPoint, error)
    
    // GetPolicies returns policy configuration
    GetPolicies() ([]*Policy, error)
    
    // GetUserSets returns user set configuration
    GetUserSets() ([]*UserSet, error)
    
    // GetProcessSets returns process set configuration
    GetProcessSets() ([]*ProcessSet, error)
    
    // WatchConfiguration watches for configuration changes
    WatchConfiguration(callback func()) error
}
```

### 6. Audit Logger Interface

```go
// AuditLogger handles audit event logging
type AuditLogger interface {
    // LogAccess logs a file access event
    LogAccess(event *AccessAuditEvent) error
    
    // LogPolicy logs a policy evaluation event
    LogPolicy(event *PolicyAuditEvent) error
    
    // LogCrypto logs a cryptographic operation event
    LogCrypto(event *CryptoAuditEvent) error
    
    // LogSystem logs a system event
    LogSystem(event *SystemAuditEvent) error
    
    // GetAuditEvents retrieves audit events with filter
    GetAuditEvents(filter *AuditFilter) ([]*AuditEvent, error)
}
```

## Data Structures

### 1. Guard Point Configuration

```go
type GuardPoint struct {
    ID                string `json:"id"`
    Name              string `json:"name"`
    ProtectedPath     string `json:"protected_path"`
    SecureStoragePath string `json:"secure_storage_path"`
    PolicyID          string `json:"policy_id"`
    KeyID             string `json:"key_id"`
    Status            string `json:"status"`
}
```

### 2. Policy Configuration

```go
type Policy struct {
    ID            string         `json:"id"`
    Code          string         `json:"code"`
    Name          string         `json:"name"`
    PolicyType    string         `json:"policy_type"`
    Description   string         `json:"description"`
    SecurityRules []*SecurityRule `json:"security_rules"`
}

type SecurityRule struct {
    ID          string        `json:"id"`
    Order       int           `json:"order"`
    ResourceSet []string      `json:"resource_set"`
    UserSet     []string      `json:"user_set"`
    ProcessSet  []string      `json:"process_set"`
    Action      []string      `json:"action"`
    Browsing    bool          `json:"browsing"`
    Effect      *RuleEffect   `json:"effect"`
}

type RuleEffect struct {
    Permission string      `json:"permission"`
    Option     *RuleOption `json:"option"`
}

type RuleOption struct {
    ApplyKey bool `json:"apply_key"`
    Audit    bool `json:"audit"`
}
```

### 3. User Set Configuration

```go
type UserSet struct {
    ID          string    `json:"id"`
    Code        string    `json:"code"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   int64     `json:"created_at"`
    ModifiedAt  int64     `json:"modified_at"`
    Users       []*User   `json:"users"`
}

type User struct {
    Index      int    `json:"index"`
    ID         string `json:"id"`
    UID        int    `json:"uid"`
    UName      string `json:"uname"`
    FName      string `json:"fname"`
    GName      string `json:"gname"`
    GID        int    `json:"gid"`
    OS         string `json:"os"`
    Type       string `json:"type"`
    Email      string `json:"email,omitempty"`
    OSDomain   string `json:"os_domain,omitempty"`
    OSUser     string `json:"os_user,omitempty"`
    CreatedAt  int64  `json:"created_at"`
    ModifiedAt int64  `json:"modified_at"`
}
```

### 4. Process Set Configuration

```go
type ProcessSet struct {
    ID              string            `json:"id"`
    Code            string            `json:"code"`
    Name            string            `json:"name"`
    Description     string            `json:"description"`
    CreatedAt       int64             `json:"created_at"`
    ModifiedAt      int64             `json:"modified_at"`
    ResourceSetList []*ProcessResource `json:"resource_set_list"`
}

type ProcessResource struct {
    Index                 int      `json:"index"`
    ID                    string   `json:"id"`
    Directory             string   `json:"directory"`
    File                  string   `json:"file"`
    Signature             []string `json:"signature"`
    RWPExemptedResources  []string `json:"rwp_exempted_resources"`
    CreatedAt             int64    `json:"created_at"`
    ModifiedAt            int64    `json:"modified_at"`
}
```

### 5. Audit Event Structures

```go
type AuditEvent struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"`
    Timestamp time.Time `json:"timestamp"`
    UserID    int       `json:"user_id"`
    ProcessID int       `json:"process_id"`
    Resource  string    `json:"resource"`
    Action    string    `json:"action"`
    Result    string    `json:"result"`
    Details   string    `json:"details"`
}

type AccessAuditEvent struct {
    *AuditEvent
    Path       string `json:"path"`
    Permission string `json:"permission"`
    PolicyID   string `json:"policy_id"`
    RuleID     string `json:"rule_id"`
}

type PolicyAuditEvent struct {
    *AuditEvent
    PolicyID   string `json:"policy_id"`
    RuleID     string `json:"rule_id"`
    Decision   string `json:"decision"`
    Reason     string `json:"reason"`
}

type CryptoAuditEvent struct {
    *AuditEvent
    Operation   string `json:"operation"`
    KeyID       string `json:"key_id"`
    DataSize    int64  `json:"data_size"`
    Success     bool   `json:"success"`
}
```

## Error Types

### 1. Policy Errors

```go
var (
    ErrPolicyNotFound     = errors.New("policy not found")
    ErrRuleNotFound       = errors.New("rule not found")
    ErrInvalidPolicy      = errors.New("invalid policy configuration")
    ErrAccessDenied       = errors.New("access denied by policy")
    ErrPolicyEvaluation   = errors.New("policy evaluation error")
)
```

### 2. Crypto Errors

```go
var (
    ErrKeyNotFound        = errors.New("encryption key not found")
    ErrInvalidKey         = errors.New("invalid encryption key")
    ErrEncryptionFailed   = errors.New("encryption operation failed")
    ErrDecryptionFailed   = errors.New("decryption operation failed")
    ErrInvalidCiphertext  = errors.New("invalid ciphertext")
    ErrKeyGeneration      = errors.New("key generation failed")
)
```

### 3. Configuration Errors

```go
var (
    ErrConfigNotFound     = errors.New("configuration file not found")
    ErrInvalidConfig      = errors.New("invalid configuration")
    ErrConfigValidation   = errors.New("configuration validation failed")
    ErrConfigReload       = errors.New("configuration reload failed")
)
```

### 4. File System Errors

```go
var (
    ErrFileNotFound       = errors.New("file not found")
    ErrPermissionDenied   = errors.New("permission denied")
    ErrInvalidPath        = errors.New("invalid file path")
    ErrMountFailed        = errors.New("FUSE mount failed")
    ErrOperationNotSupported = errors.New("operation not supported")
)
```

## Context and Middleware

### 1. Request Context

```go
type RequestContext struct {
    RequestID   string
    UserID      int
    ProcessID   int
    Binary      string
    StartTime   time.Time
    TraceID     string
}

// Context keys
var (
    ContextKeyRequestID = "request_id"
    ContextKeyUserID    = "user_id"
    ContextKeyProcessID = "process_id"
    ContextKeyBinary    = "binary"
)
```

### 2. Middleware Interface

```go
type Middleware interface {
    Handle(ctx context.Context, request interface{}, next Handler) (interface{}, error)
}

type Handler func(ctx context.Context, request interface{}) (interface{}, error)
```

## Metrics and Monitoring

### 1. Metrics Interface

```go
type MetricsCollector interface {
    // Counter metrics
    IncrementCounter(name string, tags map[string]string)
    
    // Gauge metrics
    SetGauge(name string, value float64, tags map[string]string)
    
    // Histogram metrics
    RecordHistogram(name string, value float64, tags map[string]string)
    
    // Timer metrics
    RecordTimer(name string, duration time.Duration, tags map[string]string)
}
```

### 2. Health Check Interface

```go
type HealthChecker interface {
    // CheckHealth performs a health check
    CheckHealth(ctx context.Context) *HealthStatus
    
    // GetHealthStatus returns current health status
    GetHealthStatus() *HealthStatus
}

type HealthStatus struct {
    Status     string            `json:"status"`     // "healthy", "degraded", "unhealthy"
    Timestamp  time.Time         `json:"timestamp"`
    Components map[string]string `json:"components"`
    Details    string            `json:"details,omitempty"`
}
```

## Plugin Interface (Future)

### 1. Plugin Manager

```go
type PluginManager interface {
    // LoadPlugin loads a plugin
    LoadPlugin(path string) (Plugin, error)
    
    // UnloadPlugin unloads a plugin
    UnloadPlugin(name string) error
    
    // ListPlugins lists loaded plugins
    ListPlugins() []Plugin
    
    // GetPlugin gets a plugin by name
    GetPlugin(name string) (Plugin, error)
}

type Plugin interface {
    // Name returns the plugin name
    Name() string
    
    // Version returns the plugin version
    Version() string
    
    // Initialize initializes the plugin
    Initialize(config map[string]interface{}) error
    
    // Shutdown shuts down the plugin
    Shutdown() error
}
```

### 2. Key Management Plugin

```go
type KeyManagementPlugin interface {
    Plugin
    
    // GenerateKey generates a new key
    GenerateKey(keyType string, params map[string]interface{}) (*Key, error)
    
    // ImportKey imports an existing key
    ImportKey(keyData []byte, params map[string]interface{}) (*Key, error)
    
    // ExportKey exports a key
    ExportKey(keyID string, format string) ([]byte, error)
    
    // RotateKey rotates a key
    RotateKey(keyID string) (*Key, error)
}
```

This API reference provides comprehensive documentation for all internal interfaces and data structures used in the Takakrypt system.