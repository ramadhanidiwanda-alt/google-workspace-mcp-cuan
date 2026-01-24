// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *ToolRegistrar) registerCalendarTools() {
	// calendar.list
	r.server.AddTool(
		mcp.NewTool("calendar.list",
			mcp.WithDescription("Lists all calendars for the user."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp := r.services.Calendar.List(ctx)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// calendar.createEvent
	r.server.AddTool(
		mcp.NewTool("calendar.createEvent",
			mcp.WithDescription("Creates a new calendar event."),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Event title")),
			mcp.WithString("start", mcp.Required(), mcp.Description("Start time in ISO8601 format")),
			mcp.WithString("end", mcp.Required(), mcp.Description("End time in ISO8601 format")),
			mcp.WithString("calendarId", mcp.Description("Calendar ID (defaults to primary)")),
			mcp.WithString("description", mcp.Description("Event description")),
			mcp.WithString("location", mcp.Description("Event location")),
			mcp.WithString("attendees", mcp.Description("Comma-separated list of attendee emails")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			summary := args["summary"].(string)
			start := args["start"].(string)
			end := args["end"].(string)
			var calendarID, description, location, attendees *string
			if v, ok := args["calendarId"].(string); ok && v != "" {
				calendarID = &v
			}
			if v, ok := args["description"].(string); ok && v != "" {
				description = &v
			}
			if v, ok := args["location"].(string); ok && v != "" {
				location = &v
			}
			if v, ok := args["attendees"].(string); ok && v != "" {
				attendees = &v
			}
			resp := r.services.Calendar.CreateEventSimple(ctx, summary, start, end, calendarID, description, location, attendees)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// calendar.listEvents
	r.server.AddTool(
		mcp.NewTool("calendar.listEvents",
			mcp.WithDescription("Lists calendar events within a time range."),
			mcp.WithString("calendarId", mcp.Description("Calendar ID (defaults to primary)")),
			mcp.WithString("timeMin", mcp.Description("Start of time range in ISO8601 format")),
			mcp.WithString("timeMax", mcp.Description("End of time range in ISO8601 format")),
			mcp.WithNumber("maxResults", mcp.Description("Maximum number of results")),
			mcp.WithString("pageToken", mcp.Description("Token for pagination")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			var calendarID, timeMin, timeMax, pageToken *string
			var maxResults *int
			if v, ok := args["calendarId"].(string); ok && v != "" {
				calendarID = &v
			}
			if v, ok := args["timeMin"].(string); ok && v != "" {
				timeMin = &v
			}
			if v, ok := args["timeMax"].(string); ok && v != "" {
				timeMax = &v
			}
			if v, ok := args["pageToken"].(string); ok && v != "" {
				pageToken = &v
			}
			if v, ok := args["maxResults"].(float64); ok {
				mr := int(v)
				maxResults = &mr
			}
			resp := r.services.Calendar.ListEventsSimple(ctx, calendarID, timeMin, timeMax, maxResults, pageToken)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// calendar.getEvent
	r.server.AddTool(
		mcp.NewTool("calendar.getEvent",
			mcp.WithDescription("Gets details of a specific calendar event."),
			mcp.WithString("eventId", mcp.Required(), mcp.Description("The ID of the event")),
			mcp.WithString("calendarId", mcp.Description("The calendar ID (defaults to primary)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			eventID := args["eventId"].(string)
			var calendarID *string
			if v, ok := args["calendarId"].(string); ok && v != "" {
				calendarID = &v
			}
			resp := r.services.Calendar.GetEvent(ctx, eventID, calendarID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// calendar.updateEvent
	r.server.AddTool(
		mcp.NewTool("calendar.updateEvent",
			mcp.WithDescription("Updates an existing calendar event."),
			mcp.WithString("eventId", mcp.Required(), mcp.Description("The ID of the event")),
			mcp.WithString("calendarId", mcp.Description("The calendar ID")),
			mcp.WithString("summary", mcp.Description("New event title")),
			mcp.WithString("description", mcp.Description("New description")),
			mcp.WithString("start", mcp.Description("New start time (ISO8601)")),
			mcp.WithString("end", mcp.Description("New end time (ISO8601)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			eventID := args["eventId"].(string)
			var calendarID *string
			if v, ok := args["calendarId"].(string); ok && v != "" {
				calendarID = &v
			}
			updates := make(map[string]interface{})
			if v, ok := args["summary"].(string); ok && v != "" {
				updates["summary"] = v
			}
			if v, ok := args["description"].(string); ok && v != "" {
				updates["description"] = v
			}
			if v, ok := args["start"].(string); ok && v != "" {
				updates["start"] = v
			}
			if v, ok := args["end"].(string); ok && v != "" {
				updates["end"] = v
			}
			resp := r.services.Calendar.UpdateEvent(ctx, eventID, calendarID, updates)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// calendar.respondToEvent
	r.server.AddTool(
		mcp.NewTool("calendar.respondToEvent",
			mcp.WithDescription("Responds to an event invitation (accept, decline, or tentative)."),
			mcp.WithString("eventId", mcp.Required(), mcp.Description("The ID of the event")),
			mcp.WithString("response", mcp.Required(), mcp.Description("Response: accepted, declined, or tentative")),
			mcp.WithString("calendarId", mcp.Description("The calendar ID")),
			mcp.WithString("message", mcp.Description("Optional response message")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			eventID := args["eventId"].(string)
			response := args["response"].(string)
			var calendarID, message *string
			if v, ok := args["calendarId"].(string); ok && v != "" {
				calendarID = &v
			}
			if v, ok := args["message"].(string); ok && v != "" {
				message = &v
			}
			resp := r.services.Calendar.RespondToEvent(ctx, eventID, calendarID, response, message)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// calendar.deleteEvent
	r.server.AddTool(
		mcp.NewTool("calendar.deleteEvent",
			mcp.WithDescription("Deletes a calendar event."),
			mcp.WithString("eventId", mcp.Required(), mcp.Description("The ID of the event to delete")),
			mcp.WithString("calendarId", mcp.Description("The calendar ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			eventID := args["eventId"].(string)
			var calendarID *string
			if v, ok := args["calendarId"].(string); ok && v != "" {
				calendarID = &v
			}
			resp := r.services.Calendar.DeleteEvent(ctx, eventID, calendarID)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)

	// calendar.findFreeTime
	r.server.AddTool(
		mcp.NewTool("calendar.findFreeTime",
			mcp.WithDescription("Finds a free time slot for multiple attendees."),
			mcp.WithString("attendees", mcp.Required(), mcp.Description("Comma-separated list of attendee emails")),
			mcp.WithString("timeMin", mcp.Required(), mcp.Description("Start of search window (ISO8601)")),
			mcp.WithString("timeMax", mcp.Required(), mcp.Description("End of search window (ISO8601)")),
			mcp.WithNumber("durationMinutes", mcp.Required(), mcp.Description("Required meeting duration in minutes")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			attendees := args["attendees"].(string)
			timeMin := args["timeMin"].(string)
			timeMax := args["timeMax"].(string)
			durationMinutes := int(args["durationMinutes"].(float64))
			resp := r.services.Calendar.FindFreeTimeSimple(ctx, attendees, timeMin, timeMax, durationMinutes)
			return mcp.NewToolResultText(resp.Content[0].Text), nil
		},
	)
}
