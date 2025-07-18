package main

import (
	"fmt"
	"log"
	"os"

	"github.com/takakrypt/transparent-encryption/internal/config"
	"github.com/takakrypt/transparent-encryption/internal/policy"
)

func main() {
	// Check if running FUSE tests
	if len(os.Args) > 1 && os.Args[1] == "test-fuse" {
		TestFUSEOperations()
		return
	}

	cfg, err := config.Load("./")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	policyEngine := policy.NewEngine(cfg)

	testRequest := &policy.AccessRequest{
		Path:      "/tmp/test-mount/test.txt",
		Action:    "write",
		UID:       1000,
		GID:       1000,
		ProcessID: 1234,
		Binary:    "unknown",
	}

	fmt.Printf("Testing policy for:\n")
	fmt.Printf("  Path: %s\n", testRequest.Path)
	fmt.Printf("  Action: %s\n", testRequest.Action)
	fmt.Printf("  UID: %d\n", testRequest.UID)

	result, err := policyEngine.EvaluateAccess(testRequest)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Result:\n")
		fmt.Printf("  Permission: %s\n", result.Permission)
		fmt.Printf("  Apply Key: %t\n", result.ApplyKey)
		fmt.Printf("  Audit: %t\n", result.Audit)
		fmt.Printf("  Rule ID: %s\n", result.RuleID)
	}

	fmt.Printf("\n📋 Available Resource Sets:\n")
	for _, rs := range cfg.ResourceSets {
		fmt.Printf("  - %s: %s\n", rs.Code, rs.Name)
		for _, r := range rs.ResourceList {
			fmt.Printf("    * %s/%s (subfolder: %t)\n", r.Directory, r.File, r.Subfolder)
		}
	}

	fmt.Printf("\n👥 Available User Sets:\n")
	for _, us := range cfg.UserSets {
		fmt.Printf("  - %s: %s\n", us.Code, us.Name)
		for _, u := range us.Users {
			fmt.Printf("    * UID %d: %s\n", u.UID, u.UName)
		}
	}
}