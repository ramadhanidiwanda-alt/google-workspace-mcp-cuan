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

// UploadFile uploads a file to Drive
func (s *DriveService) UploadFile(ctx context.Context, localPath string, fileName *string, folderID *string, mimeType *string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Open local file
	f, err := os.Open(localPath)
	if err != nil {
		return ErrorResponse(err)
	}
	defer f.Close()

	// Determine file name
	name := filepath.Base(localPath)
	if fileName != nil && *fileName != "" {
		name = *fileName
	}

	file := &drive.File{
		Name: name,
	}

	if folderID != nil && *folderID != "" {
		file.Parents = []string{*folderID}
	}

	if mimeType != nil && *mimeType != "" {
		file.MimeType = *mimeType
	}

	result, err := client.Files.Create(file).
		Media(f).
		Fields("id, name, mimeType, size, webViewLink").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"id":          result.Id,
		"name":        result.Name,
		"mimeType":    result.MimeType,
		"size":        result.Size,
		"webViewLink": result.WebViewLink,
	})
}

// CopyFile copies a file
func (s *DriveService) CopyFile(ctx context.Context, fileID string, newName *string, folderID *string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	copyFile := &drive.File{}
	if newName != nil && *newName != "" {
		copyFile.Name = *newName
	}
	if folderID != nil && *folderID != "" {
		copyFile.Parents = []string{*folderID}
	}

	result, err := client.Files.Copy(fileID, copyFile).
		Fields("id, name, mimeType, webViewLink").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"id":          result.Id,
		"name":        result.Name,
		"mimeType":    result.MimeType,
		"webViewLink": result.WebViewLink,
	})
}

// DeleteFile moves a file to trash or permanently deletes it
func (s *DriveService) DeleteFile(ctx context.Context, fileID string, permanent bool) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	if permanent {
		err = client.Files.Delete(fileID).Do()
	} else {
		_, err = client.Files.Update(fileID, &drive.File{Trashed: true}).Do()
	}

	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":    "success",
		"fileId":    fileID,
		"permanent": permanent,
	})
}

// GetFileInfo gets detailed file information
func (s *DriveService) GetFileInfo(ctx context.Context, fileID string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	file, err := client.Files.Get(fileID).
		Fields("id, name, mimeType, size, createdTime, modifiedTime, owners, parents, webViewLink, permissions").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	owners := make([]map[string]string, len(file.Owners))
	for i, o := range file.Owners {
		owners[i] = map[string]string{
			"displayName":  o.DisplayName,
			"emailAddress": o.EmailAddress,
		}
	}

	permissions := make([]map[string]interface{}, len(file.Permissions))
	for i, p := range file.Permissions {
		permissions[i] = map[string]interface{}{
			"id":           p.Id,
			"type":         p.Type,
			"role":         p.Role,
			"emailAddress": p.EmailAddress,
			"displayName":  p.DisplayName,
		}
	}

	return JSONResponse(map[string]interface{}{
		"id":           file.Id,
		"name":         file.Name,
		"mimeType":     file.MimeType,
		"size":         file.Size,
		"createdTime":  file.CreatedTime,
		"modifiedTime": file.ModifiedTime,
		"owners":       owners,
		"parents":      file.Parents,
		"webViewLink":  file.WebViewLink,
		"permissions":  permissions,
	})
}

// ShareFile shares a file with a user or makes it public
func (s *DriveService) ShareFile(ctx context.Context, fileID string, email *string, role string, shareType string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	permission := &drive.Permission{
		Role: role, // "reader", "writer", "commenter"
		Type: shareType, // "user", "group", "domain", "anyone"
	}

	if email != nil && *email != "" {
		permission.EmailAddress = *email
	}

	result, err := client.Permissions.Create(fileID, permission).
		Fields("id, type, role, emailAddress").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"permissionId": result.Id,
		"type":         result.Type,
		"role":         result.Role,
		"emailAddress": result.EmailAddress,
	})
}

// RemoveShare removes a sharing permission
func (s *DriveService) RemoveShare(ctx context.Context, fileID, permissionID string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	err = client.Permissions.Delete(fileID, permissionID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":       "success",
		"fileId":       fileID,
		"permissionId": permissionID,
	})
}

