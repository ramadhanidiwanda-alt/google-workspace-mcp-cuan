// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"

	"golang.org/x/oauth2"
)

// OAuthCredentialStorage provides a high-level interface for storing OAuth credentials
type OAuthCredentialStorage struct {
	storage TokenStorage
}

// NewOAuthCredentialStorage creates a new OAuthCredentialStorage instance
func NewOAuthCredentialStorage() *OAuthCredentialStorage {
	return &OAuthCredentialStorage{
		storage: NewHybridStorage(""),
	}
}

// NewOAuthCredentialStorageWithBackend creates a new OAuthCredentialStorage with a specific backend
func NewOAuthCredentialStorageWithBackend(storage TokenStorage) *OAuthCredentialStorage {
	return &OAuthCredentialStorage{
		storage: storage,
	}
}

// GetToken retrieves the stored OAuth token
func (s *OAuthCredentialStorage) GetToken(ctx context.Context) (*oauth2.Token, string, error) {
	creds, err := s.storage.GetCredentials(ctx, DefaultServerName)
	if err != nil {
		return nil, "", err
	}
	if creds == nil {
		return nil, "", nil
	}

	return creds.Token.ToOAuth2Token(), creds.Token.Scope, nil
}

// SaveToken stores an OAuth token
func (s *OAuthCredentialStorage) SaveToken(ctx context.Context, token *oauth2.Token, scope string) error {
	creds := &OAuthCredentials{
		ServerName: DefaultServerName,
		Token:      FromOAuth2Token(token, scope),
	}
	return s.storage.SetCredentials(ctx, creds)
}

// ClearCredentials removes all stored credentials
func (s *OAuthCredentialStorage) ClearCredentials(ctx context.Context) error {
	return s.storage.ClearAll(ctx)
}

// IsAvailable checks if the storage is available
func (s *OAuthCredentialStorage) IsAvailable() bool {
	return s.storage.IsAvailable()
}
