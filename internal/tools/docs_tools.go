// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerDocsTools() {
	// docs.create
	r.server.AddTool(
		mcp.NewTool("docs.create",
			mcp.WithDescription("Creates a new Google Doc. Can be blank or with Markdown content."),
			mcp.WithString("title", mcp.Required(), mcp.Description("The title for the new document")),
			mcp.WithString("folderName", mcp.Description("The name of the folder to create the document in")),
			mcp.WithString("markdown", mcp.Description("Markdown content to create the document from")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			title := args["title"].(string)
			var folderName, markdown *string
			if v, ok := args["folderName"].(string); ok && v != "" {
				folderName = &v
			}
			if v, ok := args["markdown"].(string); ok && v != "" {
				markdown = &v
			}
			resp := r.services.Docs.Create(ctx, title, folderName, markdown)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// docs.getText
	r.server.AddTool(
		mcp.NewTool("docs.getText",
			mcp.WithDescription("Retrieves the text content of a Google Doc."),
			mcp.WithString("documentId", mcp.Required(), mcp.Description("The ID of the document")),
			mcp.WithString("tabId", mcp.Description("The ID of the tab to read from")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			documentID := args["documentId"].(string)
			var tabID *string
			if v, ok := args["tabId"].(string); ok && v != "" {
				tabID = &v
			}
			resp := r.services.Docs.GetText(ctx, documentID, tabID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// docs.insertText
	r.server.AddTool(
		mcp.NewTool("docs.insertText",
			mcp.WithDescription("Inserts text at the beginning of a Google Doc."),
			mcp.WithString("documentId", mcp.Required(), mcp.Description("The ID of the document")),
			mcp.WithString("text", mcp.Required(), mcp.Description("The text to insert")),
			mcp.WithString("tabId", mcp.Description("The ID of the tab")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			documentID := args["documentId"].(string)
			text := args["text"].(string)
			var tabID *string
			if v, ok := args["tabId"].(string); ok && v != "" {
				tabID = &v
			}
			resp := r.services.Docs.InsertText(ctx, documentID, text, tabID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// docs.appendText
	r.server.AddTool(
		mcp.NewTool("docs.appendText",
			mcp.WithDescription("Appends text to the end of a Google Doc."),
			mcp.WithString("documentId", mcp.Required(), mcp.Description("The ID of the document")),
			mcp.WithString("text", mcp.Required(), mcp.Description("The text to append")),
			mcp.WithString("tabId", mcp.Description("The ID of the tab")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			documentID := args["documentId"].(string)
			text := args["text"].(string)
			var tabID *string
			if v, ok := args["tabId"].(string); ok && v != "" {
				tabID = &v
			}
			resp := r.services.Docs.AppendText(ctx, documentID, text, tabID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// docs.replaceText
	r.server.AddTool(
		mcp.NewTool("docs.replaceText",
			mcp.WithDescription("Replaces text occurrences in a Google Doc."),
			mcp.WithString("documentId", mcp.Required(), mcp.Description("The ID of the document")),
			mcp.WithString("findText", mcp.Required(), mcp.Description("The text to find")),
			mcp.WithString("replaceText", mcp.Required(), mcp.Description("The text to replace with")),
			mcp.WithString("tabId", mcp.Description("The ID of the tab")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			documentID := args["documentId"].(string)
			findText := args["findText"].(string)
			replaceText := args["replaceText"].(string)
			var tabID *string
			if v, ok := args["tabId"].(string); ok && v != "" {
				tabID = &v
			}
			resp := r.services.Docs.ReplaceText(ctx, documentID, findText, replaceText, tabID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// docs.move
	r.server.AddTool(
		mcp.NewTool("docs.move",
			mcp.WithDescription("Moves a Google Doc to a specified folder."),
			mcp.WithString("documentId", mcp.Required(), mcp.Description("The ID of the document")),
			mcp.WithString("folderName", mcp.Required(), mcp.Description("The name of the destination folder")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			documentID := args["documentId"].(string)
			folderName := args["folderName"].(string)
			resp := r.services.Docs.Move(ctx, documentID, folderName)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// docs.find
	r.server.AddTool(
		mcp.NewTool("docs.find",
			mcp.WithDescription("Searches for Google Docs by title."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
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
			resp := r.services.Docs.Find(ctx, query, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// docs.extractIdFromUrl
	r.server.AddTool(
		mcp.NewTool("docs.extractIdFromUrl",
			mcp.WithDescription("Extracts a document ID from a Google Workspace URL."),
			mcp.WithString("url", mcp.Required(), mcp.Description("The Google Workspace URL")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			url := args["url"].(string)
			resp := r.services.Docs.ExtractIDFromURL(url)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
