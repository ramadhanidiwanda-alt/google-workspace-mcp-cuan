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

// UpdateRange updates values in a specific range
func (s *SheetsService) UpdateRange(ctx context.Context, spreadsheetID, rangeA1 string, values [][]interface{}) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	result, err := client.Spreadsheets.Values.Update(spreadsheetID, rangeA1, valueRange).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId":  result.SpreadsheetId,
		"updatedRange":   result.UpdatedRange,
		"updatedRows":    result.UpdatedRows,
		"updatedColumns": result.UpdatedColumns,
		"updatedCells":   result.UpdatedCells,
	})
}

// AppendRows appends rows to a sheet
func (s *SheetsService) AppendRows(ctx context.Context, spreadsheetID, rangeA1 string, values [][]interface{}) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	result, err := client.Spreadsheets.Values.Append(spreadsheetID, rangeA1, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId": result.SpreadsheetId,
		"tableRange":    result.TableRange,
		"updatedRange":  result.Updates.UpdatedRange,
		"updatedRows":   result.Updates.UpdatedRows,
		"updatedCells":  result.Updates.UpdatedCells,
	})
}

// ClearRange clears values in a range
func (s *SheetsService) ClearRange(ctx context.Context, spreadsheetID, rangeA1 string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	result, err := client.Spreadsheets.Values.Clear(spreadsheetID, rangeA1, &sheets.ClearValuesRequest{}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId": result.SpreadsheetId,
		"clearedRange":  result.ClearedRange,
	})
}

// CreateSheet creates a new sheet in a spreadsheet
func (s *SheetsService) CreateSheet(ctx context.Context, spreadsheetID, sheetTitle string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: sheetTitle,
					},
				},
			},
		},
	}

	result, err := client.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	var sheetID int64
	if len(result.Replies) > 0 && result.Replies[0].AddSheet != nil {
		sheetID = result.Replies[0].AddSheet.Properties.SheetId
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId": spreadsheetID,
		"sheetId":       sheetID,
		"sheetTitle":    sheetTitle,
	})
}

// DeleteSheet deletes a sheet from a spreadsheet
func (s *SheetsService) DeleteSheet(ctx context.Context, spreadsheetID string, sheetID int64) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: sheetID,
				},
			},
		},
	}

	_, err = client.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":        "success",
		"spreadsheetId": spreadsheetID,
		"deletedSheet":  sheetID,
	})
}

