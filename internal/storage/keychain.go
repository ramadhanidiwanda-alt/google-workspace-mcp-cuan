// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/oowada/google-workspace-mcp/internal/util"
	"github.com/zalando/go-keyring"
)

const (
	keychainServiceName = "gemini-cli-workspace-oauth"
	testAccountPrefix   = "__keychain_test__"
)

// KeychainStorage implements TokenStorage using the system keychain
type KeychainStorage struct {
	available     bool
	availableOnce sync.Once
}

// NewKeychainStorage creates a new KeychainStorage instance
func NewKeychainStorage() *KeychainStorage {
	return &KeychainStorage{}
}

// IsAvailable checks if the keychain is available on this system
func (s *KeychainStorage) IsAvailable() bool {
	s.availableOnce.Do(func() {
		testKey := testAccountPrefix + time.Now().Format("20060102150405")

		// Try to set a test value
		err := keyring.Set(keychainServiceName, testKey, "test")
		if err != nil {
			util.LogDebug("Keychain not available: %v", err)
			s.available = false
			return
		}

		// Try to get it back
		_, err = keyring.Get(keychainServiceName, testKey)
		if err != nil {
			util.LogDebug("Keychain read failed: %v", err)
			s.available = false
			return
		}

		// Clean up
		_ = keyring.Delete(keychainServiceName, testKey)
		s.available = true
		util.LogDebug("Keychain is available")
	})

	return s.available
}

// GetCredentials retrieves credentials from the keychain
func (s *KeychainStorage) GetCredentials(ctx context.Context, serverName string) (*OAuthCredentials, error) {
	if !s.IsAvailable() {
		return nil, ErrStorageNotAvailable
	}

	data, err := keyring.Get(keychainServiceName, sanitizeServerName(serverName))
	if err == keyring.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var creds OAuthCredentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// SetCredentials stores credentials in the keychain
func (s *KeychainStorage) SetCredentials(ctx context.Context, creds *OAuthCredentials) error {
	if !s.IsAvailable() {
		return ErrStorageNotAvailable
	}

	creds.UpdatedAt = time.Now().UnixMilli()

	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	return keyring.Set(keychainServiceName, sanitizeServerName(creds.ServerName), string(data))
}

// DeleteCredentials removes credentials from the keychain
func (s *KeychainStorage) DeleteCredentials(ctx context.Context, serverName string) error {
	if !s.IsAvailable() {
		return ErrStorageNotAvailable
	}

	err := keyring.Delete(keychainServiceName, sanitizeServerName(serverName))
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

// ClearAll removes all credentials (note: keychain doesn't support listing, so we only delete known keys)
func (s *KeychainStorage) ClearAll(ctx context.Context) error {
	return s.DeleteCredentials(ctx, DefaultServerName)
}

// sanitizeServerName replaces invalid characters in server names
func sanitizeServerName(name string) string {
	// Replace characters that might cause issues with keychain
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
