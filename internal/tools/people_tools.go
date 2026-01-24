// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerPeopleTools() {
	// people.getUserProfile
	r.server.AddTool(
		mcp.NewTool("people.getUserProfile",
			mcp.WithDescription("Gets a user's profile by ID, email, or name."),
			mcp.WithString("identifier", mcp.Required(), mcp.Description("User ID, email address, or name to search for")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			identifier := args["identifier"].(string)
			resp := r.services.People.GetUserProfile(ctx, identifier)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.getMe
	r.server.AddTool(
		mcp.NewTool("people.getMe",
			mcp.WithDescription("Gets the authenticated user's profile."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.People.GetMe(ctx)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.getUserRelations
	r.server.AddTool(
		mcp.NewTool("people.getUserRelations",
			mcp.WithDescription("Gets a user's relations (manager, spouse, assistant, etc.)."),
			mcp.WithString("identifier", mcp.Required(), mcp.Description("User ID, email, name, or 'me'")),
			mcp.WithString("relationType", mcp.Description("Filter by relation type (e.g., manager, spouse)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			identifier := args["identifier"].(string)
			var relationType *string
			if v, ok := args["relationType"].(string); ok && v != "" {
				relationType = &v
			}
			resp := r.services.People.GetUserRelations(ctx, identifier, relationType)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
