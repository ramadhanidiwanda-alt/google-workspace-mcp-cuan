// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package storage

import "errors"

var (
	// ErrStorageNotAvailable is returned when the storage backend is not available
	ErrStorageNotAvailable = errors.New("storage not available")

	// ErrCredentialsNotFound is returned when credentials are not found
	ErrCredentialsNotFound = errors.New("credentials not found")
)

// DefaultServerName is the default server name used for storing credentials
const DefaultServerName = "main-account"