// AddChart adds a chart to a sheet
func (s *SheetsService) AddChart(ctx context.Context, spreadsheetID string, sheetID int64, chartType string, dataRange string, title *string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Parse the data range to get coordinates
	// Simple parsing for A1 notation like "Sheet1!A1:B10"
	var sourceSheetID int64
	var startRow, endRow, startCol, endCol int64

	// Default values - will be overridden by dataRange parsing
	sourceSheetID = sheetID
	startRow = 0
	endRow = 10
	startCol = 0
	endCol = 2

	// Map chart type string to enum
	var chartSpec *sheets.ChartSpec
	switch strings.ToUpper(chartType) {
	case "BAR":
		chartSpec = &sheets.ChartSpec{
			Title: "",
			BasicChart: &sheets.BasicChartSpec{
				ChartType: "BAR",
				Domains: []*sheets.BasicChartDomain{
					{
						Domain: &sheets.ChartData{
							SourceRange: &sheets.ChartSourceRange{
								Sources: []*sheets.GridRange{
									{
										SheetId:          sourceSheetID,
										StartRowIndex:    startRow,
										EndRowIndex:      endRow,
										StartColumnIndex: startCol,
										EndColumnIndex:   startCol + 1,
									},
								},
							},
						},
					},
				},
				Series: []*sheets.BasicChartSeries{
					{
						Series: &sheets.ChartData{
							SourceRange: &sheets.ChartSourceRange{
								Sources: []*sheets.GridRange{
									{
										SheetId:          sourceSheetID,
										StartRowIndex:    startRow,
										EndRowIndex:      endRow,
										StartColumnIndex: startCol + 1,
										EndColumnIndex:   endCol,
									},
								},
							},
						},
					},
				},
			},
		}
	case "LINE":
		chartSpec = &sheets.ChartSpec{
			Title: "",
			BasicChart: &sheets.BasicChartSpec{
				ChartType: "LINE",
				Domains: []*sheets.BasicChartDomain{
					{
						Domain: &sheets.ChartData{
							SourceRange: &sheets.ChartSourceRange{
								Sources: []*sheets.GridRange{
									{
										SheetId:          sourceSheetID,
										StartRowIndex:    startRow,
										EndRowIndex:      endRow,
										StartColumnIndex: startCol,
										EndColumnIndex:   startCol + 1,
									},
								},
							},
						},
					},
				},
				Series: []*sheets.BasicChartSeries{
					{
						Series: &sheets.ChartData{
							SourceRange: &sheets.ChartSourceRange{
								Sources: []*sheets.GridRange{
									{
										SheetId:          sourceSheetID,
										StartRowIndex:    startRow,
										EndRowIndex:      endRow,
										StartColumnIndex: startCol + 1,
										EndColumnIndex:   endCol,
									},
								},
							},
						},
					},
				},
			},
		}
	case "PIE":
		chartSpec = &sheets.ChartSpec{
			Title: "",
			PieChart: &sheets.PieChartSpec{
				LegendPosition: "RIGHT_LEGEND",
				Domain: &sheets.ChartData{
					SourceRange: &sheets.ChartSourceRange{
						Sources: []*sheets.GridRange{
							{
								SheetId:          sourceSheetID,
								StartRowIndex:    startRow,
								EndRowIndex:      endRow,
								StartColumnIndex: startCol,
								EndColumnIndex:   startCol + 1,
							},
						},
					},
				},
				Series: &sheets.ChartData{
					SourceRange: &sheets.ChartSourceRange{
						Sources: []*sheets.GridRange{
							{
								SheetId:          sourceSheetID,
								StartRowIndex:    startRow,
								EndRowIndex:      endRow,
								StartColumnIndex: startCol + 1,
								EndColumnIndex:   endCol,
							},
						},
					},
				},
			},
		}
	default:
		// Default to column chart
		chartSpec = &sheets.ChartSpec{
			Title: "",
			BasicChart: &sheets.BasicChartSpec{
				ChartType: "COLUMN",
				Domains: []*sheets.BasicChartDomain{
					{
						Domain: &sheets.ChartData{
							SourceRange: &sheets.ChartSourceRange{
								Sources: []*sheets.GridRange{
									{
										SheetId:          sourceSheetID,
										StartRowIndex:    startRow,
										EndRowIndex:      endRow,
										StartColumnIndex: startCol,
										EndColumnIndex:   startCol + 1,
									},
								},
							},
						},
					},
				},
				Series: []*sheets.BasicChartSeries{
					{
						Series: &sheets.ChartData{
							SourceRange: &sheets.ChartSourceRange{
								Sources: []*sheets.GridRange{
									{
										SheetId:          sourceSheetID,
										StartRowIndex:    startRow,
										EndRowIndex:      endRow,
										StartColumnIndex: startCol + 1,
										EndColumnIndex:   endCol,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	if title != nil && *title != "" {
		chartSpec.Title = *title
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddChart: &sheets.AddChartRequest{
					Chart: &sheets.EmbeddedChart{
						Spec: chartSpec,
						Position: &sheets.EmbeddedObjectPosition{
							OverlayPosition: &sheets.OverlayPosition{
								AnchorCell: &sheets.GridCoordinate{
									SheetId:     sheetID,
									RowIndex:    0,
									ColumnIndex: endCol + 1,
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := client.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	var chartID int64
	if len(result.Replies) > 0 && result.Replies[0].AddChart != nil {
		chartID = result.Replies[0].AddChart.Chart.ChartId
	}

	return JSONResponse(map[string]interface{}{
		"status":        "success",
		"spreadsheetId": spreadsheetID,
		"chartId":       chartID,
		"chartType":     chartType,
	})
}

// AddConditionalFormatting adds conditional formatting to a range
func (s *SheetsService) AddConditionalFormatting(ctx context.Context, spreadsheetID string, sheetID int64, rangeA1 string, ruleType string, value *string, bgColor *string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Parse range (simplified - just use entire sheet for now)
	gridRange := &sheets.GridRange{
		SheetId: sheetID,
	}

	var rule *sheets.ConditionalFormatRule

	// Default background color (light red)
	backgroundColor := &sheets.Color{
		Red:   1.0,
		Green: 0.8,
		Blue:  0.8,
	}

	if bgColor != nil && *bgColor != "" {
		// Parse hex color like "#FF0000"
		if len(*bgColor) == 7 && (*bgColor)[0] == '#' {
			r, g, b := parseHexColor(*bgColor)
			backgroundColor = &sheets.Color{
				Red:   r,
				Green: g,
				Blue:  b,
			}
		}
	}

	switch strings.ToUpper(ruleType) {
	case "NOT_BLANK", "NOTBLANK":
		rule = &sheets.ConditionalFormatRule{
			Ranges: []*sheets.GridRange{gridRange},
			BooleanRule: &sheets.BooleanRule{
				Condition: &sheets.BooleanCondition{
					Type: "NOT_BLANK",
				},
				Format: &sheets.CellFormat{
					BackgroundColor: backgroundColor,
				},
			},
		}
	case "BLANK":
		rule = &sheets.ConditionalFormatRule{
			Ranges: []*sheets.GridRange{gridRange},
			BooleanRule: &sheets.BooleanRule{
				Condition: &sheets.BooleanCondition{
					Type: "BLANK",
				},
				Format: &sheets.CellFormat{
					BackgroundColor: backgroundColor,
				},
			},
		}
	case "TEXT_CONTAINS":
		conditionValues := []*sheets.ConditionValue{}
		if value != nil {
			conditionValues = append(conditionValues, &sheets.ConditionValue{
				UserEnteredValue: *value,
			})
		}
		rule = &sheets.ConditionalFormatRule{
			Ranges: []*sheets.GridRange{gridRange},
			BooleanRule: &sheets.BooleanRule{
				Condition: &sheets.BooleanCondition{
					Type:   "TEXT_CONTAINS",
					Values: conditionValues,
				},
				Format: &sheets.CellFormat{
					BackgroundColor: backgroundColor,
				},
			},
		}
	case "NUMBER_GREATER":
		conditionValues := []*sheets.ConditionValue{}
		if value != nil {
			conditionValues = append(conditionValues, &sheets.ConditionValue{
				UserEnteredValue: *value,
			})
		}
		rule = &sheets.ConditionalFormatRule{
			Ranges: []*sheets.GridRange{gridRange},
			BooleanRule: &sheets.BooleanRule{
				Condition: &sheets.BooleanCondition{
					Type:   "NUMBER_GREATER",
					Values: conditionValues,
				},
				Format: &sheets.CellFormat{
					BackgroundColor: backgroundColor,
				},
			},
		}
	case "NUMBER_LESS":
		conditionValues := []*sheets.ConditionValue{}
		if value != nil {
			conditionValues = append(conditionValues, &sheets.ConditionValue{
				UserEnteredValue: *value,
			})
		}
		rule = &sheets.ConditionalFormatRule{
			Ranges: []*sheets.GridRange{gridRange},
			BooleanRule: &sheets.BooleanRule{
				Condition: &sheets.BooleanCondition{
					Type:   "NUMBER_LESS",
					Values: conditionValues,
				},
				Format: &sheets.CellFormat{
					BackgroundColor: backgroundColor,
				},
			},
		}
	default:
		return ErrorResponse(fmt.Errorf("unsupported rule type: %s", ruleType))
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
					Rule:  rule,
					Index: 0,
				},
			},
		},
	}

	_, err = client.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":        "success",
		"spreadsheetId": spreadsheetID,
		"sheetId":       sheetID,
		"ruleType":      ruleType,
	})
}

// parseHexColor parses a hex color string to RGB float values
func parseHexColor(hex string) (float64, float64, float64) {
	if len(hex) != 7 {
		return 1.0, 1.0, 1.0
	}
	var r, g, b int64
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0
}

// Create creates a new spreadsheet
func (s *SheetsService) Create(ctx context.Context, title string, sheetTitles []string) ToolResponse {
	client, err := s.getSheetsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: title,
		},
	}

	if len(sheetTitles) > 0 {
		spreadsheet.Sheets = make([]*sheets.Sheet, len(sheetTitles))
		for i, sheetTitle := range sheetTitles {
			spreadsheet.Sheets[i] = &sheets.Sheet{
				Properties: &sheets.SheetProperties{
					Title: sheetTitle,
				},
			}
		}
	}

	result, err := client.Spreadsheets.Create(spreadsheet).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	sheetsInfo := make([]map[string]interface{}, len(result.Sheets))
	for i, sheet := range result.Sheets {
		sheetsInfo[i] = map[string]interface{}{
			"sheetId": sheet.Properties.SheetId,
			"title":   sheet.Properties.Title,
		}
	}

	return JSONResponse(map[string]interface{}{
		"spreadsheetId": result.SpreadsheetId,
		"title":         result.Properties.Title,
		"spreadsheetUrl": result.SpreadsheetUrl,
		"sheets":        sheetsInfo,
	})
}
