// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"

	"google.golang.org/api/forms/v1"
)

// FormsService provides Google Forms operations
type FormsService struct {
	auth AuthProvider
}

// NewFormsService creates a new FormsService
func NewFormsService(auth AuthProvider) *FormsService {
	return &FormsService{auth: auth}
}

// getClient creates a new Forms API client
func (s *FormsService) getClient(ctx context.Context) (*forms.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return forms.NewService(ctx, opt)
}

// FormInfo represents a form
type FormInfo struct {
	FormID       string          `json:"formId"`
	Title        string          `json:"title"`
	Description  string          `json:"description,omitempty"`
	DocumentTitle string         `json:"documentTitle,omitempty"`
	LinkedSheetID string         `json:"linkedSheetId,omitempty"`
	ResponderURL string          `json:"responderUrl,omitempty"`
	Items        []FormItemInfo  `json:"items,omitempty"`
}

// FormItemInfo represents a form item
type FormItemInfo struct {
	ItemID      string `json:"itemId"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
}

// FormResponseInfo represents a form response
type FormResponseInfo struct {
	ResponseID   string                 `json:"responseId"`
	CreateTime   string                 `json:"createTime"`
	LastSubmitted string                `json:"lastSubmittedTime"`
	RespondentEmail string              `json:"respondentEmail,omitempty"`
	Answers      map[string]interface{} `json:"answers,omitempty"`
}

// GetForm returns form metadata and structure
func (s *FormsService) GetForm(ctx context.Context, formID string) (*FormInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	form, err := client.Forms.Get(formID).Do()
	if err != nil {
		return nil, err
	}

	info := &FormInfo{
		FormID:        form.FormId,
		Title:         form.Info.Title,
		DocumentTitle: form.Info.DocumentTitle,
		ResponderURL:  form.ResponderUri,
	}

	if form.Info.Description != "" {
		info.Description = form.Info.Description
	}

	if form.LinkedSheetId != "" {
		info.LinkedSheetID = form.LinkedSheetId
	}

	// Parse items
	for _, item := range form.Items {
		itemInfo := FormItemInfo{
			ItemID: item.ItemId,
			Title:  item.Title,
		}
		if item.Description != "" {
			itemInfo.Description = item.Description
		}

		// Determine item type
		switch {
		case item.QuestionItem != nil:
			if item.QuestionItem.Question.TextQuestion != nil {
				itemInfo.Type = "text"
			} else if item.QuestionItem.Question.ChoiceQuestion != nil {
				itemInfo.Type = "choice"
			} else if item.QuestionItem.Question.ScaleQuestion != nil {
				itemInfo.Type = "scale"
			} else if item.QuestionItem.Question.DateQuestion != nil {
				itemInfo.Type = "date"
			} else if item.QuestionItem.Question.TimeQuestion != nil {
				itemInfo.Type = "time"
			} else if item.QuestionItem.Question.FileUploadQuestion != nil {
				itemInfo.Type = "fileUpload"
			} else {
				itemInfo.Type = "question"
			}
		case item.QuestionGroupItem != nil:
			itemInfo.Type = "questionGroup"
		case item.PageBreakItem != nil:
			itemInfo.Type = "pageBreak"
		case item.TextItem != nil:
			itemInfo.Type = "text"
		case item.ImageItem != nil:
			itemInfo.Type = "image"
		case item.VideoItem != nil:
			itemInfo.Type = "video"
		default:
			itemInfo.Type = "unknown"
		}

		info.Items = append(info.Items, itemInfo)
	}

	return info, nil
}

// ListResponses returns all responses to a form
func (s *FormsService) ListResponses(ctx context.Context, formID string, pageSize int64, pageToken string) ([]FormResponseInfo, string, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, "", err
	}

	call := client.Forms.Responses.List(formID)
	if pageSize > 0 {
		call = call.PageSize(pageSize)
	}
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, "", err
	}

	var responses []FormResponseInfo
	for _, r := range resp.Responses {
		respInfo := FormResponseInfo{
			ResponseID:    r.ResponseId,
			CreateTime:    r.CreateTime,
			LastSubmitted: r.LastSubmittedTime,
		}
		if r.RespondentEmail != "" {
			respInfo.RespondentEmail = r.RespondentEmail
		}

		// Parse answers
		if len(r.Answers) > 0 {
			respInfo.Answers = make(map[string]interface{})
			for questionID, answer := range r.Answers {
				if answer.TextAnswers != nil {
					var values []string
					for _, ta := range answer.TextAnswers.Answers {
						values = append(values, ta.Value)
					}
					respInfo.Answers[questionID] = values
				}
			}
		}

		responses = append(responses, respInfo)
	}

	return responses, resp.NextPageToken, nil
}

// GetResponse returns a specific form response
func (s *FormsService) GetResponse(ctx context.Context, formID, responseID string) (*FormResponseInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	r, err := client.Forms.Responses.Get(formID, responseID).Do()
	if err != nil {
		return nil, err
	}

	respInfo := &FormResponseInfo{
		ResponseID:    r.ResponseId,
		CreateTime:    r.CreateTime,
		LastSubmitted: r.LastSubmittedTime,
	}
	if r.RespondentEmail != "" {
		respInfo.RespondentEmail = r.RespondentEmail
	}

	// Parse answers
	if len(r.Answers) > 0 {
		respInfo.Answers = make(map[string]interface{})
		for questionID, answer := range r.Answers {
			if answer.TextAnswers != nil {
				var values []string
				for _, ta := range answer.TextAnswers.Answers {
					values = append(values, ta.Value)
				}
				respInfo.Answers[questionID] = values
			}
		}
	}

	return respInfo, nil
}

// CreateForm creates a new form
func (s *FormsService) CreateForm(ctx context.Context, title, documentTitle string) (*FormInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	form := &forms.Form{
		Info: &forms.Info{
			Title:         title,
			DocumentTitle: documentTitle,
		},
	}

	created, err := client.Forms.Create(form).Do()
	if err != nil {
		return nil, err
	}

	return &FormInfo{
		FormID:        created.FormId,
		Title:         created.Info.Title,
		DocumentTitle: created.Info.DocumentTitle,
		ResponderURL:  created.ResponderUri,
	}, nil
}

// UpdateFormInfo updates form title and description
func (s *FormsService) UpdateFormInfo(ctx context.Context, formID, title, description string) (*FormInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	var requests []*forms.Request

	if title != "" {
		requests = append(requests, &forms.Request{
			UpdateFormInfo: &forms.UpdateFormInfoRequest{
				Info: &forms.Info{
					Title: title,
				},
				UpdateMask: "title",
			},
		})
	}

	if description != "" {
		requests = append(requests, &forms.Request{
			UpdateFormInfo: &forms.UpdateFormInfoRequest{
				Info: &forms.Info{
					Description: description,
				},
				UpdateMask: "description",
			},
		})
	}

	if len(requests) == 0 {
		return s.GetForm(ctx, formID)
	}

	_, err = client.Forms.BatchUpdate(formID, &forms.BatchUpdateFormRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return nil, err
	}

	return s.GetForm(ctx, formID)
}
