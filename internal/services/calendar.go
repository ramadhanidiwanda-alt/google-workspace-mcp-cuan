// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"
)

// CalendarService provides Google Calendar operations
type CalendarService struct {
	auth AuthProvider
}

// NewCalendarService creates a new CalendarService instance
func NewCalendarService(auth AuthProvider) *CalendarService {
	return &CalendarService{auth: auth}
}

// getCalendarClient returns an authenticated Calendar client
func (s *CalendarService) getCalendarClient(ctx context.Context) (*calendar.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return calendar.NewService(ctx, opt)
}

// List lists all calendars
func (s *CalendarService) List(ctx context.Context) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	result, err := client.CalendarList.List().Do()
	if err != nil {
		return ErrorResponse(err)
	}

	calendars := make([]map[string]interface{}, len(result.Items))
	for i, cal := range result.Items {
		calendars[i] = map[string]interface{}{
			"id":          cal.Id,
			"summary":     cal.Summary,
			"description": cal.Description,
			"primary":     cal.Primary,
		}
	}

	return JSONResponse(map[string]interface{}{
		"calendars": calendars,
	})
}

// CreateEventInput contains input for CreateEvent
type CreateEventInput struct {
	CalendarID  *string  `json:"calendarId,omitempty" jsonschema:"The calendar ID (defaults to primary)"`
	Summary     string   `json:"summary" jsonschema:"The event title"`
	Description *string  `json:"description,omitempty" jsonschema:"The event description"`
	Start       string   `json:"start" jsonschema:"Start time in ISO8601 datetime format"`
	End         string   `json:"end" jsonschema:"End time in ISO8601 datetime format"`
	Attendees   []string `json:"attendees,omitempty" jsonschema:"Email addresses of attendees"`
	AddMeet     bool     `json:"addMeet,omitempty" jsonschema:"Add Google Meet video conference"`
	Location    *string  `json:"location,omitempty" jsonschema:"Event location"`
	Recurrence  []string `json:"recurrence,omitempty" jsonschema:"Recurrence rules (RRULE format)"`
}

// CreateEvent creates a new calendar event
func (s *CalendarService) CreateEvent(ctx context.Context, input CreateEventInput) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	calendarID := "primary"
	if input.CalendarID != nil && *input.CalendarID != "" {
		calendarID = *input.CalendarID
	}

	// Get user's timezone from calendar settings
	calSettings, err := client.CalendarList.Get(calendarID).Do()
	timeZone := "UTC"
	if err == nil && calSettings.TimeZone != "" {
		timeZone = calSettings.TimeZone
	}

	event := &calendar.Event{
		Summary: input.Summary,
		Start: &calendar.EventDateTime{
			DateTime: input.Start,
			TimeZone: timeZone,
		},
		End: &calendar.EventDateTime{
			DateTime: input.End,
			TimeZone: timeZone,
		},
	}

	if input.Description != nil {
		event.Description = *input.Description
	}

	if input.Location != nil {
		event.Location = *input.Location
	}

	if len(input.Attendees) > 0 {
		attendees := make([]*calendar.EventAttendee, len(input.Attendees))
		for i, email := range input.Attendees {
			attendees[i] = &calendar.EventAttendee{Email: email}
		}
		event.Attendees = attendees
	}

	// Add recurrence rules if specified
	if len(input.Recurrence) > 0 {
		event.Recurrence = input.Recurrence
	}

	// Add Google Meet conference if requested
	if input.AddMeet {
		event.ConferenceData = &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId: fmt.Sprintf("meet-%d", time.Now().UnixNano()),
				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{
					Type: "hangoutsMeet",
				},
			},
		}
	}

	insertCall := client.Events.Insert(calendarID, event)
	if input.AddMeet {
		insertCall = insertCall.ConferenceDataVersion(1)
	}

	result, err := insertCall.Do()
	if err != nil {
		return ErrorResponse(err)
	}

	response := map[string]interface{}{
		"eventId":  result.Id,
		"summary":  result.Summary,
		"htmlLink": result.HtmlLink,
		"status":   result.Status,
	}

	// Include Meet link if available
	if result.ConferenceData != nil && len(result.ConferenceData.EntryPoints) > 0 {
		for _, ep := range result.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				response["meetLink"] = ep.Uri
				break
			}
		}
	}

	return JSONResponse(response)
}

