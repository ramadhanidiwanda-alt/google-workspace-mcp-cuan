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

	// gmail.sendWithAttachments
	r.server.AddTool(
		mcp.NewTool("gmail.sendWithAttachments",
			mcp.WithDescription("Sends an email with file attachments."),
			mcp.WithString("to", mcp.Required(), mcp.Description("Recipient email address")),
			mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Email body")),
			mcp.WithArray("attachmentPaths", mcp.Required(), mcp.Description("Array of local file paths to attach")),
			mcp.WithBoolean("isHtml", mcp.Description("Whether body is HTML content")),
			mcp.WithString("cc", mcp.Description("CC recipients (comma-separated)")),
			mcp.WithString("bcc", mcp.Description("BCC recipients (comma-separated)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			to := args["to"].(string)
			subject := args["subject"].(string)
			body := args["body"].(string)
			rawPaths := args["attachmentPaths"].([]interface{})
			paths := make([]string, len(rawPaths))
			for i, p := range rawPaths {
				paths[i] = p.(string)
			}
			isHTML := false
			if v, ok := args["isHtml"].(bool); ok {
				isHTML = v
			}
			var cc, bcc *string
			if v, ok := args["cc"].(string); ok && v != "" {
				cc = &v
			}
			if v, ok := args["bcc"].(string); ok && v != "" {
				bcc = &v
			}
			resp := r.services.Gmail.SendWithAttachments(ctx, to, subject, body, paths, isHTML, cc, bcc)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.createLabel
	r.server.AddTool(
		mcp.NewTool("gmail.createLabel",
			mcp.WithDescription("Creates a new Gmail label."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Label name")),
			mcp.WithString("labelListVisibility", mcp.Description("Label visibility: labelShow, labelShowIfUnread, labelHide")),
			mcp.WithString("messageListVisibility", mcp.Description("Message list visibility: show, hide")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			name := args["name"].(string)
			var labelListVisibility, messageListVisibility *string
			if v, ok := args["labelListVisibility"].(string); ok && v != "" {
				labelListVisibility = &v
			}
			if v, ok := args["messageListVisibility"].(string); ok && v != "" {
				messageListVisibility = &v
			}
			resp := r.services.Gmail.CreateLabel(ctx, name, labelListVisibility, messageListVisibility)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.deleteLabel
	r.server.AddTool(
		mcp.NewTool("gmail.deleteLabel",
			mcp.WithDescription("Deletes a Gmail label."),
			mcp.WithString("labelId", mcp.Required(), mcp.Description("The ID of the label to delete")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			labelID := args["labelId"].(string)
			resp := r.services.Gmail.DeleteLabel(ctx, labelID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.trashMessage
	r.server.AddTool(
		mcp.NewTool("gmail.trashMessage",
			mcp.WithDescription("Moves a message to trash."),
			mcp.WithString("messageId", mcp.Required(), mcp.Description("The ID of the message")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			messageID := args["messageId"].(string)
			resp := r.services.Gmail.TrashMessage(ctx, messageID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.untrashMessage
	r.server.AddTool(
		mcp.NewTool("gmail.untrashMessage",
			mcp.WithDescription("Removes a message from trash."),
			mcp.WithString("messageId", mcp.Required(), mcp.Description("The ID of the message")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			messageID := args["messageId"].(string)
			resp := r.services.Gmail.UntrashMessage(ctx, messageID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.getVacationSettings
	r.server.AddTool(
		mcp.NewTool("gmail.getVacationSettings",
			mcp.WithDescription("Gets vacation auto-reply settings."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Gmail.GetVacationSettings(ctx)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.setVacationSettings
	r.server.AddTool(
		mcp.NewTool("gmail.setVacationSettings",
			mcp.WithDescription("Sets vacation auto-reply settings."),
			mcp.WithBoolean("enable", mcp.Required(), mcp.Description("Enable or disable auto-reply")),
			mcp.WithString("subject", mcp.Description("Auto-reply subject")),
			mcp.WithString("body", mcp.Description("Auto-reply body (HTML)")),
			mcp.WithNumber("startTime", mcp.Description("Start time (Unix timestamp in ms)")),
			mcp.WithNumber("endTime", mcp.Description("End time (Unix timestamp in ms)")),
			mcp.WithBoolean("restrictToContacts", mcp.Description("Only reply to contacts")),
			mcp.WithBoolean("restrictToDomain", mcp.Description("Only reply to same domain")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			enable := args["enable"].(bool)
			var subject, body *string
			var startTime, endTime *int64
			var restrictToContacts, restrictToDomain *bool
			if v, ok := args["subject"].(string); ok && v != "" {
				subject = &v
			}
			if v, ok := args["body"].(string); ok && v != "" {
				body = &v
			}
			if v, ok := args["startTime"].(float64); ok {
				st := int64(v)
				startTime = &st
			}
			if v, ok := args["endTime"].(float64); ok {
				et := int64(v)
				endTime = &et
			}
			if v, ok := args["restrictToContacts"].(bool); ok {
				restrictToContacts = &v
			}
			if v, ok := args["restrictToDomain"].(bool); ok {
				restrictToDomain = &v
			}
			resp := r.services.Gmail.SetVacationSettings(ctx, enable, subject, body, startTime, endTime, restrictToContacts, restrictToDomain)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.listFilters
	r.server.AddTool(
		mcp.NewTool("gmail.listFilters",
			mcp.WithDescription("Lists all Gmail filters."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Gmail.ListFilters(ctx)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.createFilter
	r.server.AddTool(
		mcp.NewTool("gmail.createFilter",
			mcp.WithDescription("Creates a new Gmail filter."),
			mcp.WithString("from", mcp.Description("Filter emails from this sender")),
			mcp.WithString("to", mcp.Description("Filter emails to this recipient")),
			mcp.WithString("subject", mcp.Description("Filter emails with this subject")),
			mcp.WithString("query", mcp.Description("Filter using Gmail search syntax")),
			mcp.WithBoolean("hasAttachment", mcp.Description("Filter emails with attachments")),
			mcp.WithString("addLabelIds", mcp.Description("Comma-separated label IDs to add")),
			mcp.WithString("removeLabelIds", mcp.Description("Comma-separated label IDs to remove")),
			mcp.WithString("forward", mcp.Description("Email address to forward to")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			var from, to, subject, query, forward *string
			var hasAttachment *bool
			var addLabelIDs, removeLabelIDs []string

			if v, ok := args["from"].(string); ok && v != "" {
				from = &v
			}
			if v, ok := args["to"].(string); ok && v != "" {
				to = &v
			}
			if v, ok := args["subject"].(string); ok && v != "" {
				subject = &v
			}
			if v, ok := args["query"].(string); ok && v != "" {
				query = &v
			}
			if v, ok := args["hasAttachment"].(bool); ok {
				hasAttachment = &v
			}
			if v, ok := args["addLabelIds"].(string); ok && v != "" {
				addLabelIDs = strings.Split(v, ",")
			}
			if v, ok := args["removeLabelIds"].(string); ok && v != "" {
				removeLabelIDs = strings.Split(v, ",")
			}
			if v, ok := args["forward"].(string); ok && v != "" {
				forward = &v
			}

			resp := r.services.Gmail.CreateFilter(ctx, from, to, subject, query, hasAttachment, addLabelIDs, removeLabelIDs, forward)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.deleteFilter
	r.server.AddTool(
		mcp.NewTool("gmail.deleteFilter",
			mcp.WithDescription("Deletes a Gmail filter."),
			mcp.WithString("filterId", mcp.Required(), mcp.Description("The ID of the filter to delete")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			filterID := args["filterId"].(string)
			resp := r.services.Gmail.DeleteFilter(ctx, filterID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.getSendAs
	r.server.AddTool(
		mcp.NewTool("gmail.getSendAs",
			mcp.WithDescription("Gets send-as settings (signature, etc.). Lists all if no email specified."),
			mcp.WithString("sendAsEmail", mcp.Description("Specific send-as email to get")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			var sendAsEmail *string
			if v, ok := args["sendAsEmail"].(string); ok && v != "" {
				sendAsEmail = &v
			}
			resp := r.services.Gmail.GetSendAs(ctx, sendAsEmail)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// gmail.updateSignature
	r.server.AddTool(
		mcp.NewTool("gmail.updateSignature",
			mcp.WithDescription("Updates the email signature for a send-as address."),
			mcp.WithString("sendAsEmail", mcp.Required(), mcp.Description("The send-as email address")),
			mcp.WithString("signature", mcp.Required(), mcp.Description("The signature HTML content")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			sendAsEmail := args["sendAsEmail"].(string)
			signature := args["signature"].(string)
			resp := r.services.Gmail.UpdateSignature(ctx, sendAsEmail, signature)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
