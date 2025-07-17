package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/takakrypt/transparent-encryption/internal/audit"
	"github.com/takakrypt/transparent-encryption/internal/config"
	"github.com/takakrypt/transparent-encryption/internal/crypto"
	"github.com/takakrypt/transparent-encryption/internal/filesystem"
	"github.com/takakrypt/transparent-encryption/internal/fuse"
	"github.com/takakrypt/transparent-encryption/internal/policy"
)

type Agent struct {
	config        *config.Config
	policyEngine  *policy.Engine
	cryptoSvc     *crypto.Service
	interceptor   *filesystem.Interceptor
	mountManager  *fuse.MountManager
	auditLogger   *audit.Logger
}

func New(cfg *config.Config) (*Agent, error) {
	policyEngine := policy.NewEngine(cfg)

	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate crypto key: %w", err)
	}

	keyProvider := crypto.NewLocalKeyProvider(key)
	cryptoSvc := crypto.NewService(keyProvider)

	interceptor := filesystem.NewInterceptor(policyEngine, cryptoSvc, cfg)
	mountManager := fuse.NewMountManager(interceptor)

	auditLogger, err := audit.NewLogger("/var/log/takakrypt-audit.log", true)
	if err != nil {
		log.Printf("Warning: Failed to initialize audit logger: %v", err)
		auditLogger, _ = audit.NewLogger("", false)
	}

	return &Agent{
		config:        cfg,
		policyEngine:  policyEngine,
		cryptoSvc:     cryptoSvc,
		interceptor:   interceptor,
		mountManager:  mountManager,
		auditLogger:   auditLogger,
	}, nil
}

func (a *Agent) Start(ctx context.Context) error {
	log.Printf("Starting Takakrypt Transparent Encryption Agent")
	log.Printf("Loaded %d guard points", len(a.config.GuardPoints))
	log.Printf("Loaded %d policies", len(a.config.Policies))

	if err := a.mountManager.MountGuardPoints(ctx, a.config.GuardPoints); err != nil {
		return fmt.Errorf("failed to mount guard points: %w", err)
	}

	for _, gp := range a.config.GuardPoints {
		if gp.Enabled {
			log.Printf("Guard Point: %s -> %s (Policy: %s)", 
				gp.ProtectedPath, gp.SecureStoragePath, gp.Policy)
		}
	}

	log.Printf("Agent started successfully. FUSE filesystems mounted.")

	<-ctx.Done()
	log.Printf("Agent shutting down...")

	if err := a.mountManager.UnmountAll(); err != nil {
		log.Printf("Error during unmount: %v", err)
	}

	a.auditLogger.Close()

	return nil
}

func (a *Agent) GetInterceptor() *filesystem.Interceptor {
	return a.interceptor
}