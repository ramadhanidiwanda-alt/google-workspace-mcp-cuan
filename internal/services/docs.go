// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/oowada/google-workspace-mcp/internal/util"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
)

// DocsService provides Google Docs operations
type DocsService struct {
	auth  AuthProvider
	drive *DriveService
}

// NewDocsService creates a new DocsService instance
func NewDocsService(auth AuthProvider, drive *DriveService) *DocsService {
	return &DocsService{auth: auth, drive: drive}
}

// getDocsClient returns an authenticated Docs client
func (s *DocsService) getDocsClient(ctx context.Context) (*docs.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return docs.NewService(ctx, opt)
}

// getDriveClient returns an authenticated Drive client
func (s *DocsService) getDriveClient(ctx context.Context) (*drive.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return drive.NewService(ctx, opt)
}

// Create creates a new Google Doc
func (s *DocsService) Create(ctx context.Context, title string, folderName *string, markdown *string) ToolResponse {
	client, err := s.getDocsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Create document
	doc := &docs.Document{Title: title}
	result, err := client.Documents.Create(doc).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	util.LogDebug("Created document: %s", result.DocumentId)

	// Insert markdown content if provided
	if markdown != nil && *markdown != "" {
		requests := []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: 1},
					Text:     *markdown,
				},
			},
		}

		_, err = client.Documents.BatchUpdate(result.DocumentId, &docs.BatchUpdateDocumentRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			util.LogWarn("Failed to insert content: %v", err)
		}
	}

	// Move to folder if specified
	if folderName != nil && *folderName != "" {
		folderResp := s.drive.FindFolder(ctx, *folderName)
		// Parse folder response to get ID
		// For now, just log if folder not found
		util.LogDebug("Folder lookup response: %s", folderResp.Content[0].Text)
	}

	return JSONResponse(map[string]string{
		"documentId": result.DocumentId,
		"title":      result.Title,
	})
}

// GetText retrieves the text content of a document
func (s *DocsService) GetText(ctx context.Context, documentID string, tabID *string) ToolResponse {
	client, err := s.getDocsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	doc, err := client.Documents.Get(documentID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Extract text from document
	text := extractDocumentText(doc)

	return JSONResponse(map[string]string{
		"documentId": doc.DocumentId,
		"title":      doc.Title,
		"text":       text,
	})
}

// InsertText inserts text at the beginning of a document
func (s *DocsService) InsertText(ctx context.Context, documentID string, text string, tabID *string) ToolResponse {
	client, err := s.getDocsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: 1},
				Text:     text,
			},
		},
	}

	_, err = client.Documents.BatchUpdate(documentID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"status":     "success",
		"documentId": documentID,
	})
}

// AppendText appends text to the end of a document
func (s *DocsService) AppendText(ctx context.Context, documentID string, text string, tabID *string) ToolResponse {
	client, err := s.getDocsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Get document to find end index
	doc, err := client.Documents.Get(documentID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	endIndex := int64(1)
	if doc.Body != nil && doc.Body.Content != nil && len(doc.Body.Content) > 0 {
		lastElement := doc.Body.Content[len(doc.Body.Content)-1]
		if lastElement.EndIndex > 1 {
			endIndex = lastElement.EndIndex - 1
		}
	}

	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: endIndex},
				Text:     text,
			},
		},
	}

	_, err = client.Documents.BatchUpdate(documentID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"status":     "success",
		"documentId": documentID,
	})
}

// ReplaceText replaces text in a document
func (s *DocsService) ReplaceText(ctx context.Context, documentID string, findText string, replaceText string, tabID *string) ToolResponse {
	client, err := s.getDocsClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	requests := []*docs.Request{
		{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{
					Text:            findText,
					MatchCase:       true,
				},
				ReplaceText: replaceText,
			},
		},
	}

	result, err := client.Documents.BatchUpdate(documentID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	replacements := int64(0)
	if len(result.Replies) > 0 && result.Replies[0].ReplaceAllText != nil {
		replacements = result.Replies[0].ReplaceAllText.OccurrencesChanged
	}

	return JSONResponse(map[string]interface{}{
		"status":       "success",
		"documentId":   documentID,
		"replacements": replacements,
	})
}

// Move moves a document to a folder
func (s *DocsService) Move(ctx context.Context, documentID string, folderName string) ToolResponse {
	// Find folder
	folderResp := s.drive.FindFolder(ctx, folderName)
	if strings.Contains(folderResp.Content[0].Text, `"found":false`) {
		return ErrorResponse(fmt.Errorf("folder not found: %s", folderName))
	}

	// Extract folder ID from response (simplified)
	// In real implementation, parse JSON properly
	return s.drive.MoveFile(ctx, documentID, folderName)
}

// Find searches for documents
func (s *DocsService) Find(ctx context.Context, query string, pageToken *string, pageSize *int) ToolResponse {
	driveClient, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	driveQuery := fmt.Sprintf("name contains '%s' and mimeType = 'application/vnd.google-apps.document' and trashed = false",
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
		"documents": files,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// ExtractIDFromURL extracts a document ID from a Google Workspace URL
func (s *DocsService) ExtractIDFromURL(url string) ToolResponse {
	id := extractDocID(url)
	if id == "" {
		return ErrorResponse(fmt.Errorf("could not extract document ID from URL"))
	}
	return JSONResponse(map[string]string{
		"documentId": id,
	})
}

// Helper functions

func extractDocumentText(doc *docs.Document) string {
	var sb strings.Builder

	if doc.Body == nil || doc.Body.Content == nil {
		return ""
	}

	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			for _, elem := range element.Paragraph.Elements {
				if elem.TextRun != nil {
					sb.WriteString(elem.TextRun.Content)
				}
			}
		}
	}

	return sb.String()
}

var docIDPattern = regexp.MustCompile(`/d/([a-zA-Z0-9_-]+)`)

func extractDocID(url string) string {
	matches := docIDPattern.FindStringSubmatch(url)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
