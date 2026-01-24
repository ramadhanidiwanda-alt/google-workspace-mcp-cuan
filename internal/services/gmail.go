// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oowada/google-workspace-mcp/internal/util"
	"google.golang.org/api/gmail/v1"
)

// GmailService provides Gmail operations
type GmailService struct {
	auth AuthProvider
}

// NewGmailService creates a new GmailService instance
func NewGmailService(auth AuthProvider) *GmailService {
	return &GmailService{auth: auth}
}

// getGmailClient returns an authenticated Gmail client
func (s *GmailService) getGmailClient(ctx context.Context) (*gmail.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return gmail.NewService(ctx, opt)
}

// SearchInput contains input for Search
type GmailSearchInput struct {
	Query            string   `json:"query,omitempty" jsonschema:"Gmail search query syntax"`
	MaxResults       *int     `json:"maxResults,omitempty" jsonschema:"Maximum number of results"`
	PageToken        *string  `json:"pageToken,omitempty" jsonschema:"Token for pagination"`
	LabelIDs         []string `json:"labelIds,omitempty" jsonschema:"Filter by label IDs"`
	IncludeSpamTrash *bool    `json:"includeSpamTrash,omitempty" jsonschema:"Include spam and trash messages"`
}

// Search searches for emails
func (s *GmailService) Search(ctx context.Context, input GmailSearchInput) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.Users.Messages.List("me")

	if input.Query != "" {
		req.Q(input.Query)
	}

	if input.MaxResults != nil {
		req.MaxResults(int64(*input.MaxResults))
	} else {
		req.MaxResults(20)
	}

	if input.PageToken != nil && *input.PageToken != "" {
		req.PageToken(*input.PageToken)
	}

	if len(input.LabelIDs) > 0 {
		req.LabelIds(input.LabelIDs...)
	}

	if input.IncludeSpamTrash != nil && *input.IncludeSpamTrash {
		req.IncludeSpamTrash(true)
	}

	result, err := req.Do()
	if err != nil {
		return ErrorResponse(err)
	}

	messages := make([]map[string]string, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = map[string]string{
			"id":       msg.Id,
			"threadId": msg.ThreadId,
		}
	}

	response := map[string]interface{}{
		"messages":           messages,
		"resultSizeEstimate": result.ResultSizeEstimate,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// Get retrieves a specific message
func (s *GmailService) Get(ctx context.Context, messageID string, format *string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	msgFormat := "full"
	if format != nil && *format != "" {
		msgFormat = *format
	}

	msg, err := client.Users.Messages.Get("me", messageID).Format(msgFormat).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Extract headers
	headers := make(map[string]string)
	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from", "to", "subject", "date", "cc", "bcc":
			headers[h.Name] = h.Value
		}
	}

	// Extract body
	body := extractMessageBody(msg.Payload)

	// Extract attachments info
	attachments := extractAttachmentInfo(msg.Payload)

	return JSONResponse(map[string]interface{}{
		"id":          msg.Id,
		"threadId":    msg.ThreadId,
		"labelIds":    msg.LabelIds,
		"snippet":     msg.Snippet,
		"headers":     headers,
		"body":        body,
		"attachments": attachments,
	})
}

// DownloadAttachment downloads an email attachment
func (s *GmailService) DownloadAttachment(ctx context.Context, messageID string, attachmentID string, localPath string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	if !filepath.IsAbs(localPath) {
		return ErrorResponse(fmt.Errorf("localPath must be an absolute path"))
	}

	attachment, err := client.Users.Messages.Attachments.Get("me", messageID, attachmentID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Decode base64url data
	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return ErrorResponse(err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return ErrorResponse(err)
	}

	// Write file
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return ErrorResponse(err)
	}

	util.LogDebug("Downloaded attachment to %s (%d bytes)", localPath, len(data))

	return JSONResponse(map[string]interface{}{
		"localPath": localPath,
		"size":      len(data),
	})
}

// Modify modifies message labels
func (s *GmailService) Modify(ctx context.Context, messageID string, addLabelIDs []string, removeLabelIDs []string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	_, err = client.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		AddLabelIds:    addLabelIDs,
		RemoveLabelIds: removeLabelIDs,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"status":    "success",
		"messageId": messageID,
	})
}

// SendEmailInput contains input for Send
type SendEmailInput struct {
	To      []string `json:"to" jsonschema:"Recipient email addresses"`
	Subject string   `json:"subject" jsonschema:"Email subject line"`
	Body    string   `json:"body" jsonschema:"Email body content"`
	CC      []string `json:"cc,omitempty" jsonschema:"CC recipient email addresses"`
	BCC     []string `json:"bcc,omitempty" jsonschema:"BCC recipient email addresses"`
	IsHTML  *bool    `json:"isHtml,omitempty" jsonschema:"Whether body is HTML content"`
}