// ListTrash lists files in trash
func (s *DriveService) ListTrash(ctx context.Context, pageToken *string, pageSize *int) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.Files.List().
		Q("trashed = true").
		Fields("nextPageToken, files(id, name, mimeType, trashedTime)")

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
			"id":          f.Id,
			"name":        f.Name,
			"mimeType":    f.MimeType,
			"trashedTime": f.TrashedTime,
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

// RestoreFile restores a file from trash
func (s *DriveService) RestoreFile(ctx context.Context, fileID string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	file, err := client.Files.Update(fileID, &drive.File{Trashed: false}).
		Fields("id, name").
		Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status": "success",
		"id":     file.Id,
		"name":   file.Name,
	})
}

// EmptyTrash permanently deletes all files in trash
func (s *DriveService) EmptyTrash(ctx context.Context) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	err = client.Files.EmptyTrash().Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":  "success",
		"message": "Trash emptied",
	})
}

// ExportFile exports a Google Workspace file (Docs, Sheets, Slides) to a different format
// Supported MIME types:
// - application/pdf (all)
// - application/vnd.openxmlformats-officedocument.wordprocessingml.document (Docs -> DOCX)
// - application/vnd.openxmlformats-officedocument.spreadsheetml.sheet (Sheets -> XLSX)
// - application/vnd.openxmlformats-officedocument.presentationml.presentation (Slides -> PPTX)
// - text/plain (Docs)
// - text/csv (Sheets)
func (s *DriveService) ExportFile(ctx context.Context, fileID, mimeType, localPath string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Get file metadata first
	file, err := client.Files.Get(fileID).Fields("name, mimeType").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Export the file
	resp, err := client.Files.Export(fileID, mimeType).Download()
	if err != nil {
		return ErrorResponse(err)
	}
	defer resp.Body.Close()

	// Create directory if it doesn't exist
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ErrorResponse(fmt.Errorf("failed to create directory: %w", err))
	}

	// Create output file
	out, err := os.Create(localPath)
	if err != nil {
		return ErrorResponse(fmt.Errorf("failed to create file: %w", err))
	}
	defer out.Close()

	// Copy content
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return ErrorResponse(fmt.Errorf("failed to write file: %w", err))
	}

	return JSONResponse(map[string]interface{}{
		"status":       "success",
		"fileName":     file.Name,
		"exportedTo":   localPath,
		"exportFormat": mimeType,
		"bytesWritten": written,
	})
}

// ListComments lists comments on a file
func (s *DriveService) ListComments(ctx context.Context, fileID string, pageToken *string, pageSize *int) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.Comments.List(fileID).Fields("comments(id,content,author,createdTime,modifiedTime,resolved,replies)")

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

	comments := make([]map[string]interface{}, len(result.Comments))
	for i, c := range result.Comments {
		comment := map[string]interface{}{
			"id":           c.Id,
			"content":      c.Content,
			"createdTime":  c.CreatedTime,
			"modifiedTime": c.ModifiedTime,
			"resolved":     c.Resolved,
		}
		if c.Author != nil {
			comment["author"] = map[string]string{
				"displayName": c.Author.DisplayName,
				"emailAddress": c.Author.EmailAddress,
			}
		}
		if len(c.Replies) > 0 {
			replies := make([]map[string]interface{}, len(c.Replies))
			for j, r := range c.Replies {
				reply := map[string]interface{}{
					"id":          r.Id,
					"content":     r.Content,
					"createdTime": r.CreatedTime,
				}
				if r.Author != nil {
					reply["author"] = map[string]string{
						"displayName":  r.Author.DisplayName,
						"emailAddress": r.Author.EmailAddress,
					}
				}
				replies[j] = reply
			}
			comment["replies"] = replies
		}
		comments[i] = comment
	}

	response := map[string]interface{}{
		"comments": comments,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// CreateComment creates a comment on a file
func (s *DriveService) CreateComment(ctx context.Context, fileID, content string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	comment := &drive.Comment{
		Content: content,
	}

	result, err := client.Comments.Create(fileID, comment).Fields("id,content,author,createdTime").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	response := map[string]interface{}{
		"id":          result.Id,
		"content":     result.Content,
		"createdTime": result.CreatedTime,
	}
	if result.Author != nil {
		response["author"] = map[string]string{
			"displayName":  result.Author.DisplayName,
			"emailAddress": result.Author.EmailAddress,
		}
	}

	return JSONResponse(response)
}

// ReplyToComment replies to a comment on a file
func (s *DriveService) ReplyToComment(ctx context.Context, fileID, commentID, content string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	reply := &drive.Reply{
		Content: content,
	}

	result, err := client.Replies.Create(fileID, commentID, reply).Fields("id,content,author,createdTime").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	response := map[string]interface{}{
		"id":          result.Id,
		"content":     result.Content,
		"createdTime": result.CreatedTime,
	}
	if result.Author != nil {
		response["author"] = map[string]string{
			"displayName":  result.Author.DisplayName,
			"emailAddress": result.Author.EmailAddress,
		}
	}

	return JSONResponse(response)
}

// ResolveComment marks a comment as resolved
func (s *DriveService) ResolveComment(ctx context.Context, fileID, commentID string, resolved bool) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// To resolve a comment, we need to create a reply that resolves it
	reply := &drive.Reply{
		Content: "",
		Action:  "resolve",
	}
	if !resolved {
		reply.Action = "reopen"
	}

	_, err = client.Replies.Create(fileID, commentID, reply).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":    "success",
		"commentId": commentID,
		"resolved":  resolved,
	})
}

