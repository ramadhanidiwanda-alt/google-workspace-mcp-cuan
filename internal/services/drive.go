// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oowada/google-workspace-mcp/internal/util"
	"google.golang.org/api/drive/v3"
)

// DriveService provides Google Drive operations
type DriveService struct {
	auth AuthProvider
}

// NewDriveService creates a new DriveService instance
func NewDriveService(auth AuthProvider) *DriveService {
	return &DriveService{auth: auth}
}

// getDriveClient returns an authenticated Drive client
func (s *DriveService) getDriveClient(ctx context.Context) (*drive.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return drive.NewService(ctx, opt)
}

// FindFolderInput contains input for FindFolder
type FindFolderInput struct {
	FolderName string `json:"folderName"`
}

// FindFolderResult contains the result of FindFolder
type FindFolderResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FindFolder finds a folder by name
func (s *DriveService) FindFolder(ctx context.Context, folderName string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	query := fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.folder' and trashed = false",
		escapeQuery(folderName))

	result, err := client.Files.List().
		Q(query).
		Fields("files(id, name)").
		PageSize(1).
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	if len(result.Files) == 0 {
		return JSONResponse(map[string]interface{}{
			"found": false,
		})
	}

	return JSONResponse(map[string]interface{}{
		"found": true,
		"id":    result.Files[0].Id,
		"name":  result.Files[0].Name,
	})
}

// CreateFolderInput contains input for CreateFolder
type CreateFolderInput struct {
	FolderName string  `json:"folderName"`
	ParentID   *string `json:"parentId,omitempty"`
}

// CreateFolder creates a new folder
func (s *DriveService) CreateFolder(ctx context.Context, folderName string, parentID *string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	folder := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}

	if parentID != nil && *parentID != "" {
		folder.Parents = []string{*parentID}
	}

	result, err := client.Files.Create(folder).
		Fields("id, name").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"id":   result.Id,
		"name": result.Name,
	})
}

// SearchInput contains input for Search
type SearchInput struct {
	Query     string  `json:"query"`
	PageToken *string `json:"pageToken,omitempty"`
	PageSize  *int    `json:"pageSize,omitempty"`
}

// Search searches for files in Drive
func (s *DriveService) Search(ctx context.Context, query string, pageToken *string, pageSize *int) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Build the query
	driveQuery := buildDriveQuery(query)

	req := client.Files.List().
		Q(driveQuery).
		Fields("nextPageToken, files(id, name, mimeType, modifiedTime, size, webViewLink)")

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
			"mimeType":     f.MimeType,
			"modifiedTime": f.ModifiedTime,
			"size":         f.Size,
			"webViewLink":  f.WebViewLink,
		}
	}

	response := map[string]interface{}{
		"files": files,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// DownloadFileInput contains input for DownloadFile
type DownloadFileInput struct {
	FileID    string `json:"fileId"`
	LocalPath string `json:"localPath"`
}

// DownloadFile downloads a file from Drive
func (s *DriveService) DownloadFile(ctx context.Context, fileID string, localPath string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Get file metadata first
	file, err := client.Files.Get(fileID).Fields("name, mimeType, size").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Handle Google Docs export
	var resp *drive.Service
	var reader io.ReadCloser

	if isGoogleDoc(file.MimeType) {
		exportMimeType := getExportMimeType(file.MimeType)
		httpResp, err := client.Files.Export(fileID, exportMimeType).Download()
		if err != nil {
			return ErrorResponse(err)
		}
		reader = httpResp.Body
	} else {
		httpResp, err := client.Files.Get(fileID).Download()
		if err != nil {
			return ErrorResponse(err)
		}
		reader = httpResp.Body
	}
	defer reader.Close()
	_ = resp // unused

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return ErrorResponse(err)
	}

	// Write to file
	out, err := os.Create(localPath)
	if err != nil {
		return ErrorResponse(err)
	}
	defer out.Close()

	written, err := io.Copy(out, reader)
	if err != nil {
		return ErrorResponse(err)
	}

	util.LogDebug("Downloaded %d bytes to %s", written, localPath)

	return JSONResponse(map[string]interface{}{
		"fileName":  file.Name,
		"localPath": localPath,
		"size":      written,
	})
}

// MoveFile moves a file to a folder
func (s *DriveService) MoveFile(ctx context.Context, fileID string, folderID string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Get current parents
	file, err := client.Files.Get(fileID).Fields("parents").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	previousParents := strings.Join(file.Parents, ",")

	// Move file
	_, err = client.Files.Update(fileID, nil).
		AddParents(folderID).
		RemoveParents(previousParents).
		Fields("id, parents").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"status": "success",
		"fileId": fileID,
	})
}

// Helper functions

func escapeQuery(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}

func buildDriveQuery(query string) string {
	// If query looks like a Drive URL, extract file ID
	if strings.Contains(query, "drive.google.com") || strings.Contains(query, "docs.google.com") {
		// Let the caller handle URL parsing
		return fmt.Sprintf("name contains '%s' and trashed = false", escapeQuery(query))
	}

	// If query is already a Drive query, use it as-is
	if strings.Contains(query, "=") || strings.Contains(query, "contains") {
		return query
	}

	// Otherwise, search by name
	return fmt.Sprintf("name contains '%s' and trashed = false", escapeQuery(query))
}

func isGoogleDoc(mimeType string) bool {
	return strings.HasPrefix(mimeType, "application/vnd.google-apps.")
}

func getExportMimeType(googleMimeType string) string {
	switch googleMimeType {
	case "application/vnd.google-apps.document":
		return "application/pdf"
	case "application/vnd.google-apps.spreadsheet":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "application/vnd.google-apps.presentation":
		return "application/pdf"
	default:
		return "application/pdf"
	}
}