// SearchSimple searches for emails with simple parameters
func (s *GmailService) SearchSimple(ctx context.Context, query string, maxResults *int, pageToken *string) ToolResponse {
	return s.Search(ctx, GmailSearchInput{
		Query:      query,
		MaxResults: maxResults,
		PageToken:  pageToken,
	})
}

// SendSimple sends an email with simple parameters
func (s *GmailService) SendSimple(ctx context.Context, to, subject, body string, cc, bcc, threadID, inReplyTo *string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Build MIME message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != nil && *cc != "" {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", *cc))
	}
	if bcc != nil && *bcc != "" {
		msg.WriteString(fmt.Sprintf("Bcc: %s\r\n", *bcc))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	if inReplyTo != nil && *inReplyTo != "" {
		msg.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", *inReplyTo))
		msg.WriteString(fmt.Sprintf("References: %s\r\n", *inReplyTo))
	}
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	message := &gmail.Message{Raw: raw}
	if threadID != nil && *threadID != "" {
		message.ThreadId = *threadID
	}

	result, err := client.Users.Messages.Send("me", message).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"messageId": result.Id,
		"threadId":  result.ThreadId,
	})
}

// CreateDraftSimple creates an email draft with simple parameters
func (s *GmailService) CreateDraftSimple(ctx context.Context, to, subject, body string, cc, bcc *string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != nil && *cc != "" {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", *cc))
	}
	if bcc != nil && *bcc != "" {
		msg.WriteString(fmt.Sprintf("Bcc: %s\r\n", *bcc))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	result, err := client.Users.Drafts.Create("me", &gmail.Draft{
		Message: &gmail.Message{Raw: raw},
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"draftId":   result.Id,
		"messageId": result.Message.Id,
	})
}

// Send sends an email
func (s *GmailService) Send(ctx context.Context, input SendEmailInput) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Build MIME message
	contentType := "text/plain"
	if input.IsHTML != nil && *input.IsHTML {
		contentType = "text/html"
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(input.To, ", ")))
	if len(input.CC) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(input.CC, ", ")))
	}
	if len(input.BCC) > 0 {
		msg.WriteString(fmt.Sprintf("Bcc: %s\r\n", strings.Join(input.BCC, ", ")))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", input.Subject))
	msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType))
	msg.WriteString("\r\n")
	msg.WriteString(input.Body)

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	result, err := client.Users.Messages.Send("me", &gmail.Message{
		Raw: raw,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"messageId": result.Id,
		"threadId":  result.ThreadId,
	})
}

// CreateDraft creates an email draft
func (s *GmailService) CreateDraft(ctx context.Context, input SendEmailInput) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Build MIME message (same as Send)
	contentType := "text/plain"
	if input.IsHTML != nil && *input.IsHTML {
		contentType = "text/html"
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(input.To, ", ")))
	if len(input.CC) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(input.CC, ", ")))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", input.Subject))
	msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType))
	msg.WriteString("\r\n")
	msg.WriteString(input.Body)

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	result, err := client.Users.Drafts.Create("me", &gmail.Draft{
		Message: &gmail.Message{Raw: raw},
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"draftId":   result.Id,
		"messageId": result.Message.Id,
	})
}

// SendDraft sends a previously created draft
func (s *GmailService) SendDraft(ctx context.Context, draftID string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	result, err := client.Users.Drafts.Send("me", &gmail.Draft{
		Id: draftID,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"messageId": result.Id,
		"threadId":  result.ThreadId,
	})
}

// ListLabels lists all Gmail labels
func (s *GmailService) ListLabels(ctx context.Context) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	result, err := client.Users.Labels.List("me").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	labels := make([]map[string]interface{}, len(result.Labels))
	for i, label := range result.Labels {
		labels[i] = map[string]interface{}{
			"id":                    label.Id,
			"name":                  label.Name,
			"type":                  label.Type,
			"messageListVisibility": label.MessageListVisibility,
			"labelListVisibility":   label.LabelListVisibility,
		}
	}

	return JSONResponse(map[string]interface{}{
		"labels": labels,
	})
}

// Helper functions

func extractMessageBody(payload *gmail.MessagePart) string {
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	// Check parts recursively
	for _, part := range payload.Parts {
		if strings.HasPrefix(part.MimeType, "text/") {
			body := extractMessageBody(part)
			if body != "" {
				return body
			}
		}
	}

	return ""
}

