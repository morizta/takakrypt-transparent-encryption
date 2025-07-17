package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type Service struct {
	keyProvider KeyProvider
}

type KeyProvider interface {
	GetKey(keyID string) ([]byte, error)
	GetDefaultKey() ([]byte, error)
	GetKeyForGuardPoint(guardPointID string) ([]byte, error)
}

type LocalKeyProvider struct {
	defaultKey []byte
}

func NewLocalKeyProvider(key []byte) *LocalKeyProvider {
	return &LocalKeyProvider{
		defaultKey: key,
	}
}

func (p *LocalKeyProvider) GetKey(keyID string) ([]byte, error) {
	return p.defaultKey, nil
}

func (p *LocalKeyProvider) GetDefaultKey() ([]byte, error) {
	return p.defaultKey, nil
}

func (p *LocalKeyProvider) GetKeyForGuardPoint(guardPointID string) ([]byte, error) {
	// LocalKeyProvider doesn't support guard point specific keys
	return p.defaultKey, nil
}

func NewService(keyProvider KeyProvider) *Service {
	return &Service{
		keyProvider: keyProvider,
	}
}

func (s *Service) Encrypt(plaintext []byte) ([]byte, error) {
	key, err := s.keyProvider.GetDefaultKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *Service) Decrypt(ciphertext []byte) ([]byte, error) {
	key, err := s.keyProvider.GetDefaultKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get decryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (s *Service) EncryptForGuardPoint(plaintext []byte, guardPointID string) ([]byte, error) {
	key, err := s.keyProvider.GetKeyForGuardPoint(guardPointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key for guard point %s: %w", guardPointID, err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *Service) DecryptForGuardPoint(ciphertext []byte, guardPointID string) ([]byte, error) {
	key, err := s.keyProvider.GetKeyForGuardPoint(guardPointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get decryption key for guard point %s: %w", guardPointID, err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}