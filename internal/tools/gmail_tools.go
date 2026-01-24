// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerGmailTools() {
	// gmail.search
	r.server.AddTool(
		mcp.NewTool("gmail.search",
			mcp.WithDescription("Searches for emails using Gmail query syntax."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Gmail search query")),
			mcp.WithNumber("maxResults", mcp.Description("Maximum number of results")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			query := args["query"].(string)
			var maxResults *int
			var pageToken *string
			if v, ok := args["maxResults"].(float64); ok {
				mr := int(v)
				maxResults = &mr
			}
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			resp := r.services.Gmail.SearchSimple(ctx, query, maxResults, pageToken)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.get
	r.server.AddTool(
		mcp.NewTool("gmail.get",
			mcp.WithDescription("Gets a specific email message with full content."),
			mcp.WithString("messageId", mcp.Required(), mcp.Description("The ID of the message")),
			mcp.WithString("format", mcp.Description("Format: minimal, full, raw, or metadata")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			messageID := args["messageId"].(string)
			var format *string
			if v, ok := args["format"].(string); ok && v != "" {
				format = &v
			}
			resp := r.services.Gmail.Get(ctx, messageID, format)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.downloadAttachment
	r.server.AddTool(
		mcp.NewTool("gmail.downloadAttachment",
			mcp.WithDescription("Downloads an email attachment to a local file."),
			mcp.WithString("messageId", mcp.Required(), mcp.Description("The ID of the message")),
			mcp.WithString("attachmentId", mcp.Required(), mcp.Description("The ID of the attachment")),
			mcp.WithString("localPath", mcp.Required(), mcp.Description("Absolute path to save the file")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			messageID := args["messageId"].(string)
			attachmentID := args["attachmentId"].(string)
			localPath := args["localPath"].(string)
			resp := r.services.Gmail.DownloadAttachment(ctx, messageID, attachmentID, localPath)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.modify
	r.server.AddTool(
		mcp.NewTool("gmail.modify",
			mcp.WithDescription("Modifies message labels (add or remove)."),
			mcp.WithString("messageId", mcp.Required(), mcp.Description("The ID of the message")),
			mcp.WithString("addLabelIds", mcp.Description("Comma-separated label IDs to add")),
			mcp.WithString("removeLabelIds", mcp.Description("Comma-separated label IDs to remove")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			messageID := args["messageId"].(string)
			var addLabelIDs, removeLabelIDs []string
			if v, ok := args["addLabelIds"].(string); ok && v != "" {
				addLabelIDs = strings.Split(v, ",")
			}
			if v, ok := args["removeLabelIds"].(string); ok && v != "" {
				removeLabelIDs = strings.Split(v, ",")
			}
			resp := r.services.Gmail.Modify(ctx, messageID, addLabelIDs, removeLabelIDs)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.send
	r.server.AddTool(
		mcp.NewTool("gmail.send",
			mcp.WithDescription("Sends an email message."),
			mcp.WithString("to", mcp.Required(), mcp.Description("Recipient email address")),
			mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Email body")),
			mcp.WithString("cc", mcp.Description("CC recipients (comma-separated)")),
			mcp.WithString("bcc", mcp.Description("BCC recipients (comma-separated)")),
			mcp.WithString("threadId", mcp.Description("Thread ID for replying to existing thread")),
			mcp.WithString("inReplyTo", mcp.Description("Message-ID of the message being replied to")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			to := args["to"].(string)
			subject := args["subject"].(string)
			body := args["body"].(string)
			var cc, bcc, threadID, inReplyTo *string
			if v, ok := args["cc"].(string); ok && v != "" {
				cc = &v
			}
			if v, ok := args["bcc"].(string); ok && v != "" {
				bcc = &v
			}
			if v, ok := args["threadId"].(string); ok && v != "" {
				threadID = &v
			}
			if v, ok := args["inReplyTo"].(string); ok && v != "" {
				inReplyTo = &v
			}
			resp := r.services.Gmail.SendSimple(ctx, to, subject, body, cc, bcc, threadID, inReplyTo)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.createDraft
	r.server.AddTool(
		mcp.NewTool("gmail.createDraft",
			mcp.WithDescription("Creates an email draft."),
			mcp.WithString("to", mcp.Required(), mcp.Description("Recipient email address")),
			mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Email body")),
			mcp.WithString("cc", mcp.Description("CC recipients (comma-separated)")),
			mcp.WithString("bcc", mcp.Description("BCC recipients (comma-separated)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			to := args["to"].(string)
			subject := args["subject"].(string)
			body := args["body"].(string)
			var cc, bcc *string
			if v, ok := args["cc"].(string); ok && v != "" {
				cc = &v
			}
			if v, ok := args["bcc"].(string); ok && v != "" {
				bcc = &v
			}
			resp := r.services.Gmail.CreateDraftSimple(ctx, to, subject, body, cc, bcc)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.sendDraft
	r.server.AddTool(
		mcp.NewTool("gmail.sendDraft",
			mcp.WithDescription("Sends a previously created draft."),
			mcp.WithString("draftId", mcp.Required(), mcp.Description("The ID of the draft to send")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			draftID := args["draftId"].(string)
			resp := r.services.Gmail.SendDraft(ctx, draftID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.listLabels
	r.server.AddTool(
		mcp.NewTool("gmail.listLabels",
			mcp.WithDescription("Lists all Gmail labels."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Gmail.ListLabels(ctx)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
