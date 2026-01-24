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
}
