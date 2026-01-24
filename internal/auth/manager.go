// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/oowada/google-workspace-mcp/internal/storage"
	"github.com/oowada/google-workspace-mcp/internal/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	// DefaultClientID is the OAuth 2.0 client ID for the extension (fallback)
	DefaultClientID = "338689075775-o75k922vn5fdl18qergr96rp8g63e4d7.apps.googleusercontent.com"

	// CloudFunctionURL is the URL of the cloud function that handles OAuth
	CloudFunctionURL = "https://google-workspace-extension.geminicli.com"

	// GoogleAuthURL is the Google OAuth 2.0 authorization endpoint
	GoogleAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"

	// GoogleTokenURL is the Google OAuth 2.0 token endpoint
	GoogleTokenURL = "https://oauth2.googleapis.com/token"

	// TokenExpiryBuffer is the time before expiry to trigger a refresh (5 minutes)
	TokenExpiryBuffer = 5 * time.Minute

	// AuthTimeout is the timeout for the browser auth flow
	AuthTimeout = 5 * time.Minute
)

// AuthManager handles OAuth2 authentication with Google
type AuthManager struct {
	scopes       []string
	token        *oauth2.Token
	scope        string
	tokenMu      sync.RWMutex
	storage      *storage.OAuthCredentialStorage
	clientID     string
	clientSecret string
	useLocalAuth bool
	fullAccess   bool
}

// NewAuthManager creates a new AuthManager instance
func NewAuthManager(basicScopes, fullScopes []string) *AuthManager {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	useLocalAuth := clientID != "" && clientSecret != ""

	var scopes []string
	var fullAccess bool

	if useLocalAuth {
		scopes = fullScopes
		fullAccess = true
		util.LogInfo("Full access mode (custom OAuth credentials)")
	} else {
		scopes = basicScopes
		fullAccess = false
		clientID = DefaultClientID
		util.LogInfo("Basic mode (read-only). Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET for full access.")
	}

	return &AuthManager{
		scopes:       scopes,
		storage:      storage.NewOAuthCredentialStorage(),
		clientID:     clientID,
		clientSecret: clientSecret,
		useLocalAuth: useLocalAuth,
		fullAccess:   fullAccess,
	}
}

// IsFullAccess returns whether full access mode is enabled
func (m *AuthManager) IsFullAccess() bool {
	return m.fullAccess
}

// ErrNotAuthenticated is returned when no valid credentials are available
var ErrNotAuthenticated = fmt.Errorf("not authenticated - please use auth.login tool first")

// GetAuthenticatedClient ensures we have valid credentials and returns the token
// Returns ErrNotAuthenticated if not authenticated (does NOT auto-trigger auth)
func (m *AuthManager) GetAuthenticatedClient(ctx context.Context) (*oauth2.Token, error) {
	m.tokenMu.RLock()
	if m.token != nil && !m.isTokenExpiringSoon() {
		defer m.tokenMu.RUnlock()
		return m.token, nil
	}
	m.tokenMu.RUnlock()

	// Try loading from storage
	if err := m.loadCachedCredentials(ctx); err == nil && m.token != nil {
		if !m.isTokenExpiringSoon() {
			return m.token, nil
		}

		// Token is expiring soon, try to refresh
		util.LogDebug("Token expiring soon, attempting refresh")
		if err := m.RefreshToken(ctx); err != nil {
			util.LogWarn("Token refresh failed: %v", err)
			// Don't auto re-authenticate, return error
			return nil, ErrNotAuthenticated
		}
		return m.token, nil
	}

	// Not authenticated - don't auto-trigger, return error
	return nil, ErrNotAuthenticated
}

// IsAuthenticated checks if we have valid credentials without triggering auth
func (m *AuthManager) IsAuthenticated(ctx context.Context) bool {
	m.tokenMu.RLock()
	if m.token != nil && !m.isTokenExpiringSoon() {
		m.tokenMu.RUnlock()
		return true
	}
	m.tokenMu.RUnlock()

	// Try loading from storage
	if err := m.loadCachedCredentials(ctx); err == nil && m.token != nil {
		return !m.isTokenExpiringSoon()
	}

	return false
}