// CreateMeetingWithMeet creates a calendar event with Google Meet link
func (s *CalendarService) CreateMeetingWithMeet(ctx context.Context, summary, start, end string, calendarID, description, attendees *string) ToolResponse {
	var attendeeList []string
	if attendees != nil && *attendees != "" {
		for _, email := range strings.Split(*attendees, ",") {
			attendeeList = append(attendeeList, strings.TrimSpace(email))
		}
	}
	return s.CreateEvent(ctx, CreateEventInput{
		CalendarID:  calendarID,
		Summary:     summary,
		Description: description,
		Start:       start,
		End:         end,
		Attendees:   attendeeList,
		AddMeet:     true,
	})
}

// ListEventsInput contains input for ListEvents
type ListEventsInput struct {
	CalendarID             *string  `json:"calendarId,omitempty" jsonschema:"The calendar ID (defaults to primary)"`
	TimeMin                *string  `json:"timeMin,omitempty" jsonschema:"Start of time range (ISO8601)"`
	TimeMax                *string  `json:"timeMax,omitempty" jsonschema:"End of time range (ISO8601)"`
	MaxResults             *int     `json:"maxResults,omitempty" jsonschema:"Maximum number of events to return"`
	AttendeeResponseStatus []string `json:"attendeeResponseStatus,omitempty" jsonschema:"Filter by attendee response status"`
}

// ListEvents lists calendar events
func (s *CalendarService) ListEvents(ctx context.Context, input ListEventsInput) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	calendarID := "primary"
	if input.CalendarID != nil && *input.CalendarID != "" {
		calendarID = *input.CalendarID
	}

	req := client.Events.List(calendarID).
		SingleEvents(true).
		OrderBy("startTime")

	// Default to next 30 days if not specified
	now := time.Now()
	if input.TimeMin != nil {
		req.TimeMin(*input.TimeMin)
	} else {
		req.TimeMin(now.Format(time.RFC3339))
	}

	if input.TimeMax != nil {
		req.TimeMax(*input.TimeMax)
	} else {
		req.TimeMax(now.AddDate(0, 0, 30).Format(time.RFC3339))
	}

	if input.MaxResults != nil {
		req.MaxResults(int64(*input.MaxResults))
	} else {
		req.MaxResults(50)
	}

	result, err := req.Do()
	if err != nil {
		return ErrorResponse(err)
	}

	events := make([]map[string]interface{}, 0, len(result.Items))
	for _, event := range result.Items {
		if event.Status == "cancelled" {
			continue
		}

		eventData := map[string]interface{}{
			"id":          event.Id,
			"summary":     event.Summary,
			"description": event.Description,
			"status":      event.Status,
			"htmlLink":    event.HtmlLink,
		}

		if event.Start != nil {
			if event.Start.DateTime != "" {
				eventData["start"] = event.Start.DateTime
			} else {
				eventData["start"] = event.Start.Date
			}
		}

		if event.End != nil {
			if event.End.DateTime != "" {
				eventData["end"] = event.End.DateTime
			} else {
				eventData["end"] = event.End.Date
			}
		}

		if len(event.Attendees) > 0 {
			attendees := make([]map[string]string, len(event.Attendees))
			for i, a := range event.Attendees {
				attendees[i] = map[string]string{
					"email":          a.Email,
					"responseStatus": a.ResponseStatus,
				}
			}
			eventData["attendees"] = attendees
		}

		events = append(events, eventData)
	}

	return JSONResponse(map[string]interface{}{
		"events": events,
	})
}

// GetEvent gets a specific event
func (s *CalendarService) GetEvent(ctx context.Context, eventID string, calendarID *string) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	calID := "primary"
	if calendarID != nil && *calendarID != "" {
		calID = *calendarID
	}

	event, err := client.Events.Get(calID, eventID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"id":          event.Id,
		"summary":     event.Summary,
		"description": event.Description,
		"start":       event.Start,
		"end":         event.End,
		"status":      event.Status,
		"htmlLink":    event.HtmlLink,
		"attendees":   event.Attendees,
	})
}

