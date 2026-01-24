// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oowada/google-workspace-mcp/internal/services"
)

func (r *ToolRegistrar) registerAuthTools() {
	// auth.login
	r.server.AddTool(
		mcp.NewTool("auth.login",
			mcp.WithDescription("Initiates Google OAuth authentication. Opens a browser for user to sign in. Must be called before using other tools."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Check if already authenticated
			if r.auth.IsAuthenticated(ctx) {
				resp := services.JSONResponse(map[string]string{
					"status":  "success",
					"message": "Already authenticated",
				})
				return mcp.NewToolResultText(resp.Content[0].Text), nil
			}

			// Trigger login
			if _, err := r.auth.Login(ctx); err != nil {
				resp := services.ErrorResponse(err)
				return mcp.NewToolResultText(resp.Content[0].Text), nil
			}
			resp := services.JSONResponse(map[string]string{
				"status":  "success",
				"message": "Authentication successful",
			})
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// auth.status
	r.server.AddTool(
		mcp.NewTool("auth.status",
			mcp.WithDescription("Checks current authentication status without triggering login."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			authenticated := r.auth.IsAuthenticated(ctx)
			resp := services.JSONResponse(map[string]interface{}{
				"authenticated": authenticated,
				"message": func() string {
					if authenticated {
						return "Authenticated and ready"
					}
					return "Not authenticated - please use auth.login tool"
				}(),
			})
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// auth.clear
	r.server.AddTool(
		mcp.NewTool("auth.clear",
			mcp.WithDescription("Clears authentication credentials, forcing re-login on next operation."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := r.auth.ClearAuth(ctx); err != nil {
				resp := services.ErrorResponse(err)
				return mcp.NewToolResultText(resp.Content[0].Text), nil
			}
			resp := services.JSONResponse(map[string]string{
				"status":  "success",
				"message": "Authentication credentials cleared",
			})
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// auth.refreshToken
	r.server.AddTool(
		mcp.NewTool("auth.refreshToken",
			mcp.WithDescription("Manually triggers a token refresh process."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := r.auth.RefreshToken(ctx); err != nil {
				resp := services.ErrorResponse(err)
				return mcp.NewToolResultText(resp.Content[0].Text), nil
			}
			resp := services.JSONResponse(map[string]string{
				"status":  "success",
				"message": "Token refreshed successfully",
			})
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