func extractAttachmentInfo(payload *gmail.MessagePart) []map[string]interface{} {
	var attachments []map[string]interface{}

	if payload.Filename != "" && payload.Body != nil && payload.Body.AttachmentId != "" {
		attachments = append(attachments, map[string]interface{}{
			"filename":     payload.Filename,
			"mimeType":     payload.MimeType,
			"attachmentId": payload.Body.AttachmentId,
			"size":         payload.Body.Size,
		})
	}

	for _, part := range payload.Parts {
		attachments = append(attachments, extractAttachmentInfo(part)...)
	}

	return attachments
}

// SendWithAttachments sends an email with file attachments
func (s *GmailService) SendWithAttachments(ctx context.Context, to, subject, body string, attachmentPaths []string, isHTML bool, cc, bcc *string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	boundary := "boundary_" + fmt.Sprintf("%d", generateMessageID())

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != nil && *cc != "" {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", *cc))
	}
	if bcc != nil && *bcc != "" {
		msg.WriteString(fmt.Sprintf("Bcc: %s\r\n", *bcc))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// Body part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}
	msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType))
	msg.WriteString("\r\n")
	msg.WriteString(body)
	msg.WriteString("\r\n")

	// Attachment parts
	for _, path := range attachmentPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return ErrorResponse(fmt.Errorf("failed to read attachment %s: %w", path, err))
		}

		filename := filepath.Base(path)
		encoded := base64.StdEncoding.EncodeToString(data)

		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString(fmt.Sprintf("Content-Type: application/octet-stream; name=\"%s\"\r\n", filename))
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filename))
		msg.WriteString("\r\n")

		// Write base64 data in chunks of 76 characters
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			msg.WriteString(encoded[i:end])
			msg.WriteString("\r\n")
		}
	}

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	result, err := client.Users.Messages.Send("me", &gmail.Message{
		Raw: raw,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"messageId": result.Id,
		"threadId":  result.ThreadId,
	})
}

var messageIDCounter int64

func generateMessageID() int64 {
	messageIDCounter++
	return messageIDCounter
}

// CreateLabel creates a new Gmail label
func (s *GmailService) CreateLabel(ctx context.Context, name string, labelListVisibility, messageListVisibility *string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	label := &gmail.Label{
		Name: name,
	}

	if labelListVisibility != nil && *labelListVisibility != "" {
		label.LabelListVisibility = *labelListVisibility
	}
	if messageListVisibility != nil && *messageListVisibility != "" {
		label.MessageListVisibility = *messageListVisibility
	}

	result, err := client.Users.Labels.Create("me", label).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"id":                    result.Id,
		"name":                  result.Name,
		"labelListVisibility":   result.LabelListVisibility,
		"messageListVisibility": result.MessageListVisibility,
	})
}

// DeleteLabel deletes a Gmail label
func (s *GmailService) DeleteLabel(ctx context.Context, labelID string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	err = client.Users.Labels.Delete("me", labelID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":  "success",
		"labelId": labelID,
	})
}

// GetVacationSettings gets vacation auto-reply settings
func (s *GmailService) GetVacationSettings(ctx context.Context) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	settings, err := client.Users.Settings.GetVacation("me").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"enableAutoReply":    settings.EnableAutoReply,
		"responseSubject":    settings.ResponseSubject,
		"responseBodyHtml":   settings.ResponseBodyHtml,
		"startTime":          settings.StartTime,
		"endTime":            settings.EndTime,
		"restrictToContacts": settings.RestrictToContacts,
		"restrictToDomain":   settings.RestrictToDomain,
	})
}

// SetVacationSettings sets vacation auto-reply settings
func (s *GmailService) SetVacationSettings(ctx context.Context, enable bool, subject, body *string, startTime, endTime *int64, restrictToContacts, restrictToDomain *bool) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	settings := &gmail.VacationSettings{
		EnableAutoReply: enable,
	}

	if subject != nil {
		settings.ResponseSubject = *subject
	}
	if body != nil {
		settings.ResponseBodyHtml = *body
	}
	if startTime != nil {
		settings.StartTime = *startTime
	}
	if endTime != nil {
		settings.EndTime = *endTime
	}
	if restrictToContacts != nil {
		settings.RestrictToContacts = *restrictToContacts
	}
	if restrictToDomain != nil {
		settings.RestrictToDomain = *restrictToDomain
	}

	result, err := client.Users.Settings.UpdateVacation("me", settings).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":          "success",
		"enableAutoReply": result.EnableAutoReply,
	})
}

// TrashMessage moves a message to trash
func (s *GmailService) TrashMessage(ctx context.Context, messageID string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	_, err = client.Users.Messages.Trash("me", messageID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":    "success",
		"messageId": messageID,
	})
}

// UntrashMessage removes a message from trash
func (s *GmailService) UntrashMessage(ctx context.Context, messageID string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	_, err = client.Users.Messages.Untrash("me", messageID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":    "success",
		"messageId": messageID,
	})
}