// UpdateEvent updates an existing event
func (s *CalendarService) UpdateEvent(ctx context.Context, eventID string, calendarID *string, updates map[string]interface{}) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	calID := "primary"
	if calendarID != nil && *calendarID != "" {
		calID = *calendarID
	}

	// Get existing event
	event, err := client.Events.Get(calID, eventID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Apply updates
	if summary, ok := updates["summary"].(string); ok {
		event.Summary = summary
	}
	if description, ok := updates["description"].(string); ok {
		event.Description = description
	}
	if start, ok := updates["start"].(string); ok {
		event.Start = &calendar.EventDateTime{DateTime: start}
	}
	if end, ok := updates["end"].(string); ok {
		event.End = &calendar.EventDateTime{DateTime: end}
	}

	result, err := client.Events.Update(calID, eventID, event).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"status":  "success",
		"eventId": result.Id,
	})
}

// RespondToEvent responds to an event invitation
func (s *CalendarService) RespondToEvent(ctx context.Context, eventID string, calendarID *string, response string, message *string) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	calID := "primary"
	if calendarID != nil && *calendarID != "" {
		calID = *calendarID
	}

	// Get existing event
	event, err := client.Events.Get(calID, eventID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Find self in attendees and update response
	for _, attendee := range event.Attendees {
		if attendee.Self {
			attendee.ResponseStatus = response
			if message != nil {
				attendee.Comment = *message
			}
			break
		}
	}

	result, err := client.Events.Update(calID, eventID, event).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"status":   "success",
		"eventId":  result.Id,
		"response": response,
	})
}

// DeleteEvent deletes an event
func (s *CalendarService) DeleteEvent(ctx context.Context, eventID string, calendarID *string) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	calID := "primary"
	if calendarID != nil && *calendarID != "" {
		calID = *calendarID
	}

	err = client.Events.Delete(calID, eventID).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"status":  "success",
		"eventId": eventID,
	})
}

// FindFreeTimeInput contains input for FindFreeTime
type FindFreeTimeInput struct {
	Attendees    []string `json:"attendees" jsonschema:"Email addresses of attendees to check"`
	TimeMin      string   `json:"timeMin" jsonschema:"Start of search range (ISO8601)"`
	TimeMax      string   `json:"timeMax" jsonschema:"End of search range (ISO8601)"`
	DurationMins int      `json:"durationMins" jsonschema:"Required meeting duration in minutes"`
}

// CreateEventSimple creates a new calendar event with simple parameters
func (s *CalendarService) CreateEventSimple(ctx context.Context, summary, start, end string, calendarID, description, location, attendees *string) ToolResponse {
	var attendeeList []string
	if attendees != nil && *attendees != "" {
		for _, email := range strings.Split(*attendees, ",") {
			attendeeList = append(attendeeList, strings.TrimSpace(email))
		}
	}
	return s.CreateEvent(ctx, CreateEventInput{
		CalendarID:  calendarID,
		Summary:     summary,
		Description: description,
		Start:       start,
		End:         end,
		Attendees:   attendeeList,
	})
}

// ListEventsSimple lists calendar events with simple parameters
func (s *CalendarService) ListEventsSimple(ctx context.Context, calendarID, timeMin, timeMax *string, maxResults *int, pageToken *string) ToolResponse {
	return s.ListEvents(ctx, ListEventsInput{
		CalendarID: calendarID,
		TimeMin:    timeMin,
		TimeMax:    timeMax,
		MaxResults: maxResults,
	})
}

// FindFreeTimeSimple finds a free time slot with simple parameters
func (s *CalendarService) FindFreeTimeSimple(ctx context.Context, attendees, timeMin, timeMax string, durationMinutes int) ToolResponse {
	var attendeeList []string
	for _, email := range strings.Split(attendees, ",") {
		attendeeList = append(attendeeList, strings.TrimSpace(email))
	}
	return s.FindFreeTime(ctx, FindFreeTimeInput{
		Attendees:    attendeeList,
		TimeMin:      timeMin,
		TimeMax:      timeMax,
		DurationMins: durationMinutes,
	})
}

