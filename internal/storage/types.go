// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

// OAuthToken represents stored OAuth token data
type OAuthToken struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"` // Unix milliseconds
	TokenType    string `json:"tokenType"`
	Scope        string `json:"scope,omitempty"`
}

// OAuthCredentials represents stored OAuth credentials
type OAuthCredentials struct {
	ServerName string     `json:"serverName"`
	Token      OAuthToken `json:"token"`
	UpdatedAt  int64      `json:"updatedAt"` // Unix milliseconds
}

// ToOAuth2Token converts OAuthToken to oauth2.Token
func (t *OAuthToken) ToOAuth2Token() *oauth2.Token {
	var expiry time.Time
	if t.ExpiresAt > 0 {
		expiry = time.UnixMilli(t.ExpiresAt)
	}

	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       expiry,
	}
}

// FromOAuth2Token converts oauth2.Token to OAuthToken
func FromOAuth2Token(token *oauth2.Token, scope string) OAuthToken {
	var expiresAt int64
	if !token.Expiry.IsZero() {
		expiresAt = token.Expiry.UnixMilli()
	}

	return OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    token.TokenType,
		Scope:        scope,
	}
}

// TokenStorage defines the interface for token storage backends
type TokenStorage interface {
	// GetCredentials retrieves stored credentials for a server
	GetCredentials(ctx context.Context, serverName string) (*OAuthCredentials, error)

	// SetCredentials stores credentials for a server
	SetCredentials(ctx context.Context, credentials *OAuthCredentials) error

	// DeleteCredentials removes credentials for a server
	DeleteCredentials(ctx context.Context, serverName string) error

	// ClearAll removes all stored credentials
	ClearAll(ctx context.Context) error

	// IsAvailable checks if the storage backend is available
	IsAvailable() bool
}

// StorageType indicates the type of storage backend
type StorageType int

const (
	StorageTypeKeychain StorageType = iota
	StorageTypeFile
)

func (s StorageType) String() string {
	switch s {
	case StorageTypeKeychain:
		return "keychain"
	case StorageTypeFile:
		return "file"
	default:
		return "unknown"
	}
}
