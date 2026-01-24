// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/oowada/google-workspace-mcp/internal/util"
	"golang.org/x/oauth2"
)

// OAuthResult contains the result of an OAuth callback
type OAuthResult struct {
	Token *oauth2.Token
	Error error
}

// StartCallbackServer starts a local HTTP server to receive OAuth callbacks
func StartCallbackServer(ctx context.Context, csrfToken string) (string, <-chan OAuthResult, func(), error) {
	// Determine host and port
	host := os.Getenv("OAUTH_CALLBACK_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 0 // Let OS assign a port
	if portStr := os.Getenv("OAUTH_CALLBACK_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// Create listener
	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://%s:%d/oauth2callback", host, actualPort)

	util.LogDebug("OAuth callback server listening on %s", callbackURL)

	resultCh := make(chan OAuthResult, 1)

	// Create HTTP handler
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		handleOAuthCallback(w, r, csrfToken, resultCh)
	})

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			util.LogError("Callback server error: %v", err)
		}
	}()

	// Cleanup function
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}

	return callbackURL, resultCh, cleanup, nil
}

// handleOAuthCallback handles the OAuth callback request
func handleOAuthCallback(w http.ResponseWriter, r *http.Request, csrfToken string, resultCh chan<- OAuthResult) {
	query := r.URL.Query()

	// Check for errors from OAuth provider
	if errCode := query.Get("error"); errCode != "" {
		errDesc := query.Get("error_description")
		if errDesc == "" {
			errDesc = errCode
		}
		resultCh <- OAuthResult{Error: fmt.Errorf("OAuth error: %s", errDesc)}
		http.Error(w, "Authentication failed: "+errDesc, http.StatusBadRequest)
		return
	}

	// Validate CSRF token
	state := query.Get("state")
	if state != csrfToken {
		util.LogWarn("CSRF token mismatch")
		resultCh <- OAuthResult{Error: fmt.Errorf("CSRF token mismatch")}
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Extract tokens from query parameters
	accessToken := query.Get("access_token")
	if accessToken == "" {
		resultCh <- OAuthResult{Error: fmt.Errorf("no access token in callback")}
		http.Error(w, "No access token received", http.StatusBadRequest)
		return
	}

	refreshToken := query.Get("refresh_token")
	tokenType := query.Get("token_type")
	if tokenType == "" {
		tokenType = "Bearer"
	}

	var expiry time.Time
	if expiryStr := query.Get("expiry_date"); expiryStr != "" {
		if expiryMs, err := strconv.ParseInt(expiryStr, 10, 64); err == nil {
			expiry = time.UnixMilli(expiryMs)
		}
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		Expiry:       expiry,
	}

	resultCh <- OAuthResult{Token: token}

	// Send success response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .container {
            text-align: center;
            padding: 2rem;
            background: rgba(255,255,255,0.1);
            border-radius: 16px;
            backdrop-filter: blur(10px);
        }
        h1 { margin: 0 0 1rem 0; }
        p { opacity: 0.9; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authentication Successful!</h1>
        <p>You can close this tab and return to the terminal.</p>
    </div>
</body>
</html>`)
}

// generateCSRFToken generates a random CSRF token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