// Login explicitly triggers browser-based OAuth authentication
func (m *AuthManager) Login(ctx context.Context) (*oauth2.Token, error) {
	return m.performWebAuth(ctx)
}

// GetClientOption returns a Google API client option with authentication
func (m *AuthManager) GetClientOption(ctx context.Context) (option.ClientOption, error) {
	token, err := m.GetAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	return option.WithTokenSource(oauth2.StaticTokenSource(token)), nil
}

// RefreshToken manually triggers a token refresh
func (m *AuthManager) RefreshToken(ctx context.Context) error {
	m.tokenMu.Lock()
	defer m.tokenMu.Unlock()

	if m.token == nil || m.token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	refreshToken := m.token.RefreshToken

	if m.useLocalAuth {
		// Use direct Google token refresh
		config := &oauth2.Config{
			ClientID:     m.clientID,
			ClientSecret: m.clientSecret,
			Endpoint:     google.Endpoint,
		}
		tokenSource := config.TokenSource(ctx, m.token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return fmt.Errorf("token refresh failed: %w", err)
		}

		m.token = newToken
		if m.token.RefreshToken == "" {
			m.token.RefreshToken = refreshToken
		}

		if err := m.storage.SaveToken(ctx, m.token, m.scope); err != nil {
			util.LogWarn("Failed to save refreshed token: %v", err)
		}

		util.LogDebug("Token refreshed successfully (local)")
		return nil
	}

	// Call cloud function to refresh token
	reqBody, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", CloudFunctionURL+"/refreshToken", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiryDate  int64  `json:"expiry_date"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode refresh response: %w", err)
	}

	m.token = &oauth2.Token{
		AccessToken:  result.AccessToken,
		RefreshToken: refreshToken,
		TokenType:    result.TokenType,
		Expiry:       time.UnixMilli(result.ExpiryDate),
	}
	m.scope = result.Scope

	if err := m.storage.SaveToken(ctx, m.token, m.scope); err != nil {
		util.LogWarn("Failed to save refreshed token: %v", err)
	}

	util.LogDebug("Token refreshed successfully (cloud)")
	return nil
}

// ClearAuth clears stored credentials
func (m *AuthManager) ClearAuth(ctx context.Context) error {
	m.tokenMu.Lock()
	defer m.tokenMu.Unlock()

	m.token = nil
	m.scope = ""

	return m.storage.ClearCredentials(ctx)
}

// isTokenExpiringSoon checks if the token will expire within the buffer time
func (m *AuthManager) isTokenExpiringSoon() bool {
	if m.token == nil || m.token.Expiry.IsZero() {
		return true
	}
	return time.Until(m.token.Expiry) < TokenExpiryBuffer
}

// loadCachedCredentials loads credentials from storage
func (m *AuthManager) loadCachedCredentials(ctx context.Context) error {
	token, scope, err := m.storage.GetToken(ctx)
	if err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("no cached credentials")
	}

	// Validate scopes
	if !m.hasRequiredScopes(scope) {
		util.LogDebug("Cached token missing required scopes, clearing")
		_ = m.storage.ClearCredentials(ctx)
		return fmt.Errorf("cached token missing required scopes")
	}

	m.tokenMu.Lock()
	m.token = token
	m.scope = scope
	m.tokenMu.Unlock()

	util.LogDebug("Loaded cached credentials")
	return nil
}

// hasRequiredScopes checks if the token has all required scopes
func (m *AuthManager) hasRequiredScopes(tokenScopes string) bool {
	scopeSet := make(map[string]bool)
	for _, s := range strings.Fields(tokenScopes) {
		scopeSet[s] = true
	}

	for _, required := range m.scopes {
		if !scopeSet[required] {
			return false
		}
	}
	return true
}

// performWebAuth performs browser-based OAuth authentication
func (m *AuthManager) performWebAuth(ctx context.Context) (*oauth2.Token, error) {
	if m.useLocalAuth {
		return m.performLocalAuth(ctx)
	}
	return m.performCloudAuth(ctx)
}