// CreateRecurringEvent creates a recurring calendar event
func (s *CalendarService) CreateRecurringEvent(ctx context.Context, summary, start, end string, recurrence string, calendarID, description, location, attendees *string) ToolResponse {
	var attendeeList []string
	if attendees != nil && *attendees != "" {
		for _, email := range strings.Split(*attendees, ",") {
			attendeeList = append(attendeeList, strings.TrimSpace(email))
		}
	}

	// Parse recurrence rule - support simple format like "DAILY", "WEEKLY", "MONTHLY", "YEARLY"
	// or full RRULE format
	var recurrenceRules []string
	if recurrence != "" {
		if strings.HasPrefix(strings.ToUpper(recurrence), "RRULE:") {
			recurrenceRules = []string{recurrence}
		} else {
			// Simple format conversion
			upper := strings.ToUpper(recurrence)
			switch {
			case strings.HasPrefix(upper, "DAILY"):
				recurrenceRules = []string{"RRULE:FREQ=DAILY"}
			case strings.HasPrefix(upper, "WEEKLY"):
				recurrenceRules = []string{"RRULE:FREQ=WEEKLY"}
			case strings.HasPrefix(upper, "MONTHLY"):
				recurrenceRules = []string{"RRULE:FREQ=MONTHLY"}
			case strings.HasPrefix(upper, "YEARLY"):
				recurrenceRules = []string{"RRULE:FREQ=YEARLY"}
			case strings.HasPrefix(upper, "WEEKDAYS"):
				recurrenceRules = []string{"RRULE:FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"}
			default:
				recurrenceRules = []string{"RRULE:" + recurrence}
			}
		}
	}

	return s.CreateEvent(ctx, CreateEventInput{
		CalendarID:  calendarID,
		Summary:     summary,
		Description: description,
		Location:    location,
		Start:       start,
		End:         end,
		Attendees:   attendeeList,
		Recurrence:  recurrenceRules,
	})
}

// FindFreeTime finds a free time slot for attendees
func (s *CalendarService) FindFreeTime(ctx context.Context, input FindFreeTimeInput) ToolResponse {
	client, err := s.getCalendarClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	items := make([]*calendar.FreeBusyRequestItem, len(input.Attendees))
	for i, email := range input.Attendees {
		items[i] = &calendar.FreeBusyRequestItem{Id: email}
	}

	result, err := client.Freebusy.Query(&calendar.FreeBusyRequest{
		TimeMin: input.TimeMin,
		TimeMax: input.TimeMax,
		Items:   items,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	// Find first available slot
	timeMin, _ := time.Parse(time.RFC3339, input.TimeMin)
	timeMax, _ := time.Parse(time.RFC3339, input.TimeMax)
	duration := time.Duration(input.DurationMins) * time.Minute

	// Collect all busy periods
	var busyPeriods []struct{ Start, End time.Time }
	for _, cal := range result.Calendars {
		for _, busy := range cal.Busy {
			start, _ := time.Parse(time.RFC3339, busy.Start)
			end, _ := time.Parse(time.RFC3339, busy.End)
			busyPeriods = append(busyPeriods, struct{ Start, End time.Time }{start, end})
		}
	}

	// Find first free slot
	current := timeMin
	for current.Add(duration).Before(timeMax) || current.Add(duration).Equal(timeMax) {
		slotEnd := current.Add(duration)
		isFree := true

		for _, busy := range busyPeriods {
			if current.Before(busy.End) && slotEnd.After(busy.Start) {
				isFree = false
				current = busy.End
				break
			}
		}

		if isFree {
			return JSONResponse(map[string]interface{}{
				"found": true,
				"start": current.Format(time.RFC3339),
				"end":   slotEnd.Format(time.RFC3339),
			})
		}
	}

	return JSONResponse(map[string]interface{}{
		"found":   false,
		"message": fmt.Sprintf("No free slot found between %s and %s", input.TimeMin, input.TimeMax),
	})
}
