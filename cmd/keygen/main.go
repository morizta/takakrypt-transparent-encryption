package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/takakrypt/transparent-encryption/internal/crypto"
)

var (
	output = flag.String("output", "keys.json", "Output file for keys")
	generate = flag.Bool("generate", false, "Generate new keys")
)

func main() {
	flag.Parse()

	if *generate {
		generateKeys(*output)
	} else {
		flag.Usage()
	}
}

func generateKeys(outputFile string) {
	keys := []crypto.KeyMetadata{
		{
			ID:           "key-sensitive-data-001",
			Name:         "Sensitive Data Encryption Key",
			Type:         "AES256-GCM",
			GuardPointID: "gp-sensitive",
			KeyMaterial:  generateBase64Key(),
			CreatedAt:    time.Now().UnixNano(),
			ModifiedAt:   time.Now().UnixNano(),
			Status:       "active",
			Description:  "Encryption key for sensitive data guard point",
		},
		{
			ID:           "key-database-files-001",
			Name:         "Database Files Encryption Key",
			Type:         "AES256-GCM",
			GuardPointID: "gp-database",
			KeyMaterial:  generateBase64Key(),
			CreatedAt:    time.Now().UnixNano(),
			ModifiedAt:   time.Now().UnixNano(),
			Status:       "active",
			Description:  "Encryption key for database files guard point",
		},
		{
			ID:           "key-public-data-001",
			Name:         "Public Data Key (No Encryption)",
			Type:         "NONE",
			GuardPointID: "gp-public",
			KeyMaterial:  "",
			CreatedAt:    time.Now().UnixNano(),
			ModifiedAt:   time.Now().UnixNano(),
			Status:       "active",
			Description:  "No encryption for public data",
		},
	}

	data, err := json.MarshalIndent(keys, "", "    ")
	if err != nil {
		log.Fatalf("Failed to marshal keys: %v", err)
	}

	if err := os.WriteFile(outputFile, data, 0600); err != nil {
		log.Fatalf("Failed to write keys file: %v", err)
	}

	fmt.Printf("Generated keys file: %s\n", outputFile)
	for _, key := range keys {
		if key.Type != "NONE" {
			fmt.Printf("  %s: %s\n", key.ID, key.KeyMaterial)
		}
	}
}

func generateBase64Key() string {
	key := make([]byte, 32) // 256 bits for AES256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(key)
}