// ListRevisions lists file revisions (version history)
func (s *DriveService) ListRevisions(ctx context.Context, fileID string, pageToken *string, pageSize *int) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.Revisions.List(fileID).Fields("revisions(id,modifiedTime,lastModifyingUser,size,keepForever)")

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

	revisions := make([]map[string]interface{}, len(result.Revisions))
	for i, r := range result.Revisions {
		revision := map[string]interface{}{
			"id":           r.Id,
			"modifiedTime": r.ModifiedTime,
			"size":         r.Size,
			"keepForever":  r.KeepForever,
		}
		if r.LastModifyingUser != nil {
			revision["lastModifyingUser"] = map[string]string{
				"displayName":  r.LastModifyingUser.DisplayName,
				"emailAddress": r.LastModifyingUser.EmailAddress,
			}
		}
		revisions[i] = revision
	}

	response := map[string]interface{}{
		"revisions": revisions,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// GetRevision gets a specific revision
func (s *DriveService) GetRevision(ctx context.Context, fileID, revisionID string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	result, err := client.Revisions.Get(fileID, revisionID).Fields("*").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	response := map[string]interface{}{
		"id":           result.Id,
		"modifiedTime": result.ModifiedTime,
		"size":         result.Size,
		"keepForever":  result.KeepForever,
		"mimeType":     result.MimeType,
	}
	if result.LastModifyingUser != nil {
		response["lastModifyingUser"] = map[string]string{
			"displayName":  result.LastModifyingUser.DisplayName,
			"emailAddress": result.LastModifyingUser.EmailAddress,
		}
	}

	return JSONResponse(response)
}

// GetExportFormats returns available export formats for a Google Workspace file
func (s *DriveService) GetExportFormats(ctx context.Context, fileID string) ToolResponse {
	client, err := s.getDriveClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	file, err := client.Files.Get(fileID).Fields("name, mimeType, exportLinks").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	formats := make(map[string]string)

	// Define common export formats based on source type
	switch file.MimeType {
	case "application/vnd.google-apps.document":
		formats["pdf"] = "application/pdf"
		formats["docx"] = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		formats["txt"] = "text/plain"
		formats["html"] = "text/html"
		formats["rtf"] = "application/rtf"
		formats["odt"] = "application/vnd.oasis.opendocument.text"
	case "application/vnd.google-apps.spreadsheet":
		formats["pdf"] = "application/pdf"
		formats["xlsx"] = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		formats["csv"] = "text/csv"
		formats["ods"] = "application/vnd.oasis.opendocument.spreadsheet"
	case "application/vnd.google-apps.presentation":
		formats["pdf"] = "application/pdf"
		formats["pptx"] = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		formats["odp"] = "application/vnd.oasis.opendocument.presentation"
	case "application/vnd.google-apps.drawing":
		formats["pdf"] = "application/pdf"
		formats["png"] = "image/png"
		formats["jpg"] = "image/jpeg"
		formats["svg"] = "image/svg+xml"
	}

	return JSONResponse(map[string]interface{}{
		"fileId":   fileID,
		"fileName": file.Name,
		"mimeType": file.MimeType,
		"formats":  formats,
	})
}
