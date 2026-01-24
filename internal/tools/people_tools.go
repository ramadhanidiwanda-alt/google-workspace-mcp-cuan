// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerPeopleTools() {
	// people.getUserProfile
	r.server.AddTool(
		mcp.NewTool("people.getUserProfile",
			mcp.WithDescription("Gets a user's profile by ID, email, or name."),
			mcp.WithString("identifier", mcp.Required(), mcp.Description("User ID, email address, or name to search for")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			identifier := args["identifier"].(string)
			resp := r.services.People.GetUserProfile(ctx, identifier)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.getMe
	r.server.AddTool(
		mcp.NewTool("people.getMe",
			mcp.WithDescription("Gets the authenticated user's profile."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.People.GetMe(ctx)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.getUserRelations
	r.server.AddTool(
		mcp.NewTool("people.getUserRelations",
			mcp.WithDescription("Gets a user's relations (manager, spouse, assistant, etc.)."),
			mcp.WithString("identifier", mcp.Required(), mcp.Description("User ID, email, name, or 'me'")),
			mcp.WithString("relationType", mcp.Description("Filter by relation type (e.g., manager, spouse)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			identifier := args["identifier"].(string)
			var relationType *string
			if v, ok := args["relationType"].(string); ok && v != "" {
				relationType = &v
			}
			resp := r.services.People.GetUserRelations(ctx, identifier, relationType)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.listContacts
	r.server.AddTool(
		mcp.NewTool("people.listContacts",
			mcp.WithDescription("Lists the user's contacts."),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			var pageSize int64 = 0
			pageToken := ""
			if v, ok := args["pageSize"].(float64); ok {
				pageSize = int64(v)
			}
			if v, ok := args["pageToken"].(string); ok {
				pageToken = v
			}
			resp := r.services.People.ListContacts(ctx, pageSize, pageToken)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.createContact
	r.server.AddTool(
		mcp.NewTool("people.createContact",
			mcp.WithDescription("Creates a new contact."),
			mcp.WithString("givenName", mcp.Description("First name")),
			mcp.WithString("familyName", mcp.Description("Last name")),
			mcp.WithString("email", mcp.Description("Email address")),
			mcp.WithString("phone", mcp.Description("Phone number")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			givenName := ""
			familyName := ""
			email := ""
			phone := ""
			if v, ok := args["givenName"].(string); ok {
				givenName = v
			}
			if v, ok := args["familyName"].(string); ok {
				familyName = v
			}
			if v, ok := args["email"].(string); ok {
				email = v
			}
			if v, ok := args["phone"].(string); ok {
				phone = v
			}
			resp := r.services.People.CreateContact(ctx, givenName, familyName, email, phone)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.updateContact
	r.server.AddTool(
		mcp.NewTool("people.updateContact",
			mcp.WithDescription("Updates an existing contact."),
			mcp.WithString("resourceName", mcp.Required(), mcp.Description("The resource name of the contact (e.g., people/c12345)")),
			mcp.WithString("givenName", mcp.Description("New first name")),
			mcp.WithString("familyName", mcp.Description("New last name")),
			mcp.WithString("email", mcp.Description("New email address")),
			mcp.WithString("phone", mcp.Description("New phone number")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			resourceName := args["resourceName"].(string)
			givenName := ""
			familyName := ""
			email := ""
			phone := ""
			if v, ok := args["givenName"].(string); ok {
				givenName = v
			}
			if v, ok := args["familyName"].(string); ok {
				familyName = v
			}
			if v, ok := args["email"].(string); ok {
				email = v
			}
			if v, ok := args["phone"].(string); ok {
				phone = v
			}
			resp := r.services.People.UpdateContact(ctx, resourceName, givenName, familyName, email, phone)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.deleteContact
	r.server.AddTool(
		mcp.NewTool("people.deleteContact",
			mcp.WithDescription("Deletes a contact."),
			mcp.WithString("resourceName", mcp.Required(), mcp.Description("The resource name of the contact (e.g., people/c12345)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			resourceName := args["resourceName"].(string)
			resp := r.services.People.DeleteContact(ctx, resourceName)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// people.searchContacts
	r.server.AddTool(
		mcp.NewTool("people.searchContacts",
			mcp.WithDescription("Searches the user's contacts."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			query := args["query"].(string)
			var pageSize int64 = 0
			if v, ok := args["pageSize"].(float64); ok {
				pageSize = int64(v)
			}
			resp := r.services.People.SearchContacts(ctx, query, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
