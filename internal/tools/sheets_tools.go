// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerSheetsTools() {
	// sheets.getText
	r.server.AddTool(
		mcp.NewTool("sheets.getText",
			mcp.WithDescription("Retrieves spreadsheet content in text, CSV, or JSON format."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithString("sheetName", mcp.Description("Name of specific sheet (defaults to first)")),
			mcp.WithString("format", mcp.Description("Output format: text, csv, or json")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			var sheetName, format *string
			if v, ok := args["sheetName"].(string); ok && v != "" {
				sheetName = &v
			}
			if v, ok := args["format"].(string); ok && v != "" {
				format = &v
			}
			resp := r.services.Sheets.GetText(ctx, spreadsheetID, sheetName, format)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.getRange
	r.server.AddTool(
		mcp.NewTool("sheets.getRange",
			mcp.WithDescription("Gets values from a specific range using A1 notation."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithString("range", mcp.Required(), mcp.Description("Range in A1 notation (e.g., Sheet1!A1:D10)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			rangeStr := args["range"].(string)
			resp := r.services.Sheets.GetRange(ctx, spreadsheetID, rangeStr)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.find
	r.server.AddTool(
		mcp.NewTool("sheets.find",
			mcp.WithDescription("Searches for spreadsheets by title."),
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
			resp := r.services.Sheets.Find(ctx, query, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.getMetadata
	r.server.AddTool(
		mcp.NewTool("sheets.getMetadata",
			mcp.WithDescription("Gets spreadsheet metadata (sheets, dimensions, timezone)."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			resp := r.services.Sheets.GetMetadata(ctx, spreadsheetID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
