// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/slides/v1"
)

// SlidesService provides Google Slides operations
type SlidesService struct {
	auth AuthProvider
}

// NewSlidesService creates a new SlidesService instance
func NewSlidesService(auth AuthProvider) *SlidesService {
	return &SlidesService{auth: auth}
}

// getSlidesClient returns an authenticated Slides client
func (s *SlidesService) getSlidesClient(ctx context.Context) (*slides.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return slides.NewService(ctx, opt)
}

// getDriveClient returns an authenticated Drive client
func (s *SlidesService) getDriveClient(ctx context.Context) (*drive.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return drive.NewService(ctx, opt)
}

// GetText retrieves text content from a presentation
func (s *SlidesService) GetText(ctx context.Context, presentationID string) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	presentation, err := client.Presentations.Get(presentationID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	var slideTexts []map[string]interface{}
	for i, slide := range presentation.Slides {
		slideText := map[string]interface{}{
			"slideNumber": i + 1,
			"slideId":     slide.ObjectId,
			"text":        extractSlideText(slide),
		}
		slideTexts = append(slideTexts, slideText)
	}

	return JSONResponse(map[string]interface{}{
		"presentationId": presentation.PresentationId,
		"title":          presentation.Title,
		"slides":         slideTexts,
	})
}

// Find searches for presentations
func (s *SlidesService) Find(ctx context.Context, query string, pageToken *string, pageSize *int) ToolResponse {
	driveClient, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	driveQuery := fmt.Sprintf("name contains '%s' and mimeType = 'application/vnd.google-apps.presentation' and trashed = false",
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
		"presentations": files,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// GetMetadata gets presentation metadata
func (s *SlidesService) GetMetadata(ctx context.Context, presentationID string) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	presentation, err := client.Presentations.Get(presentationID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	var pageWidth, pageHeight float64
	if presentation.PageSize != nil {
		if presentation.PageSize.Width != nil {
			pageWidth = presentation.PageSize.Width.Magnitude
		}
		if presentation.PageSize.Height != nil {
			pageHeight = presentation.PageSize.Height.Magnitude
		}
	}

	return JSONResponse(map[string]interface{}{
		"presentationId": presentation.PresentationId,
		"title":          presentation.Title,
		"locale":         presentation.Locale,
		"slideCount":     len(presentation.Slides),
		"pageSize": map[string]interface{}{
			"width":  pageWidth,
			"height": pageHeight,
		},
	})
}

// extractSlideText extracts all text from a slide
func extractSlideText(slide *slides.Page) string {
	var texts []string

	for _, element := range slide.PageElements {
		if element.Shape != nil && element.Shape.Text != nil {
			for _, textElement := range element.Shape.Text.TextElements {
				if textElement.TextRun != nil {
					texts = append(texts, textElement.TextRun.Content)
				}
			}
		}
		if element.Table != nil {
			for _, row := range element.Table.TableRows {
				for _, cell := range row.TableCells {
					if cell.Text != nil {
						for _, textElement := range cell.Text.TextElements {
							if textElement.TextRun != nil {
								texts = append(texts, textElement.TextRun.Content)
							}
						}
					}
				}
			}
		}
	}

	return strings.Join(texts, "")
}
