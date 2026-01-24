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

// Create creates a new presentation
func (s *SlidesService) Create(ctx context.Context, title string) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	presentation := &slides.Presentation{
		Title: title,
	}

	result, err := client.Presentations.Create(presentation).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"presentationId": result.PresentationId,
		"title":          result.Title,
		"slideCount":     len(result.Slides),
	})
}

// AddSlide adds a new slide to a presentation
func (s *SlidesService) AddSlide(ctx context.Context, presentationID string, layout *string, insertionIndex *int) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Use predefined layout
	layoutName := "BLANK"
	if layout != nil && *layout != "" {
		layoutName = strings.ToUpper(*layout)
	}

	req := &slides.Request{
		CreateSlide: &slides.CreateSlideRequest{
			SlideLayoutReference: &slides.LayoutReference{
				PredefinedLayout: layoutName,
			},
		},
	}

	if insertionIndex != nil {
		req.CreateSlide.InsertionIndex = int64(*insertionIndex)
	}

	result, err := client.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{req},
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	var slideID string
	if len(result.Replies) > 0 && result.Replies[0].CreateSlide != nil {
		slideID = result.Replies[0].CreateSlide.ObjectId
	}

	return JSONResponse(map[string]interface{}{
		"presentationId": presentationID,
		"slideId":        slideID,
	})
}

// DeleteSlide deletes a slide from a presentation
func (s *SlidesService) DeleteSlide(ctx context.Context, presentationID, slideID string) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	_, err = client.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteObject: &slides.DeleteObjectRequest{
					ObjectId: slideID,
				},
			},
		},
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":         "success",
		"presentationId": presentationID,
		"deletedSlide":   slideID,
	})
}

// AddTextBox adds a text box to a slide
func (s *SlidesService) AddTextBox(ctx context.Context, presentationID, slideID, text string, x, y, width, height float64) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	elementID := fmt.Sprintf("textbox_%d", generateID())

	requests := []*slides.Request{
		{
			CreateShape: &slides.CreateShapeRequest{
				ObjectId:  elementID,
				ShapeType: "TEXT_BOX",
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: slideID,
					Size: &slides.Size{
						Width:  &slides.Dimension{Magnitude: width, Unit: "PT"},
						Height: &slides.Dimension{Magnitude: height, Unit: "PT"},
					},
					Transform: &slides.AffineTransform{
						ScaleX:     1,
						ScaleY:     1,
						TranslateX: x,
						TranslateY: y,
						Unit:       "PT",
					},
				},
			},
		},
		{
			InsertText: &slides.InsertTextRequest{
				ObjectId: elementID,
				Text:     text,
			},
		},
	}

	_, err = client.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"presentationId": presentationID,
		"slideId":        slideID,
		"elementId":      elementID,
	})
}

// AddImage adds an image to a slide
func (s *SlidesService) AddImage(ctx context.Context, presentationID, slideID, imageURL string, x, y, width, height float64) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	elementID := fmt.Sprintf("image_%d", generateID())

	_, err = client.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateImage: &slides.CreateImageRequest{
					ObjectId: elementID,
					Url:      imageURL,
					ElementProperties: &slides.PageElementProperties{
						PageObjectId: slideID,
						Size: &slides.Size{
							Width:  &slides.Dimension{Magnitude: width, Unit: "PT"},
							Height: &slides.Dimension{Magnitude: height, Unit: "PT"},
						},
						Transform: &slides.AffineTransform{
							ScaleX:     1,
							ScaleY:     1,
							TranslateX: x,
							TranslateY: y,
							Unit:       "PT",
						},
					},
				},
			},
		},
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"presentationId": presentationID,
		"slideId":        slideID,
		"elementId":      elementID,
	})
}

// UpdateText updates text in a shape
func (s *SlidesService) UpdateText(ctx context.Context, presentationID, shapeID, text string) ToolResponse {
	client, err := s.getSlidesClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	_, err = client.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteText: &slides.DeleteTextRequest{
					ObjectId: shapeID,
					TextRange: &slides.Range{
						Type: "ALL",
					},
				},
			},
			{
				InsertText: &slides.InsertTextRequest{
					ObjectId: shapeID,
					Text:     text,
				},
			},
		},
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":         "success",
		"presentationId": presentationID,
		"shapeId":        shapeID,
	})
}

var idCounter int64

func generateID() int64 {
	idCounter++
	return idCounter
}
