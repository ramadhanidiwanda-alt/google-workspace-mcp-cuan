// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// AuthProvider provides authenticated HTTP clients for Google APIs
type AuthProvider interface {
	// GetAuthenticatedClient returns the current OAuth token
	GetAuthenticatedClient(ctx context.Context) (*oauth2.Token, error)

	// GetClientOption returns a Google API client option with authentication
	GetClientOption(ctx context.Context) (option.ClientOption, error)

	// RefreshToken manually triggers a token refresh
	RefreshToken(ctx context.Context) error

	// ClearAuth clears stored credentials
	ClearAuth(ctx context.Context) error
}

// ToolResponse is the standard MCP tool response format
type ToolResponse struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content block in the response
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// TextResponse creates a successful text response
func TextResponse(data interface{}) ToolResponse {
	var text string
	switch v := data.(type) {
	case string:
		text = v
	case []byte:
		text = string(v)
	default:
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			text = fmt.Sprintf("%v", data)
		} else {
			text = string(jsonBytes)
		}
	}

	return ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

// ErrorResponse creates an error response
func ErrorResponse(err error) ToolResponse {
	return ToolResponse{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf(`{"error": %q}`, err.Error()),
		}},
	}
}

// JSONResponse creates a JSON response from any data
func JSONResponse(data interface{}) ToolResponse {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ErrorResponse(err)
	}
	return ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: string(jsonBytes)}},
	}
}
