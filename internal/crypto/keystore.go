package crypto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type KeyMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	GuardPointID string `json:"guard_point_id"`
	KeyMaterial string `json:"key_material"` // Base64 encoded
	CreatedAt   int64  `json:"created_at"`
	ModifiedAt  int64  `json:"modified_at"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type FileKeyProvider struct {
	keys          map[string]*KeyMetadata
	guardPointMap map[string]string // guardPointID -> keyID
}

func NewFileKeyProvider(keysFile string) (*FileKeyProvider, error) {
	provider := &FileKeyProvider{
		keys:          make(map[string]*KeyMetadata),
		guardPointMap: make(map[string]string),
	}

	if err := provider.loadKeys(keysFile); err != nil {
		return nil, fmt.Errorf("failed to load keys: %w", err)
	}

	return provider, nil
}

func (p *FileKeyProvider) loadKeys(keysFile string) error {
	data, err := os.ReadFile(keysFile)
	if err != nil {
		return fmt.Errorf("failed to read keys file: %w", err)
	}

	var keysList []KeyMetadata
	if err := json.Unmarshal(data, &keysList); err != nil {
		return fmt.Errorf("failed to parse keys file: %w", err)
	}

	for _, key := range keysList {
		p.keys[key.ID] = &key
		if key.GuardPointID != "" {
			p.guardPointMap[key.GuardPointID] = key.ID
		}
	}

	log.Printf("[CRYPTO] Loaded %d keys from file", len(keysList))
	return nil
}

func (p *FileKeyProvider) GetKey(keyID string) ([]byte, error) {
	log.Printf("[CRYPTO] Looking up key: %s", keyID)
	metadata, exists := p.keys[keyID]
	if !exists {
		log.Printf("[CRYPTO] ERROR: Key not found: %s", keyID)
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	log.Printf("[CRYPTO] Found key: %s, type: %s, status: %s", keyID, metadata.Type, metadata.Status)
	if metadata.Status != "active" {
		log.Printf("[CRYPTO] ERROR: Key is not active: %s", keyID)
		return nil, fmt.Errorf("key is not active: %s", keyID)
	}

	if metadata.Type == "NONE" {
		log.Printf("[CRYPTO] ERROR: Key type is NONE for keyID: %s", keyID)
		return nil, fmt.Errorf("no encryption for this key")
	}

	// Decode base64 key material
	keyBytes, err := base64.StdEncoding.DecodeString(metadata.KeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key material: %w", err)
	}

	// For AES256, we need exactly 32 bytes
	if metadata.Type == "AES256-GCM" && len(keyBytes) != 32 {
		return nil, fmt.Errorf("invalid key length for AES256: got %d, want 32", len(keyBytes))
	}

	return keyBytes, nil
}

func (p *FileKeyProvider) GetKeyForGuardPoint(guardPointID string) ([]byte, error) {
	log.Printf("[CRYPTO] Looking for key for guard point: %s", guardPointID)
	keyID, exists := p.guardPointMap[guardPointID]
	if !exists {
		log.Printf("[CRYPTO] ERROR: No key configured for guard point: %s", guardPointID)
		log.Printf("[CRYPTO] Available guard points: %v", p.guardPointMap)
		return nil, fmt.Errorf("no key configured for guard point: %s", guardPointID)
	}

	log.Printf("[CRYPTO] Found key ID: %s for guard point: %s", keyID, guardPointID)
	return p.GetKey(keyID)
}

func (p *FileKeyProvider) GetDefaultKey() ([]byte, error) {
	// For now, return error - we should always use guard point specific keys
	return nil, fmt.Errorf("default key not supported - use guard point specific keys")
}