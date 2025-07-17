package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/takakrypt/transparent-encryption/internal/config"
	"github.com/takakrypt/transparent-encryption/internal/crypto"
	"github.com/takakrypt/transparent-encryption/internal/policy"
)

type Interceptor struct {
	policyEngine *policy.Engine
	cryptoSvc    *crypto.Service
	config       *config.Config
}

type FileOperation struct {
	Type     string
	Path     string
	Data     []byte
	Mode     os.FileMode
	Flags    int
	UID      int
	GID      int
	PID      int
	Binary   string
}

type OperationResult struct {
	Data       []byte
	Allowed    bool
	Encrypted  bool
	Error      error
	AuditEvent *AuditEvent
}

type AuditEvent struct {
	Operation  string
	Path       string
	User       int
	Process    string
	Permission string
	RuleID     string
	Success    bool
	Timestamp  int64
}

func NewInterceptor(policyEngine *policy.Engine, cryptoSvc *crypto.Service, cfg *config.Config) *Interceptor {
	return &Interceptor{
		policyEngine: policyEngine,
		cryptoSvc:    cryptoSvc,
		config:       cfg,
	}
}

func (i *Interceptor) InterceptOpen(ctx context.Context, op *FileOperation) (*OperationResult, error) {
	req := &policy.AccessRequest{
		Path:      op.Path,
		Action:    "read",
		UID:       op.UID,
		GID:       op.GID,
		ProcessID: op.PID,
		Binary:    op.Binary,
	}

	result, err := i.policyEngine.EvaluateAccess(req)
	if err != nil {
		return &OperationResult{
			Allowed: false,
			Error:   fmt.Errorf("policy evaluation failed: %w", err),
		}, err
	}

	auditEvent := &AuditEvent{
		Operation:  "open",
		Path:       op.Path,
		User:       op.UID,
		Process:    op.Binary,
		Permission: result.Permission,
		RuleID:     result.RuleID,
		Success:    result.Permission == "permit",
		Timestamp:  getCurrentTimestamp(),
	}

	if result.Permission != "permit" {
		return &OperationResult{
			Allowed:    false,
			AuditEvent: auditEvent,
			Error:      fmt.Errorf("access denied by policy"),
		}, nil
	}

	guardPoint := i.findGuardPointForPath(op.Path)
	if guardPoint == nil || !result.ApplyKey {
		return &OperationResult{
			Allowed:    true,
			Encrypted:  false,
			AuditEvent: auditEvent,
		}, nil
	}

	encryptedPath := i.getEncryptedPath(guardPoint, op.Path)
	data, err := i.readAndDecrypt(encryptedPath)
	if err != nil {
		auditEvent.Success = false
		return &OperationResult{
			Allowed:    false,
			AuditEvent: auditEvent,
			Error:      fmt.Errorf("failed to decrypt file: %w", err),
		}, err
	}

	return &OperationResult{
		Data:       data,
		Allowed:    true,
		Encrypted:  true,
		AuditEvent: auditEvent,
	}, nil
}

func (i *Interceptor) InterceptWrite(ctx context.Context, op *FileOperation) (*OperationResult, error) {
	req := &policy.AccessRequest{
		Path:      op.Path,
		Action:    "write",
		UID:       op.UID,
		GID:       op.GID,
		ProcessID: op.PID,
		Binary:    op.Binary,
	}

	result, err := i.policyEngine.EvaluateAccess(req)
	if err != nil {
		return &OperationResult{
			Allowed: false,
			Error:   fmt.Errorf("policy evaluation failed: %w", err),
		}, err
	}

	auditEvent := &AuditEvent{
		Operation:  "write",
		Path:       op.Path,
		User:       op.UID,
		Process:    op.Binary,
		Permission: result.Permission,
		RuleID:     result.RuleID,
		Success:    result.Permission == "permit",
		Timestamp:  getCurrentTimestamp(),
	}

	if result.Permission != "permit" {
		return &OperationResult{
			Allowed:    false,
			AuditEvent: auditEvent,
			Error:      fmt.Errorf("access denied by policy"),
		}, nil
	}

	guardPoint := i.findGuardPointForPath(op.Path)
	if guardPoint == nil || !result.ApplyKey {
		err := i.writeFile(op.Path, op.Data, op.Mode)
		if err != nil {
			auditEvent.Success = false
		}
		return &OperationResult{
			Allowed:    true,
			Encrypted:  false,
			AuditEvent: auditEvent,
			Error:      err,
		}, err
	}

	encryptedPath := i.getEncryptedPath(guardPoint, op.Path)
	err = i.encryptAndWrite(encryptedPath, op.Data, op.Mode)
	if err != nil {
		auditEvent.Success = false
		return &OperationResult{
			Allowed:    false,
			AuditEvent: auditEvent,
			Error:      fmt.Errorf("failed to encrypt and write file: %w", err),
		}, err
	}

	return &OperationResult{
		Allowed:    true,
		Encrypted:  true,
		AuditEvent: auditEvent,
	}, nil
}

func (i *Interceptor) InterceptList(ctx context.Context, op *FileOperation) (*OperationResult, error) {
	req := &policy.AccessRequest{
		Path:      op.Path,
		Action:    "browse",
		UID:       op.UID,
		GID:       op.GID,
		ProcessID: op.PID,
		Binary:    op.Binary,
	}

	result, err := i.policyEngine.EvaluateAccess(req)
	if err != nil {
		return &OperationResult{
			Allowed: false,
			Error:   fmt.Errorf("policy evaluation failed: %w", err),
		}, err
	}

	auditEvent := &AuditEvent{
		Operation:  "list",
		Path:       op.Path,
		User:       op.UID,
		Process:    op.Binary,
		Permission: result.Permission,
		RuleID:     result.RuleID,
		Success:    result.Permission == "permit",
		Timestamp:  getCurrentTimestamp(),
	}

	if result.Permission != "permit" {
		return &OperationResult{
			Allowed:    false,
			AuditEvent: auditEvent,
			Error:      fmt.Errorf("browse access denied by policy"),
		}, nil
	}

	return &OperationResult{
		Allowed:    true,
		AuditEvent: auditEvent,
	}, nil
}

func (i *Interceptor) findGuardPointForPath(path string) *config.GuardPoint {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil
	}

	var bestMatch *config.GuardPoint
	maxDepth := -1

	for j := range i.config.GuardPoints {
		gp := &i.config.GuardPoints[j]
		if !gp.Enabled {
			continue
		}

		absGuardPath, err := filepath.Abs(gp.ProtectedPath)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(absGuardPath, absPath)
		if err != nil {
			continue
		}

		if !filepath.IsAbs(rel) && rel != ".." && !filepath.HasPrefix(rel, "../") {
			depth := len(filepath.SplitList(absGuardPath))
			if depth > maxDepth {
				maxDepth = depth
				bestMatch = gp
			}
		}
	}

	return bestMatch
}

func (i *Interceptor) getEncryptedPath(gp *config.GuardPoint, originalPath string) string {
	rel, err := filepath.Rel(gp.ProtectedPath, originalPath)
	if err != nil {
		return originalPath
	}
	return filepath.Join(gp.SecureStoragePath, rel)
}

func (i *Interceptor) readAndDecrypt(path string) ([]byte, error) {
	encryptedData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}

	plainData, err := i.cryptoSvc.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plainData, nil
}

func (i *Interceptor) encryptAndWrite(path string, data []byte, mode os.FileMode) error {
	encryptedData, err := i.cryptoSvc.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, encryptedData, mode); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

func (i *Interceptor) writeFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(path, data, mode)
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}