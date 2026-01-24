// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerSlidesTools() {
	// slides.getText
	r.server.AddTool(
		mcp.NewTool("slides.getText",
			mcp.WithDescription("Retrieves text content from all slides in a presentation."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			resp := r.services.Slides.GetText(ctx, presentationID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.find
	r.server.AddTool(
		mcp.NewTool("slides.find",
			mcp.WithDescription("Searches for presentations by title."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			query := args["query"].(string)
			var pageToken *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Slides.Find(ctx, query, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.getMetadata
	r.server.AddTool(
		mcp.NewTool("slides.getMetadata",
			mcp.WithDescription("Gets presentation metadata (slide count, dimensions, etc.)."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			resp := r.services.Slides.GetMetadata(ctx, presentationID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
