// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oowada/google-workspace-mcp/internal/services"
)

// registerFormsTools registers Google Forms tools
func (r *ToolRegistrar) registerFormsTools() {
	// forms.get - Get form structure and metadata
	r.server.AddTool(
		mcp.NewTool("forms_get",
			mcp.WithDescription("Gets a form's structure and metadata including all questions."),
			mcp.WithString("formId", mcp.Required(), mcp.Description("The ID of the form")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			formID := args["formId"].(string)
			form, err := r.services.Forms.GetForm(ctx, formID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(form).Content[0].Text), nil
		},
	)

	// forms.listResponses - List form responses
	r.server.AddTool(
		mcp.NewTool("forms_listResponses",
			mcp.WithDescription("Lists all responses to a form."),
			mcp.WithString("formId", mcp.Required(), mcp.Description("The ID of the form")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			formID := args["formId"].(string)
			var pageSize int64 = 0
			pageToken := ""
			if v, ok := args["pageSize"].(float64); ok {
				pageSize = int64(v)
			}
			if v, ok := args["pageToken"].(string); ok {
				pageToken = v
			}

			responses, nextPageToken, err := r.services.Forms.ListResponses(ctx, formID, pageSize, pageToken)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}

			result := map[string]interface{}{
				"responses": responses,
			}
			if nextPageToken != "" {
				result["nextPageToken"] = nextPageToken
			}
			return mcp.NewToolResultText(services.JSONResponse(result).Content[0].Text), nil
		},
	)

	// forms.getResponse - Get a specific response
	r.server.AddTool(
		mcp.NewTool("forms_getResponse",
			mcp.WithDescription("Gets a specific form response."),
			mcp.WithString("formId", mcp.Required(), mcp.Description("The ID of the form")),
			mcp.WithString("responseId", mcp.Required(), mcp.Description("The ID of the response")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			formID := args["formId"].(string)
			responseID := args["responseId"].(string)
			response, err := r.services.Forms.GetResponse(ctx, formID, responseID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(response).Content[0].Text), nil
		},
	)

	// forms.create - Create a new form
	r.server.AddTool(
		mcp.NewTool("forms_create",
			mcp.WithDescription("Creates a new Google Form."),
			mcp.WithString("title", mcp.Required(), mcp.Description("The title displayed in the form")),
			mcp.WithString("documentTitle", mcp.Description("The title of the form document (defaults to title)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			title := args["title"].(string)
			documentTitle := title
			if v, ok := args["documentTitle"].(string); ok && v != "" {
				documentTitle = v
			}
			form, err := r.services.Forms.CreateForm(ctx, title, documentTitle)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(form).Content[0].Text), nil
		},
	)

	// forms.updateInfo - Update form info
	r.server.AddTool(
		mcp.NewTool("forms_updateInfo",
			mcp.WithDescription("Updates a form's title and description."),
			mcp.WithString("formId", mcp.Required(), mcp.Description("The ID of the form")),
			mcp.WithString("title", mcp.Description("New title for the form")),
			mcp.WithString("description", mcp.Description("New description for the form")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			formID := args["formId"].(string)
			title := ""
			description := ""
			if v, ok := args["title"].(string); ok {
				title = v
			}
			if v, ok := args["description"].(string); ok {
				description = v
			}
			form, err := r.services.Forms.UpdateFormInfo(ctx, formID, title, description)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(form).Content[0].Text), nil
		},
	)
}