// ListFilters lists all Gmail filters
func (s *GmailService) ListFilters(ctx context.Context) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	result, err := client.Users.Settings.Filters.List("me").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	filters := make([]map[string]interface{}, len(result.Filter))
	for i, f := range result.Filter {
		filter := map[string]interface{}{
			"id": f.Id,
		}
		if f.Criteria != nil {
			filter["criteria"] = map[string]interface{}{
				"from":           f.Criteria.From,
				"to":             f.Criteria.To,
				"subject":        f.Criteria.Subject,
				"query":          f.Criteria.Query,
				"hasAttachment":  f.Criteria.HasAttachment,
				"excludeChats":   f.Criteria.ExcludeChats,
				"size":           f.Criteria.Size,
				"sizeComparison": f.Criteria.SizeComparison,
			}
		}
		if f.Action != nil {
			filter["action"] = map[string]interface{}{
				"addLabelIds":    f.Action.AddLabelIds,
				"removeLabelIds": f.Action.RemoveLabelIds,
				"forward":        f.Action.Forward,
			}
		}
		filters[i] = filter
	}

	return JSONResponse(map[string]interface{}{
		"filters": filters,
	})
}

// CreateFilter creates a new Gmail filter
func (s *GmailService) CreateFilter(ctx context.Context, from, to, subject, query *string, hasAttachment *bool, addLabelIDs, removeLabelIDs []string, forward *string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	filter := &gmail.Filter{
		Criteria: &gmail.FilterCriteria{},
		Action:   &gmail.FilterAction{},
	}

	// Set criteria
	if from != nil && *from != "" {
		filter.Criteria.From = *from
	}
	if to != nil && *to != "" {
		filter.Criteria.To = *to
	}
	if subject != nil && *subject != "" {
		filter.Criteria.Subject = *subject
	}
	if query != nil && *query != "" {
		filter.Criteria.Query = *query
	}
	if hasAttachment != nil {
		filter.Criteria.HasAttachment = *hasAttachment
	}

	// Set actions
	if len(addLabelIDs) > 0 {
		filter.Action.AddLabelIds = addLabelIDs
	}
	if len(removeLabelIDs) > 0 {
		filter.Action.RemoveLabelIds = removeLabelIDs
	}
	if forward != nil && *forward != "" {
		filter.Action.Forward = *forward
	}

	result, err := client.Users.Settings.Filters.Create("me", filter).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"id":     result.Id,
		"status": "created",
	})
}

// DeleteFilter deletes a Gmail filter
func (s *GmailService) DeleteFilter(ctx context.Context, filterID string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	err = client.Users.Settings.Filters.Delete("me", filterID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":   "deleted",
		"filterId": filterID,
	})
}

// GetSendAs gets the send-as settings (signature, etc.)
func (s *GmailService) GetSendAs(ctx context.Context, sendAsEmail *string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	if sendAsEmail == nil || *sendAsEmail == "" {
		// List all send-as addresses
		result, err := client.Users.Settings.SendAs.List("me").Do()
		if err != nil {
			return ErrorResponse(err)
		}

		sendAsList := make([]map[string]interface{}, len(result.SendAs))
		for i, sa := range result.SendAs {
			sendAsList[i] = map[string]interface{}{
				"sendAsEmail":     sa.SendAsEmail,
				"displayName":     sa.DisplayName,
				"isDefault":       sa.IsDefault,
				"isPrimary":       sa.IsPrimary,
				"signature":       sa.Signature,
				"replyToAddress":  sa.ReplyToAddress,
				"treatAsAlias":    sa.TreatAsAlias,
			}
		}

		return JSONResponse(map[string]interface{}{
			"sendAs": sendAsList,
		})
	}

	// Get specific send-as
	result, err := client.Users.Settings.SendAs.Get("me", *sendAsEmail).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"sendAsEmail":    result.SendAsEmail,
		"displayName":    result.DisplayName,
		"isDefault":      result.IsDefault,
		"isPrimary":      result.IsPrimary,
		"signature":      result.Signature,
		"replyToAddress": result.ReplyToAddress,
		"treatAsAlias":   result.TreatAsAlias,
	})
}

// UpdateSignature updates the email signature
func (s *GmailService) UpdateSignature(ctx context.Context, sendAsEmail, signature string) ToolResponse {
	client, err := s.getGmailClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Get current settings first
	current, err := client.Users.Settings.SendAs.Get("me", sendAsEmail).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Update only the signature
	current.Signature = signature

	result, err := client.Users.Settings.SendAs.Patch("me", sendAsEmail, current).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status":      "updated",
		"sendAsEmail": result.SendAsEmail,
		"signature":   result.Signature,
	})
}
