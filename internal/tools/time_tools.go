// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerTimeTools() {
	// time.getCurrentDate
	r.server.AddTool(
		mcp.NewTool("time.getCurrentDate",
			mcp.WithDescription("Gets the current date in both UTC and local timezone."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Time.GetCurrentDate()
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// time.getCurrentTime
	r.server.AddTool(
		mcp.NewTool("time.getCurrentTime",
			mcp.WithDescription("Gets the current time in both UTC and local timezone with timezone info."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Time.GetCurrentTime()
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// time.getTimeZone
	r.server.AddTool(
		mcp.NewTool("time.getTimeZone",
			mcp.WithDescription("Gets the local timezone name and offset from UTC."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Time.GetTimeZone()
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
