// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	debugEnabled bool
	debugMu      sync.RWMutex
	logger       *log.Logger
)

func init() {
	logger = log.New(os.Stderr, "", log.LstdFlags)
}

// EnableDebug enables debug logging
func EnableDebug() {
	debugMu.Lock()
	defer debugMu.Unlock()
	debugEnabled = true
}

// IsDebugEnabled returns whether debug logging is enabled
func IsDebugEnabled() bool {
	debugMu.RLock()
	defer debugMu.RUnlock()
	return debugEnabled
}

// LogDebug logs a debug message (only if debug is enabled)
func LogDebug(format string, args ...interface{}) {
	if IsDebugEnabled() {
		logger.Printf("[DEBUG] "+format, args...)
	}
}

// LogInfo logs an info message
func LogInfo(format string, args ...interface{}) {
	logger.Printf("[INFO] "+format, args...)
}

// LogError logs an error message
func LogError(format string, args ...interface{}) {
	logger.Printf("[ERROR] "+format, args...)
}

// LogWarn logs a warning message
func LogWarn(format string, args ...interface{}) {
	logger.Printf("[WARN] "+format, args...)
}

// FormatError formats an error for consistent JSON responses
func FormatError(err error) string {
	return fmt.Sprintf(`{"error": %q}`, err.Error())
}
