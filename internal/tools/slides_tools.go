// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerSlidesTools() {
	// slides.getText
	r.server.AddTool(
		mcp.NewTool("slides.getText",
			mcp.WithDescription("Retrieves text content from all slides in a presentation."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			resp := r.services.Slides.GetText(ctx, presentationID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.find
	r.server.AddTool(
		mcp.NewTool("slides.find",
			mcp.WithDescription("Searches for presentations by title."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
			mcp.WithNumber("pageSize", mcp.Description("Number of results per page")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			query := args["query"].(string)
			var pageToken *string
			var pageSize *int
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["pageSize"].(float64); ok {
				ps := int(v)
				pageSize = &ps
			}
			resp := r.services.Slides.Find(ctx, query, pageToken, pageSize)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.getMetadata
	r.server.AddTool(
		mcp.NewTool("slides.getMetadata",
			mcp.WithDescription("Gets presentation metadata (slide count, dimensions, etc.)."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			resp := r.services.Slides.GetMetadata(ctx, presentationID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.create
	r.server.AddTool(
		mcp.NewTool("slides.create",
			mcp.WithDescription("Creates a new presentation."),
			mcp.WithString("title", mcp.Required(), mcp.Description("The title of the presentation")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			title := args["title"].(string)
			resp := r.services.Slides.Create(ctx, title)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.addSlide
	r.server.AddTool(
		mcp.NewTool("slides.addSlide",
			mcp.WithDescription("Adds a new slide to a presentation."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
			mcp.WithString("layout", mcp.Description("Layout type: BLANK, TITLE, TITLE_AND_BODY, etc.")),
			mcp.WithNumber("insertionIndex", mcp.Description("Position to insert the slide")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			var layout *string
			var insertionIndex *int
			if v, ok := args["layout"].(string); ok && v != "" {
				layout = &v
			}
			if v, ok := args["insertionIndex"].(float64); ok {
				idx := int(v)
				insertionIndex = &idx
			}
			resp := r.services.Slides.AddSlide(ctx, presentationID, layout, insertionIndex)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.deleteSlide
	r.server.AddTool(
		mcp.NewTool("slides.deleteSlide",
			mcp.WithDescription("Deletes a slide from a presentation."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
			mcp.WithString("slideId", mcp.Required(), mcp.Description("The ID of the slide to delete")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			slideID := args["slideId"].(string)
			resp := r.services.Slides.DeleteSlide(ctx, presentationID, slideID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.addTextBox
	r.server.AddTool(
		mcp.NewTool("slides.addTextBox",
			mcp.WithDescription("Adds a text box to a slide."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
			mcp.WithString("slideId", mcp.Required(), mcp.Description("The ID of the slide")),
			mcp.WithString("text", mcp.Required(), mcp.Description("The text content")),
			mcp.WithNumber("x", mcp.Required(), mcp.Description("X position in points")),
			mcp.WithNumber("y", mcp.Required(), mcp.Description("Y position in points")),
			mcp.WithNumber("width", mcp.Required(), mcp.Description("Width in points")),
			mcp.WithNumber("height", mcp.Required(), mcp.Description("Height in points")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			slideID := args["slideId"].(string)
			text := args["text"].(string)
			x := args["x"].(float64)
			y := args["y"].(float64)
			width := args["width"].(float64)
			height := args["height"].(float64)
			resp := r.services.Slides.AddTextBox(ctx, presentationID, slideID, text, x, y, width, height)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.addImage
	r.server.AddTool(
		mcp.NewTool("slides.addImage",
			mcp.WithDescription("Adds an image to a slide from a URL."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
			mcp.WithString("slideId", mcp.Required(), mcp.Description("The ID of the slide")),
			mcp.WithString("imageUrl", mcp.Required(), mcp.Description("URL of the image")),
			mcp.WithNumber("x", mcp.Required(), mcp.Description("X position in points")),
			mcp.WithNumber("y", mcp.Required(), mcp.Description("Y position in points")),
			mcp.WithNumber("width", mcp.Required(), mcp.Description("Width in points")),
			mcp.WithNumber("height", mcp.Required(), mcp.Description("Height in points")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			slideID := args["slideId"].(string)
			imageURL := args["imageUrl"].(string)
			x := args["x"].(float64)
			y := args["y"].(float64)
			width := args["width"].(float64)
			height := args["height"].(float64)
			resp := r.services.Slides.AddImage(ctx, presentationID, slideID, imageURL, x, y, width, height)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// slides.updateText
	r.server.AddTool(
		mcp.NewTool("slides.updateText",
			mcp.WithDescription("Updates text in a shape."),
			mcp.WithString("presentationId", mcp.Required(), mcp.Description("The ID of the presentation")),
			mcp.WithString("shapeId", mcp.Required(), mcp.Description("The ID of the shape")),
			mcp.WithString("text", mcp.Required(), mcp.Description("The new text content")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			presentationID := args["presentationId"].(string)
			shapeID := args["shapeId"].(string)
			text := args["text"].(string)
			resp := r.services.Slides.UpdateText(ctx, presentationID, shapeID, text)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