// performLocalAuth performs OAuth using local credentials
func (m *AuthManager) performLocalAuth(ctx context.Context) (*oauth2.Token, error) {
	util.LogInfo("Starting local OAuth authentication...")

	ctx, cancel := context.WithTimeout(ctx, AuthTimeout)
	defer cancel()

	// Start local callback server
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Find available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	// OAuth2 config
	config := &oauth2.Config{
		ClientID:     m.clientID,
		ClientSecret: m.clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
		Scopes:       m.scopes,
	}

	// Generate state
	state, _ := generateCSRFToken()

	// Start HTTP server for callback
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("auth error: %s", errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code received")
			http.Error(w, "No code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h1>Authentication successful!</h1><p>You can close this window.</p></body></html>"))
		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Close()

	// Build auth URL and open browser
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	util.LogInfo("Opening browser for authentication...")
	if err := OpenBrowser(authURL); err != nil {
		util.LogWarn("Failed to open browser: %v", err)
		util.LogInfo("Please open the following URL manually:")
		fmt.Println(authURL)
	}

	// Wait for callback
	select {
	case code := <-codeCh:
		// Exchange code for token
		token, err := config.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange code: %w", err)
		}

		m.tokenMu.Lock()
		m.token = token
		m.scope = strings.Join(m.scopes, " ")
		m.tokenMu.Unlock()

		if err := m.storage.SaveToken(ctx, token, m.scope); err != nil {
			util.LogWarn("Failed to save token: %v", err)
		}

		util.LogInfo("Authentication successful!")
		return token, nil

	case err := <-errCh:
		return nil, err

	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timed out")
	}
}

// performCloudAuth performs OAuth using cloud function (default mode)
func (m *AuthManager) performCloudAuth(ctx context.Context) (*oauth2.Token, error) {
	util.LogInfo("Starting cloud OAuth authentication...")

	ctx, cancel := context.WithTimeout(ctx, AuthTimeout)
	defer cancel()

	csrfToken, err := generateCSRFToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	callbackURL, resultCh, cleanup, err := StartCallbackServer(ctx, csrfToken)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer cleanup()

	stateData := map[string]interface{}{
		"csrf": csrfToken,
	}

	if ShouldLaunchBrowser() {
		stateData["uri"] = callbackURL
		stateData["manual"] = false
	} else {
		stateData["manual"] = true
	}

	stateJSON, _ := json.Marshal(stateData)
	state := base64.StdEncoding.EncodeToString(stateJSON)

	authURL := buildAuthURL(m.scopes, state)

	if ShouldLaunchBrowser() {
		util.LogInfo("Opening browser for authentication...")
		if err := OpenBrowser(authURL); err != nil {
			util.LogWarn("Failed to open browser: %v", err)
			util.LogInfo("Please open the following URL manually:")
			fmt.Println(authURL)
		}
	} else {
		util.LogInfo("Please open the following URL in a browser:")
		fmt.Println(authURL)
	}

	select {
	case result := <-resultCh:
		if result.Error != nil {
			return nil, result.Error
		}

		m.tokenMu.Lock()
		m.token = result.Token
		m.scope = strings.Join(m.scopes, " ")
		m.tokenMu.Unlock()

		if err := m.storage.SaveToken(ctx, result.Token, m.scope); err != nil {
			util.LogWarn("Failed to save token: %v", err)
		}

		util.LogInfo("Authentication successful!")
		return result.Token, nil

	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timed out")
	}
}

// buildAuthURL builds the Google OAuth authorization URL for cloud auth
func buildAuthURL(scopes []string, state string) string {
	params := url.Values{
		"client_id":     {DefaultClientID},
		"redirect_uri":  {CloudFunctionURL},
		"response_type": {"code"},
		"scope":         {strings.Join(scopes, " ")},
		"state":         {state},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}

	return GoogleAuthURL + "?" + params.Encode()
}
