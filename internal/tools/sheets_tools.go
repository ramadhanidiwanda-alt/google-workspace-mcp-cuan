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

	// sheets.create
	r.server.AddTool(
		mcp.NewTool("sheets.create",
			mcp.WithDescription("Creates a new spreadsheet."),
			mcp.WithString("title", mcp.Required(), mcp.Description("The title of the spreadsheet")),
			mcp.WithArray("sheetTitles", mcp.Description("Optional list of sheet names to create")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			title := args["title"].(string)
			var sheetTitles []string
			if v, ok := args["sheetTitles"].([]interface{}); ok {
				for _, s := range v {
					if str, ok := s.(string); ok {
						sheetTitles = append(sheetTitles, str)
					}
				}
			}
			resp := r.services.Sheets.Create(ctx, title, sheetTitles)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.updateRange
	r.server.AddTool(
		mcp.NewTool("sheets.updateRange",
			mcp.WithDescription("Updates values in a specific range."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithString("range", mcp.Required(), mcp.Description("Range in A1 notation (e.g., Sheet1!A1:D10)")),
			mcp.WithArray("values", mcp.Required(), mcp.Description("2D array of values to write")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			rangeStr := args["range"].(string)
			rawValues := args["values"].([]interface{})
			values := make([][]interface{}, len(rawValues))
			for i, row := range rawValues {
				if r, ok := row.([]interface{}); ok {
					values[i] = r
				}
			}
			resp := r.services.Sheets.UpdateRange(ctx, spreadsheetID, rangeStr, values)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.appendRows
	r.server.AddTool(
		mcp.NewTool("sheets.appendRows",
			mcp.WithDescription("Appends rows to the end of a sheet."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithString("range", mcp.Required(), mcp.Description("Range in A1 notation (e.g., Sheet1!A:Z)")),
			mcp.WithArray("values", mcp.Required(), mcp.Description("2D array of rows to append")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			rangeStr := args["range"].(string)
			rawValues := args["values"].([]interface{})
			values := make([][]interface{}, len(rawValues))
			for i, row := range rawValues {
				if r, ok := row.([]interface{}); ok {
					values[i] = r
				}
			}
			resp := r.services.Sheets.AppendRows(ctx, spreadsheetID, rangeStr, values)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.clearRange
	r.server.AddTool(
		mcp.NewTool("sheets.clearRange",
			mcp.WithDescription("Clears values in a range."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithString("range", mcp.Required(), mcp.Description("Range in A1 notation to clear")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			rangeStr := args["range"].(string)
			resp := r.services.Sheets.ClearRange(ctx, spreadsheetID, rangeStr)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.createSheet
	r.server.AddTool(
		mcp.NewTool("sheets.createSheet",
			mcp.WithDescription("Creates a new sheet tab in a spreadsheet."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithString("sheetTitle", mcp.Required(), mcp.Description("The title of the new sheet")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			sheetTitle := args["sheetTitle"].(string)
			resp := r.services.Sheets.CreateSheet(ctx, spreadsheetID, sheetTitle)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.deleteSheet
	r.server.AddTool(
		mcp.NewTool("sheets.deleteSheet",
			mcp.WithDescription("Deletes a sheet tab from a spreadsheet."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithNumber("sheetId", mcp.Required(), mcp.Description("The ID of the sheet to delete")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			sheetID := int64(args["sheetId"].(float64))
			resp := r.services.Sheets.DeleteSheet(ctx, spreadsheetID, sheetID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.addChart
	r.server.AddTool(
		mcp.NewTool("sheets.addChart",
			mcp.WithDescription("Adds a chart to a sheet. Supports BAR, LINE, PIE, and COLUMN chart types."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithNumber("sheetId", mcp.Required(), mcp.Description("The ID of the sheet")),
			mcp.WithString("chartType", mcp.Required(), mcp.Description("Chart type: BAR, LINE, PIE, or COLUMN")),
			mcp.WithString("dataRange", mcp.Required(), mcp.Description("Data range in A1 notation (e.g., A1:B10)")),
			mcp.WithString("title", mcp.Description("Chart title")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			sheetID := int64(args["sheetId"].(float64))
			chartType := args["chartType"].(string)
			dataRange := args["dataRange"].(string)
			var title *string
			if v, ok := args["title"].(string); ok && v != "" {
				title = &v
			}
			resp := r.services.Sheets.AddChart(ctx, spreadsheetID, sheetID, chartType, dataRange, title)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// sheets.addConditionalFormatting
	r.server.AddTool(
		mcp.NewTool("sheets.addConditionalFormatting",
			mcp.WithDescription("Adds conditional formatting to a sheet. Rule types: NOT_BLANK, BLANK, TEXT_CONTAINS, NUMBER_GREATER, NUMBER_LESS."),
			mcp.WithString("spreadsheetId", mcp.Required(), mcp.Description("The ID of the spreadsheet")),
			mcp.WithNumber("sheetId", mcp.Required(), mcp.Description("The ID of the sheet")),
			mcp.WithString("range", mcp.Required(), mcp.Description("Range in A1 notation")),
			mcp.WithString("ruleType", mcp.Required(), mcp.Description("Rule type: NOT_BLANK, BLANK, TEXT_CONTAINS, NUMBER_GREATER, NUMBER_LESS")),
			mcp.WithString("value", mcp.Description("Value for comparison (required for TEXT_CONTAINS, NUMBER_GREATER, NUMBER_LESS)")),
			mcp.WithString("bgColor", mcp.Description("Background color in hex format (e.g., #FF0000)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spreadsheetID := args["spreadsheetId"].(string)
			sheetID := int64(args["sheetId"].(float64))
			rangeA1 := args["range"].(string)
			ruleType := args["ruleType"].(string)
			var value, bgColor *string
			if v, ok := args["value"].(string); ok && v != "" {
				value = &v
			}
			if v, ok := args["bgColor"].(string); ok && v != "" {
				bgColor = &v
			}
			resp := r.services.Sheets.AddConditionalFormatting(ctx, spreadsheetID, sheetID, rangeA1, ruleType, value, bgColor)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
