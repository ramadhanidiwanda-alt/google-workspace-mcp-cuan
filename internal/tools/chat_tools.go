// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerChatTools() {
	// chat.listSpaces
	r.server.AddTool(
		mcp.NewTool("chat.listSpaces",
			mcp.WithDescription("Lists all Chat spaces the user is a member of."),
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
			resp := r.services.Chat.ListSpaces(ctx, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.findSpaceByName
	r.server.AddTool(
		mcp.NewTool("chat.findSpaceByName",
			mcp.WithDescription("Finds a Chat space by its display name."),
			mcp.WithString("displayName", mcp.Required(), mcp.Description("The display name of the space to find")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			displayName := args["displayName"].(string)
			resp := r.services.Chat.FindSpaceByName(ctx, displayName)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.sendMessage
	r.server.AddTool(
		mcp.NewTool("chat.sendMessage",
			mcp.WithDescription("Sends a message to a Chat space."),
			mcp.WithString("spaceName", mcp.Required(), mcp.Description("The resource name of the space (e.g., spaces/xxx)")),
			mcp.WithString("text", mcp.Required(), mcp.Description("The message text")),
			mcp.WithString("threadKey", mcp.Description("Thread key for replying to a thread")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spaceName := args["spaceName"].(string)
			text := args["text"].(string)
			var threadKey *string
			if v, ok := args["threadKey"].(string); ok && v != "" {
				threadKey = &v
			}
			resp := r.services.Chat.SendMessage(ctx, spaceName, text, threadKey)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.getMessages
	r.server.AddTool(
		mcp.NewTool("chat.getMessages",
			mcp.WithDescription("Gets messages from a Chat space."),
			mcp.WithString("spaceName", mcp.Required(), mcp.Description("The resource name of the space")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
			mcp.WithString("threadKey", mcp.Description("Thread key to filter by")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spaceName := args["spaceName"].(string)
			var pageToken, threadKey *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			if v, ok := args["threadKey"].(string); ok && v != "" {
				threadKey = &v
			}
			resp := r.services.Chat.GetMessages(ctx, spaceName, pageToken, pageSize, threadKey)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.sendDm
	r.server.AddTool(
		mcp.NewTool("chat.sendDm",
			mcp.WithDescription("Sends a direct message to a user by email."),
			mcp.WithString("email", mcp.Required(), mcp.Description("The email address of the recipient")),
			mcp.WithString("text", mcp.Required(), mcp.Description("The message text")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			email := args["email"].(string)
			text := args["text"].(string)
			resp := r.services.Chat.SendDm(ctx, email, text)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.findDmByEmail
	r.server.AddTool(
		mcp.NewTool("chat.findDmByEmail",
			mcp.WithDescription("Finds a DM space by user email."),
			mcp.WithString("email", mcp.Required(), mcp.Description("The email address of the user")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			email := args["email"].(string)
			resp := r.services.Chat.FindDmByEmail(ctx, email)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.listThreads
	r.server.AddTool(
		mcp.NewTool("chat.listThreads",
			mcp.WithDescription("Lists threads from a Chat space."),
			mcp.WithString("spaceName", mcp.Required(), mcp.Description("The resource name of the space")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spaceName := args["spaceName"].(string)
			var pageToken *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Chat.ListThreads(ctx, spaceName, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.setUpSpace
	r.server.AddTool(
		mcp.NewTool("chat.setUpSpace",
			mcp.WithDescription("Creates a new Chat space with members."),
			mcp.WithString("displayName", mcp.Required(), mcp.Description("The display name for the space")),
			mcp.WithString("memberEmails", mcp.Required(), mcp.Description("Comma-separated email addresses of members to add")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			displayName := args["displayName"].(string)
			memberEmailsStr := args["memberEmails"].(string)
			memberEmails := strings.Split(memberEmailsStr, ",")
			for i := range memberEmails {
				memberEmails[i] = strings.TrimSpace(memberEmails[i])
			}
			resp := r.services.Chat.SetUpSpace(ctx, displayName, memberEmails)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.addReaction
	r.server.AddTool(
		mcp.NewTool("chat.addReaction",
			mcp.WithDescription("Adds a reaction (emoji) to a message."),
			mcp.WithString("messageName", mcp.Required(), mcp.Description("The resource name of the message (e.g., spaces/xxx/messages/yyy)")),
			mcp.WithString("emoji", mcp.Required(), mcp.Description("The emoji to add (Unicode character like 👍)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			messageName := args["messageName"].(string)
			emoji := args["emoji"].(string)
			resp := r.services.Chat.AddReaction(ctx, messageName, emoji)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.removeReaction
	r.server.AddTool(
		mcp.NewTool("chat.removeReaction",
			mcp.WithDescription("Removes a reaction from a message."),
			mcp.WithString("reactionName", mcp.Required(), mcp.Description("The resource name of the reaction to remove")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			reactionName := args["reactionName"].(string)
			resp := r.services.Chat.RemoveReaction(ctx, reactionName)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.listReactions
	r.server.AddTool(
		mcp.NewTool("chat.listReactions",
			mcp.WithDescription("Lists reactions on a message."),
			mcp.WithString("messageName", mcp.Required(), mcp.Description("The resource name of the message")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			messageName := args["messageName"].(string)
			var pageToken *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Chat.ListReactions(ctx, messageName, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.sendCardMessage
	r.server.AddTool(
		mcp.NewTool("chat.sendCardMessage",
			mcp.WithDescription("Sends a simple card message to a space."),
			mcp.WithString("spaceName", mcp.Required(), mcp.Description("The resource name of the space")),
			mcp.WithString("content", mcp.Required(), mcp.Description("The text content for the card")),
			mcp.WithString("threadKey", mcp.Description("Thread key for replying to a thread")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spaceName := args["spaceName"].(string)
			content := args["content"].(string)
			var threadKey *string
			if v, ok := args["threadKey"].(string); ok && v != "" {
				threadKey = &v
			}
			resp := r.services.Chat.SendCardMessage(ctx, spaceName, content, threadKey)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// chat.sendRichCard
	r.server.AddTool(
		mcp.NewTool("chat.sendRichCard",
			mcp.WithDescription("Sends a rich card message with header, text, and optional button."),
			mcp.WithString("spaceName", mcp.Required(), mcp.Description("The resource name of the space")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Card title")),
			mcp.WithString("subtitle", mcp.Description("Card subtitle")),
			mcp.WithString("imageUrl", mcp.Description("Header image URL")),
			mcp.WithString("text", mcp.Description("Card body text")),
			mcp.WithString("buttonText", mcp.Description("Button text")),
			mcp.WithString("buttonUrl", mcp.Description("Button URL")),
			mcp.WithString("threadKey", mcp.Description("Thread key for replying to a thread")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			spaceName := args["spaceName"].(string)
			title := args["title"].(string)
			subtitle := ""
			imageURL := ""
			text := ""
			if v, ok := args["subtitle"].(string); ok {
				subtitle = v
			}
			if v, ok := args["imageUrl"].(string); ok {
				imageURL = v
			}
			if v, ok := args["text"].(string); ok {
				text = v
			}
			var buttonText, buttonURL, threadKey *string
			if v, ok := args["buttonText"].(string); ok && v != "" {
				buttonText = &v
			}
			if v, ok := args["buttonUrl"].(string); ok && v != "" {
				buttonURL = &v
			}
			if v, ok := args["threadKey"].(string); ok && v != "" {
				threadKey = &v
			}
			resp := r.services.Chat.SendRichCard(ctx, spaceName, title, subtitle, imageURL, text, buttonText, buttonURL, threadKey)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
