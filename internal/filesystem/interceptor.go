package filesystem

import (
	"context"
	"fmt"
	"log"
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

	result, evalErr := i.policyEngine.EvaluateAccess(req)
	if evalErr != nil {
		return &OperationResult{
			Allowed: false,
			Error:   fmt.Errorf("policy evaluation failed: %w", evalErr),
		}, evalErr
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
	
	// Determine if we should decrypt based on authorization and apply_key
	shouldDecrypt := result.Permission == "permit" && result.ApplyKey
	
	var data []byte
	var err error
	
	if shouldDecrypt {
		// Authorized + apply_key: true → Decrypt and return plaintext
		log.Printf("[CRYPTO] Decrypting file for authorized user: %s", encryptedPath)
		
		// Check if file exists and get its size first
		fileInfo, statErr := os.Stat(encryptedPath)
		if statErr != nil {
			return &OperationResult{
				Allowed:    false,
				AuditEvent: auditEvent,
				Error:      fmt.Errorf("file not found: %w", statErr),
			}, statErr
		}
		
		// Handle empty files (newly created)
		if fileInfo.Size() == 0 {
			log.Printf("[CRYPTO] File is empty (newly created): %s", encryptedPath)
			return &OperationResult{
				Data:       []byte{},
				Allowed:    true,
				Encrypted:  true,
				AuditEvent: auditEvent,
			}, nil
		}
		
		data, err = i.readAndDecrypt(encryptedPath, guardPoint.ID)
		if err != nil {
			log.Printf("[CRYPTO] Decryption failed for %s: %v", encryptedPath, err)
			// Try to read as plain text if decryption fails (legacy files)
			plainData, readErr := os.ReadFile(encryptedPath)
			if readErr != nil {
				auditEvent.Success = false
				return &OperationResult{
					Allowed:    false,
					AuditEvent: auditEvent,
					Error:      fmt.Errorf("failed to decrypt data: %w", err),
				}, err
			}
			// Check if file is actually encrypted by looking at first few bytes
			isEncrypted := false
			if len(plainData) > 0 {
				// Check for non-printable characters indicating encryption
				for i := 0; i < len(plainData) && i < 32; i++ {
					if plainData[i] < 0x20 || plainData[i] > 0x7E {
						isEncrypted = true
						break
					}
				}
			}
			if isEncrypted {
				log.Printf("[CRYPTO] ERROR: File %s is encrypted but decryption failed: %v", encryptedPath, err)
				auditEvent.Success = false
				return &OperationResult{
					Allowed:    false,
					AuditEvent: auditEvent,
					Error:      fmt.Errorf("file is encrypted but decryption failed: %w", err),
				}, err
			}
			log.Printf("[CRYPTO] File %s appears to be plain text, reading without decryption", encryptedPath)
			data = plainData
		}
	} else {
		// Authorized + apply_key: false OR Unauthorized + apply_key: true → Return raw ciphertext
		log.Printf("[CRYPTO] Reading raw encrypted data (no decryption): %s", encryptedPath)
		data, err = os.ReadFile(encryptedPath)
		if err != nil {
			auditEvent.Success = false
			return &OperationResult{
				Allowed:    false,
				AuditEvent: auditEvent,
				Error:      fmt.Errorf("failed to read encrypted file: %w", err),
			}, err
		}
	}

	return &OperationResult{
		Data:       data,
		Allowed:    true,
		Encrypted:  true,
		AuditEvent: auditEvent,
	}, nil
}

func (i *Interceptor) InterceptWrite(ctx context.Context, op *FileOperation) (*OperationResult, error) {
	log.Printf("[INTERCEPT] InterceptWrite called: path=%s, uid=%d, gid=%d, pid=%d", op.Path, op.UID, op.GID, op.PID)
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
	if guardPoint == nil {
		// Not a guard point - write as plain text
		log.Printf("[INTERCEPT] Writing plain file: %s", op.Path)
		err := i.writeFile(op.Path, op.Data, op.Mode, op.UID, op.GID)
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

	// Always encrypt when writing to guard points (regardless of apply_key)
	encryptedPath := i.getEncryptedPath(guardPoint, op.Path)
	log.Printf("[CRYPTO] Writing encrypted file to: %s", encryptedPath)
	log.Printf("[CRYPTO] Using guard point ID: %s", guardPoint.ID)
	log.Printf("[INTERCEPT] Writing encrypted file: %s -> %s", op.Path, encryptedPath)
	err = i.encryptAndWrite(encryptedPath, op.Data, op.Mode, guardPoint.ID, op.UID, op.GID)
	if err != nil {
		log.Printf("[CRYPTO] ERROR: Failed to encrypt and write file: %v", err)
		auditEvent.Success = false
		return &OperationResult{
			Allowed:    false,
			AuditEvent: auditEvent,
			Error:      fmt.Errorf("failed to encrypt and write file: %w", err),
		}, err
	}
	log.Printf("[CRYPTO] Successfully encrypted and wrote file: %s", encryptedPath)

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

func (i *Interceptor) readAndDecrypt(path string, guardPointID string) ([]byte, error) {
	encryptedData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}

	plainData, err := i.cryptoSvc.DecryptForGuardPoint(encryptedData, guardPointID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plainData, nil
}

func (i *Interceptor) encryptAndWrite(path string, data []byte, mode os.FileMode, guardPointID string, uid, gid int) error {
	encryptedData, err := i.cryptoSvc.EncryptForGuardPoint(data, guardPointID)
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

	// Set correct ownership after file creation
	if err := os.Chown(path, uid, gid); err != nil {
		log.Printf("[INTERCEPT] Warning: Failed to set encrypted file ownership to %d:%d: %v", uid, gid, err)
	} else {
		log.Printf("[INTERCEPT] Set encrypted file ownership to %d:%d for %s", uid, gid, path)
	}

	return nil
}

func (i *Interceptor) writeFile(path string, data []byte, mode os.FileMode, uid, gid int) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Set correct ownership after file creation
	if err := os.Chown(path, uid, gid); err != nil {
		log.Printf("[INTERCEPT] Warning: Failed to set plain file ownership to %d:%d: %v", uid, gid, err)
	} else {
		log.Printf("[INTERCEPT] Set plain file ownership to %d:%d for %s", uid, gid, path)
	}

	return nil
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}