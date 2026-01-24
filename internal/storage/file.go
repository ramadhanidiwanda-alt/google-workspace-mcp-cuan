// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oowada/google-workspace-mcp/internal/util"
	"golang.org/x/crypto/scrypt"
)

const (
	tokenFileName     = "gemini-cli-workspace-token.json"
	masterKeyFileName = ".gemini-cli-workspace-master-key"
	filePermission    = 0600
	keyLength         = 32
)

// FileStorage implements TokenStorage using an encrypted file
type FileStorage struct {
	baseDir       string
	tokenFilePath string
	encryptionKey []byte
	mu            sync.RWMutex
}

// NewFileStorage creates a new FileStorage instance
func NewFileStorage(baseDir string) (*FileStorage, error) {
	if baseDir == "" {
		var err error
		baseDir, err = getDefaultBaseDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get base directory: %w", err)
		}
	}

	masterKeyPath := filepath.Join(baseDir, masterKeyFileName)
	tokenPath := filepath.Join(baseDir, tokenFileName)

	// Load or create master key
	masterKey, err := loadOrCreateMasterKey(masterKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load master key: %w", err)
	}

	// Derive encryption key using scrypt
	encryptionKey, err := deriveKey(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	return &FileStorage{
		baseDir:       baseDir,
		tokenFilePath: tokenPath,
		encryptionKey: encryptionKey,
	}, nil
}

// IsAvailable always returns true for file storage
func (s *FileStorage) IsAvailable() bool {
	return true
}

// GetCredentials retrieves credentials from the encrypted file
func (s *FileStorage) GetCredentials(ctx context.Context, serverName string) (*OAuthCredentials, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.readAndDecrypt()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	creds, ok := data[serverName]
	if !ok {
		return nil, nil
	}

	return &creds, nil
}

// SetCredentials stores credentials in the encrypted file
func (s *FileStorage) SetCredentials(ctx context.Context, creds *OAuthCredentials) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing data
	data, err := s.readAndDecrypt()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if data == nil {
		data = make(map[string]OAuthCredentials)
	}

	// Update credentials
	creds.UpdatedAt = time.Now().UnixMilli()
	data[creds.ServerName] = *creds

	// Encrypt and write
	return s.encryptAndWrite(data)
}

// DeleteCredentials removes credentials from the encrypted file
func (s *FileStorage) DeleteCredentials(ctx context.Context, serverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readAndDecrypt()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	delete(data, serverName)

	if len(data) == 0 {
		return os.Remove(s.tokenFilePath)
	}

	return s.encryptAndWrite(data)
}

// ClearAll removes the token file
func (s *FileStorage) ClearAll(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.tokenFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// readAndDecrypt reads and decrypts the token file
func (s *FileStorage) readAndDecrypt() (map[string]OAuthCredentials, error) {
	encryptedData, err := os.ReadFile(s.tokenFilePath)
	if err != nil {
		return nil, err
	}

	plaintext, err := s.decrypt(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	var data map[string]OAuthCredentials
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return data, nil
}

// encryptAndWrite encrypts and writes data to the token file
func (s *FileStorage) encryptAndWrite(data map[string]OAuthCredentials) error {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return err
	}

	encrypted, err := s.encrypt(plaintext)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.tokenFilePath), 0700); err != nil {
		return err
	}

	return os.WriteFile(s.tokenFilePath, []byte(encrypted), filePermission)
}

// encrypt encrypts plaintext using AES-256-GCM
func (s *FileStorage) encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Format: nonce:authTag:ciphertext (all hex encoded)
	// The auth tag is the last gcm.Overhead() bytes of ciphertext
	overhead := gcm.Overhead()
	authTag := ciphertext[len(ciphertext)-overhead:]
	encryptedData := ciphertext[:len(ciphertext)-overhead]

	return fmt.Sprintf("%s:%s:%s",
		hex.EncodeToString(nonce),
		hex.EncodeToString(authTag),
		hex.EncodeToString(encryptedData),
	), nil
}

// decrypt decrypts data encrypted with encrypt()
func (s *FileStorage) decrypt(encryptedStr string) ([]byte, error) {
	parts := strings.Split(encryptedStr, ":")
	if len(parts) != 3 {
		return nil, errors.New("invalid encrypted data format")
	}

	nonce, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	authTag, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid auth tag: %w", err)
	}

	ciphertext, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %w", err)
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Reconstruct ciphertext with auth tag
	fullCiphertext := append(ciphertext, authTag...)

	return gcm.Open(nil, nonce, fullCiphertext, nil)
}

// loadOrCreateMasterKey loads or creates the master encryption key
func loadOrCreateMasterKey(path string) ([]byte, error) {
	// Try to read existing key
	data, err := os.ReadFile(path)
	if err == nil {
		key, err := hex.DecodeString(strings.TrimSpace(string(data)))
		if err == nil && len(key) == keyLength {
			return key, nil
		}
		util.LogWarn("Invalid master key file, regenerating")
	}

	// Generate new key
	key := make([]byte, keyLength)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	// Write key with restricted permissions
	if err := os.WriteFile(path, []byte(hex.EncodeToString(key)), filePermission); err != nil {
		return nil, err
	}

	util.LogDebug("Created new master key at %s", path)
	return key, nil
}

// deriveKey derives an encryption key from the master key using scrypt
func deriveKey(masterKey []byte) ([]byte, error) {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	salt := fmt.Sprintf("%s-%s-gemini-cli-workspace", hostname, username)

	// scrypt parameters: N=32768, r=8, p=1, keyLen=32
	return scrypt.Key(masterKey, []byte(salt), 32768, 8, 1, keyLength)
}

// getDefaultBaseDir returns the default base directory for storage
func getDefaultBaseDir() (string, error) {
	// Try to find project root by looking for go.mod or gemini-extension.json
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "gemini-extension.json")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, use current directory
			return cwd, nil
		}
		dir = parent
	}
}
