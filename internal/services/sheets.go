// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

// SheetsService provides Google Sheets operations
type SheetsService struct {
	auth AuthProvider
}

// NewSheetsService creates a new SheetsService instance
func NewSheetsService(auth AuthProvider) *SheetsService {
	return &SheetsService{auth: auth}
}

// getSheetsClient returns an authenticated Sheets client
func (s *SheetsService) getSheetsClient(ctx context.Context) (*sheets.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return sheets.NewService(ctx, opt)
}

// getDriveClient returns an authenticated Drive client
func (s *SheetsService) getDriveClient(ctx context.Context) (*drive.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return drive.NewService(ctx, opt)
}

// GetTextInput contains input for GetText
type GetTextInput struct {
	SpreadsheetID string  `json:"spreadsheetId"`
	SheetName     *string `json:"sheetName,omitempty"`
	Format        *string `json:"format,omitempty"` // "text", "csv", "json"
}

// GetText retrieves spreadsheet content
func (s *SheetsService) GetText(ctx context.Context, spreadsheetID string, sheetName *string, format *string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Get spreadsheet metadata first
	spreadsheet, err := client.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Determine which sheet to read
	var targetSheet string
	if sheetName != nil && *sheetName != "" {
		targetSheet = *sheetName
	} else if len(spreadsheet.Sheets) > 0 {
		targetSheet = spreadsheet.Sheets[0].Properties.Title
	} else {
		return ErrorResponse(fmt.Errorf("spreadsheet has no sheets"))
	}

	// Get values
	values, err := client.Spreadsheets.Values.Get(spreadsheetID, targetSheet).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Format output
	outputFormat := "text"
	if format != nil && *format != "" {
		outputFormat = *format
	}

	var content interface{}
	switch outputFormat {
	case "csv":
		content = formatAsCSV(values.Values)
	case "json":
		content = formatAsJSON(values.Values)
	default: // "text"
		content = formatAsText(values.Values)
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId": spreadsheet.SpreadsheetId,
		"title":         spreadsheet.Properties.Title,
		"sheetName":     targetSheet,
		"format":        outputFormat,
		"content":       content,
	})
}

// GetRange gets values from a specific range
func (s *SheetsService) GetRange(ctx context.Context, spreadsheetID string, rangeA1 string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	values, err := client.Spreadsheets.Values.Get(spreadsheetID, rangeA1).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId": spreadsheetID,
		"range":         values.Range,
		"values":        values.Values,
	})
}

// Find searches for spreadsheets
func (s *SheetsService) Find(ctx context.Context, query string, pageToken *string, pageSize *int) ToolResponse {
	driveClient, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	driveQuery := fmt.Sprintf("name contains '%s' and mimeType = 'application/vnd.google-apps.spreadsheet' and trashed = false",
		escapeQuery(query))

	req := driveClient.Files.List().
		Q(driveQuery).
		Fields("nextPageToken, files(id, name, modifiedTime, webViewLink)")

	if pageSize != nil {
		req.PageSize(int64(*pageSize))
	} else {
		req.PageSize(20)
	}

	if pageToken != nil && *pageToken != "" {
		req.PageToken(*pageToken)
	}

	result, err := req.Do()
	if err != nil {
		return ErrorResponse(err)
	}

	files := make([]map[string]interface{}, len(result.Files))
	for i, f := range result.Files {
		files[i] = map[string]interface{}{
			"id":           f.Id,
			"name":         f.Name,
			"modifiedTime": f.ModifiedTime,
			"webViewLink":  f.WebViewLink,
		}
	}

	response := map[string]interface{}{
		"spreadsheets": files,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// GetMetadata gets spreadsheet metadata
func (s *SheetsService) GetMetadata(ctx context.Context, spreadsheetID string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	spreadsheet, err := client.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	sheetsInfo := make([]map[string]interface{}, len(spreadsheet.Sheets))
	for i, sheet := range spreadsheet.Sheets {
		props := sheet.Properties
		sheetsInfo[i] = map[string]interface{}{
			"sheetId":    props.SheetId,
			"title":      props.Title,
			"index":      props.Index,
			"sheetType":  props.SheetType,
			"rowCount":   props.GridProperties.RowCount,
			"columnCount": props.GridProperties.ColumnCount,
		}
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId": spreadsheet.SpreadsheetId,
		"title":         spreadsheet.Properties.Title,
		"locale":        spreadsheet.Properties.Locale,
		"timeZone":      spreadsheet.Properties.TimeZone,
		"sheets":        sheetsInfo,
	})
}

// Helper functions

func formatAsCSV(values [][]interface{}) string {
	var lines []string
	for _, row := range values {
		var cells []string
		for _, cell := range row {
			str := fmt.Sprintf("%v", cell)
			// Escape quotes and wrap in quotes if contains comma
			if strings.Contains(str, ",") || strings.Contains(str, "\"") || strings.Contains(str, "\n") {
				str = "\"" + strings.ReplaceAll(str, "\"", "\"\"") + "\""
			}
			cells = append(cells, str)
		}
		lines = append(lines, strings.Join(cells, ","))
	}
	return strings.Join(lines, "\n")
}

func formatAsJSON(values [][]interface{}) interface{} {
	if len(values) == 0 {
		return []interface{}{}
	}

	// Use first row as headers
	headers := make([]string, len(values[0]))
	for i, h := range values[0] {
		headers[i] = fmt.Sprintf("%v", h)
	}

	// Convert remaining rows to objects
	result := make([]map[string]interface{}, 0, len(values)-1)
	for _, row := range values[1:] {
		obj := make(map[string]interface{})
		for i, cell := range row {
			if i < len(headers) {
				obj[headers[i]] = cell
			}
		}
		result = append(result, obj)
	}

	return result
}

func formatAsText(values [][]interface{}) string {
	var lines []string
	for _, row := range values {
		var cells []string
		for _, cell := range row {
			cells = append(cells, fmt.Sprintf("%v", cell))
		}
		lines = append(lines, strings.Join(cells, "\t"))
	}
	return strings.Join(lines, "\n")
}
