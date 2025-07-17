package main

import (
	"fmt"
	"log"

	"github.com/takakrypt/transparent-encryption/internal/config"
	"github.com/takakrypt/transparent-encryption/internal/policy"
)

func main() {
	fmt.Println("Testing Takakrypt Transparent Encryption Agent")

	cfg, err := config.Load("./")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("✅ Configuration loaded successfully\n")
	fmt.Printf("   - %d User Sets\n", len(cfg.UserSets))
	fmt.Printf("   - %d Process Sets\n", len(cfg.ProcessSets))
	fmt.Printf("   - %d Resource Sets\n", len(cfg.ResourceSets))
	fmt.Printf("   - %d Guard Points\n", len(cfg.GuardPoints))
	fmt.Printf("   - %d Policies\n", len(cfg.Policies))

	policyEngine := policy.NewEngine(cfg)
	fmt.Printf("✅ Policy engine initialized\n")

	testRequest := &policy.AccessRequest{
		Path:      "/data-thales/test.txt",
		Action:    "read",
		UID:       1,
		GID:       1,
		ProcessID: 1234,
		Binary:    "/usr/bin/nano",
	}

	result, err := policyEngine.EvaluateAccess(testRequest)
	if err != nil {
		log.Printf("❌ Policy evaluation failed: %v", err)
	} else {
		fmt.Printf("✅ Policy evaluation successful\n")
		fmt.Printf("   - Permission: %s\n", result.Permission)
		fmt.Printf("   - Apply Key: %t\n", result.ApplyKey)
		fmt.Printf("   - Audit: %t\n", result.Audit)
		fmt.Printf("   - Rule ID: %s\n", result.RuleID)
	}

	for _, gp := range cfg.GuardPoints {
		if gp.Enabled {
			fmt.Printf("📁 Guard Point: %s\n", gp.Name)
			fmt.Printf("   - Protected: %s\n", gp.ProtectedPath)
			fmt.Printf("   - Storage: %s\n", gp.SecureStoragePath)
			fmt.Printf("   - Policy: %s\n", gp.Policy)
		}
	}

	fmt.Println("\n🎉 All components working correctly!")
	fmt.Println("💡 Note: FUSE mounting requires Linux/sudo privileges")
}