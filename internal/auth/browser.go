// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/oowada/google-workspace-mcp/internal/util"
)

// ShouldLaunchBrowser determines if we should attempt to launch a browser
func ShouldLaunchBrowser() bool {
	// Check for explicit browser override
	if os.Getenv("BROWSER") == "www-browser" {
		util.LogDebug("BROWSER=www-browser, skipping browser launch")
		return false
	}

	// Check for CI/CD environment
	if os.Getenv("CI") != "" || os.Getenv("DEBIAN_FRONTEND") == "noninteractive" {
		util.LogDebug("CI/CD environment detected, skipping browser launch")
		return false
	}

	// Check for SSH session
	if os.Getenv("SSH_CONNECTION") != "" {
		if runtime.GOOS != "linux" {
			util.LogDebug("SSH session on non-Linux OS, skipping browser launch")
			return false
		}
		// On Linux, check for display server
		if !hasDisplayServer() {
			util.LogDebug("SSH session without display server, skipping browser launch")
			return false
		}
	}

	// On Linux, verify display server is available
	if runtime.GOOS == "linux" && !hasDisplayServer() {
		util.LogDebug("No display server detected on Linux, skipping browser launch")
		return false
	}

	return true
}

// hasDisplayServer checks if a display server is available on Linux
func hasDisplayServer() bool {
	return os.Getenv("DISPLAY") != "" ||
		os.Getenv("WAYLAND_DISPLAY") != "" ||
		os.Getenv("MIR_SOCKET") != ""
}

// OpenBrowser opens the specified URL in the default browser
func OpenBrowser(urlStr string) error {
	// Validate URL
	if err := validateURL(urlStr); err != nil {
		return err
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", urlStr)

	case "windows":
		// Use PowerShell to avoid cmd.exe security issues
		cmd = exec.Command("powershell.exe",
			"-NoProfile",
			"-NonInteractive",
			"-WindowStyle", "Hidden",
			"-Command", fmt.Sprintf("Start-Process '%s'", urlStr))

	default: // Linux, BSD, etc.
		// Try xdg-open first, then fallback to alternatives
		browsers := []string{"xdg-open", "gnome-open", "kde-open", "firefox", "chromium", "google-chrome"}
		for _, browser := range browsers {
			if path, err := exec.LookPath(browser); err == nil {
				cmd = exec.Command(path, urlStr)
				break
			}
		}
		if cmd == nil {
			return fmt.Errorf("no browser found")
		}
	}

	// Start detached (don't wait for browser to close)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// validateURL ensures the URL is safe to open
func validateURL(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTP/HTTPS
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (only http/https allowed)", scheme)
	}

	// Check for control characters
	for _, r := range urlStr {
		if r < 32 || r == 127 {
			return fmt.Errorf("URL contains control characters")
		}
	}

	return nil
}
