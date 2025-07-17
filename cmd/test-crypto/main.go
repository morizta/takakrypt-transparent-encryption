package main

import (
	"fmt"
	"log"

	"github.com/takakrypt/transparent-encryption/internal/crypto"
)

func main() {
	fmt.Println("🔐 Testing Crypto Service")
	
	// Generate a key
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	fmt.Printf("✅ Generated key: %d bytes\n", len(key))
	
	// Create key provider and service
	keyProvider := crypto.NewLocalKeyProvider(key)
	cryptoSvc := crypto.NewService(keyProvider)
	
	// Test data
	plaintext := []byte("Hello, Takakrypt Transparent Encryption!")
	fmt.Printf("📝 Original data: %s\n", string(plaintext))
	
	// Encrypt
	ciphertext, err := cryptoSvc.Encrypt(plaintext)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	fmt.Printf("🔒 Encrypted: %d bytes\n", len(ciphertext))
	
	// Decrypt
	decrypted, err := cryptoSvc.Decrypt(ciphertext)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}
	fmt.Printf("🔓 Decrypted: %s\n", string(decrypted))
	
	// Verify
	if string(plaintext) == string(decrypted) {
		fmt.Println("✅ Encryption/Decryption test PASSED")
	} else {
		fmt.Println("❌ Encryption/Decryption test FAILED")
	}
}