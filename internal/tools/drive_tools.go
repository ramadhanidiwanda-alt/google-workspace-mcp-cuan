// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerDriveTools() {
	// drive.findFolder
	r.server.AddTool(
		mcp.NewTool("drive.findFolder",
			mcp.WithDescription("Finds a folder in Google Drive by name."),
			mcp.WithString("folderName", mcp.Required(), mcp.Description("The name of the folder to find")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			folderName := args["folderName"].(string)
			resp := r.services.Drive.FindFolder(ctx, folderName)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.createFolder
	r.server.AddTool(
		mcp.NewTool("drive.createFolder",
			mcp.WithDescription("Creates a new folder in Google Drive."),
			mcp.WithString("folderName", mcp.Required(), mcp.Description("The name of the folder to create")),
			mcp.WithString("parentId", mcp.Description("The ID of the parent folder")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			folderName := args["folderName"].(string)
			var parentID *string
			if v, ok := args["parentId"].(string); ok && v != "" {
				parentID = &v
			}
			resp := r.services.Drive.CreateFolder(ctx, folderName, parentID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.search
	r.server.AddTool(
		mcp.NewTool("drive.search",
			mcp.WithDescription("Searches for files and folders in Google Drive."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query or Drive URL")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			query := args["query"].(string)
			var pageToken *string
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			var pageSize *int
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Drive.Search(ctx, query, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.downloadFile
	r.server.AddTool(
		mcp.NewTool("drive.downloadFile",
			mcp.WithDescription("Downloads a file from Google Drive to a local path."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file to download")),
			mcp.WithString("localPath", mcp.Required(), mcp.Description("The local path to save the file")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			localPath := args["localPath"].(string)
			resp := r.services.Drive.DownloadFile(ctx, fileID, localPath)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.uploadFile
	r.server.AddTool(
		mcp.NewTool("drive.uploadFile",
			mcp.WithDescription("Uploads a local file to Google Drive."),
			mcp.WithString("localPath", mcp.Required(), mcp.Description("The local path of the file to upload")),
			mcp.WithString("fileName", mcp.Description("Override file name in Drive")),
			mcp.WithString("folderId", mcp.Description("The ID of the destination folder")),
			mcp.WithString("mimeType", mcp.Description("MIME type of the file")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			localPath := args["localPath"].(string)
			var fileName, folderID, mimeType *string
			if v, ok := args["fileName"].(string); ok && v != "" {
				fileName = &v
			}
			if v, ok := args["folderId"].(string); ok && v != "" {
				folderID = &v
			}
			if v, ok := args["mimeType"].(string); ok && v != "" {
				mimeType = &v
			}
			resp := r.services.Drive.UploadFile(ctx, localPath, fileName, folderID, mimeType)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.copyFile
	r.server.AddTool(
		mcp.NewTool("drive.copyFile",
			mcp.WithDescription("Creates a copy of a file."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file to copy")),
			mcp.WithString("newName", mcp.Description("Name for the copy")),
			mcp.WithString("folderId", mcp.Description("Destination folder ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			var newName, folderID *string
			if v, ok := args["newName"].(string); ok && v != "" {
				newName = &v
			}
			if v, ok := args["folderId"].(string); ok && v != "" {
				folderID = &v
			}
			resp := r.services.Drive.CopyFile(ctx, fileID, newName, folderID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.moveFile
	r.server.AddTool(
		mcp.NewTool("drive.moveFile",
			mcp.WithDescription("Moves a file to a different folder."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file to move")),
			mcp.WithString("folderId", mcp.Required(), mcp.Description("The ID of the destination folder")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			folderID := args["folderId"].(string)
			resp := r.services.Drive.MoveFile(ctx, fileID, folderID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.deleteFile
	r.server.AddTool(
		mcp.NewTool("drive.deleteFile",
			mcp.WithDescription("Moves a file to trash or permanently deletes it."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file to delete")),
			mcp.WithBoolean("permanent", mcp.Description("If true, permanently delete (default: move to trash)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			permanent := false
			if v, ok := args["permanent"].(bool); ok {
				permanent = v
			}
			resp := r.services.Drive.DeleteFile(ctx, fileID, permanent)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.getFileInfo
	r.server.AddTool(
		mcp.NewTool("drive.getFileInfo",
			mcp.WithDescription("Gets detailed information about a file."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			resp := r.services.Drive.GetFileInfo(ctx, fileID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.shareFile
	r.server.AddTool(
		mcp.NewTool("drive.shareFile",
			mcp.WithDescription("Shares a file with a user or makes it public."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file to share")),
			mcp.WithString("role", mcp.Required(), mcp.Description("Permission role: reader, writer, or commenter")),
			mcp.WithString("type", mcp.Required(), mcp.Description("Permission type: user, group, domain, or anyone")),
			mcp.WithString("email", mcp.Description("Email address (required for user/group type)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			role := args["role"].(string)
			shareType := args["type"].(string)
			var email *string
			if v, ok := args["email"].(string); ok && v != "" {
				email = &v
			}
			resp := r.services.Drive.ShareFile(ctx, fileID, email, role, shareType)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.removeShare
	r.server.AddTool(
		mcp.NewTool("drive.removeShare",
			mcp.WithDescription("Removes a sharing permission from a file."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
			mcp.WithString("permissionId", mcp.Required(), mcp.Description("The ID of the permission to remove")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			permissionID := args["permissionId"].(string)
			resp := r.services.Drive.RemoveShare(ctx, fileID, permissionID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.listTrash
	r.server.AddTool(
		mcp.NewTool("drive.listTrash",
			mcp.WithDescription("Lists files in trash."),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			var pageToken *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Drive.ListTrash(ctx, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.restoreFile
	r.server.AddTool(
		mcp.NewTool("drive.restoreFile",
			mcp.WithDescription("Restores a file from trash."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file to restore")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			resp := r.services.Drive.RestoreFile(ctx, fileID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.emptyTrash
	r.server.AddTool(
		mcp.NewTool("drive.emptyTrash",
			mcp.WithDescription("Permanently deletes all files in trash."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Drive.EmptyTrash(ctx)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.exportFile
	r.server.AddTool(
		mcp.NewTool("drive.exportFile",
			mcp.WithDescription("Exports a Google Workspace file (Docs, Sheets, Slides) to PDF or other formats."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file to export")),
			mcp.WithString("mimeType", mcp.Required(), mcp.Description("Export format MIME type (e.g., application/pdf)")),
			mcp.WithString("localPath", mcp.Required(), mcp.Description("Local path to save the exported file")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			mimeType := args["mimeType"].(string)
			localPath := args["localPath"].(string)
			resp := r.services.Drive.ExportFile(ctx, fileID, mimeType, localPath)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.getExportFormats
	r.server.AddTool(
		mcp.NewTool("drive.getExportFormats",
			mcp.WithDescription("Gets available export formats for a Google Workspace file."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			resp := r.services.Drive.GetExportFormats(ctx, fileID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.listComments
	r.server.AddTool(
		mcp.NewTool("drive.listComments",
			mcp.WithDescription("Lists comments on a file."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			var pageToken *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Drive.ListComments(ctx, fileID, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.createComment
	r.server.AddTool(
		mcp.NewTool("drive.createComment",
			mcp.WithDescription("Creates a comment on a file."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
			mcp.WithString("content", mcp.Required(), mcp.Description("The comment content")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			content := args["content"].(string)
			resp := r.services.Drive.CreateComment(ctx, fileID, content)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.replyToComment
	r.server.AddTool(
		mcp.NewTool("drive.replyToComment",
			mcp.WithDescription("Replies to a comment on a file."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
			mcp.WithString("commentId", mcp.Required(), mcp.Description("The ID of the comment")),
			mcp.WithString("content", mcp.Required(), mcp.Description("The reply content")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			commentID := args["commentId"].(string)
			content := args["content"].(string)
			resp := r.services.Drive.ReplyToComment(ctx, fileID, commentID, content)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.resolveComment
	r.server.AddTool(
		mcp.NewTool("drive.resolveComment",
			mcp.WithDescription("Resolves or reopens a comment."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
			mcp.WithString("commentId", mcp.Required(), mcp.Description("The ID of the comment")),
			mcp.WithBoolean("resolved", mcp.Required(), mcp.Description("True to resolve, false to reopen")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			commentID := args["commentId"].(string)
			resolved := args["resolved"].(bool)
			resp := r.services.Drive.ResolveComment(ctx, fileID, commentID, resolved)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.listRevisions
	r.server.AddTool(
		mcp.NewTool("drive.listRevisions",
			mcp.WithDescription("Lists file revisions (version history)."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			var pageToken *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Drive.ListRevisions(ctx, fileID, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// drive.getRevision
	r.server.AddTool(
		mcp.NewTool("drive.getRevision",
			mcp.WithDescription("Gets a specific file revision."),
			mcp.WithString("fileId", mcp.Required(), mcp.Description("The ID of the file")),
			mcp.WithString("revisionId", mcp.Required(), mcp.Description("The ID of the revision")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			fileID := args["fileId"].(string)
			revisionID := args["revisionId"].(string)
			resp := r.services.Drive.GetRevision(ctx, fileID, revisionID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
