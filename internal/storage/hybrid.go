// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"os"
	"sync"

	"github.com/oowada/google-workspace-mcp/internal/util"
)

// HybridStorage implements TokenStorage with keychain as primary and file as fallback
type HybridStorage struct {
	storage     TokenStorage
	storageType StorageType
	initOnce    sync.Once
	initErr     error
}

// NewHybridStorage creates a new HybridStorage instance
func NewHybridStorage(baseDir string) *HybridStorage {
	return &HybridStorage{}
}

// initStorage initializes the storage backend
func (h *HybridStorage) initStorage() error {
	h.initOnce.Do(func() {
		// Check for forced file storage
		if os.Getenv("GEMINI_CLI_WORKSPACE_FORCE_FILE_STORAGE") == "true" {
			util.LogDebug("Forcing file storage due to environment variable")
			h.storage, h.initErr = NewFileStorage("")
			if h.initErr == nil {
				h.storageType = StorageTypeFile
			}
			return
		}

		// Try keychain first
		keychain := NewKeychainStorage()
		if keychain.IsAvailable() {
			util.LogDebug("Using keychain storage")
			h.storage = keychain
			h.storageType = StorageTypeKeychain
			return
		}

		// Fall back to file storage
		util.LogDebug("Keychain not available, falling back to file storage")
		h.storage, h.initErr = NewFileStorage("")
		if h.initErr == nil {
			h.storageType = StorageTypeFile
		}
	})

	return h.initErr
}

// getStorage returns the initialized storage backend
func (h *HybridStorage) getStorage() (TokenStorage, error) {
	if err := h.initStorage(); err != nil {
		return nil, err
	}
	return h.storage, nil
}

// IsAvailable checks if any storage backend is available
func (h *HybridStorage) IsAvailable() bool {
	storage, err := h.getStorage()
	if err != nil {
		return false
	}
	return storage.IsAvailable()
}

// GetCredentials retrieves credentials from the storage backend
func (h *HybridStorage) GetCredentials(ctx context.Context, serverName string) (*OAuthCredentials, error) {
	storage, err := h.getStorage()
	if err != nil {
		return nil, err
	}
	return storage.GetCredentials(ctx, serverName)
}

// SetCredentials stores credentials in the storage backend
func (h *HybridStorage) SetCredentials(ctx context.Context, creds *OAuthCredentials) error {
	storage, err := h.getStorage()
	if err != nil {
		return err
	}
	return storage.SetCredentials(ctx, creds)
}

// DeleteCredentials removes credentials from the storage backend
func (h *HybridStorage) DeleteCredentials(ctx context.Context, serverName string) error {
	storage, err := h.getStorage()
	if err != nil {
		return err
	}
	return storage.DeleteCredentials(ctx, serverName)
}

// ClearAll removes all credentials from the storage backend
func (h *HybridStorage) ClearAll(ctx context.Context) error {
	storage, err := h.getStorage()
	if err != nil {
		return err
	}
	return storage.ClearAll(ctx)
}

// StorageType returns the type of storage backend being used
func (h *HybridStorage) StorageType() StorageType {
	_ = h.initStorage()
	return h.storageType
}
