// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/chat/v1"
)

// ChatService provides Google Chat operations
type ChatService struct {
	auth AuthProvider
}

// NewChatService creates a new ChatService instance
func NewChatService(auth AuthProvider) *ChatService {
	return &ChatService{auth: auth}
}

// getChatClient returns an authenticated Chat client
func (s *ChatService) getChatClient(ctx context.Context) (*chat.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return chat.NewService(ctx, opt)
}

// ListSpaces lists all spaces the user is a member of
func (s *ChatService) ListSpaces(ctx context.Context, pageToken *string, pageSize *int) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.Spaces.List()

	if pageSize != nil {
		req.PageSize(int64(*pageSize))
	} else {
		req.PageSize(20)
	}

	if pageToken != nil && *pageToken != "" {
		req.PageToken(*pageToken)
	}

	result, err := req.Do()
	if err != nil {
		return ErrorResponse(err)
	}

	spaces := make([]map[string]interface{}, len(result.Spaces))
	for i, space := range result.Spaces {
		spaces[i] = map[string]interface{}{
			"name":        space.Name,
			"displayName": space.DisplayName,
			"type":        space.Type,
			"spaceType":   space.SpaceType,
		}
	}

	response := map[string]interface{}{
		"spaces": spaces,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// FindSpaceByName finds a space by its display name
func (s *ChatService) FindSpaceByName(ctx context.Context, displayName string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// List all spaces and filter
	var allSpaces []*chat.Space
	pageToken := ""

	for {
		req := client.Spaces.List().PageSize(100)
		if pageToken != "" {
			req.PageToken(pageToken)
		}

		result, err := req.Do()
		if err != nil {
			return ErrorResponse(err)
		}

		allSpaces = append(allSpaces, result.Spaces...)

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	// Find matching space
	for _, space := range allSpaces {
		if strings.EqualFold(space.DisplayName, displayName) {
			return JSONResponse(map[string]interface{}{
				"found":       true,
				"name":        space.Name,
				"displayName": space.DisplayName,
				"type":        space.Type,
			})
		}
	}

	return JSONResponse(map[string]interface{}{
		"found": false,
	})
}

// SendMessage sends a message to a space
func (s *ChatService) SendMessage(ctx context.Context, spaceName string, text string, threadKey *string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	message := &chat.Message{
		Text: text,
	}

	if threadKey != nil && *threadKey != "" {
		message.Thread = &chat.Thread{
			ThreadKey: *threadKey,
		}
	}

	result, err := client.Spaces.Messages.Create(spaceName, message).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"name":       result.Name,
		"createTime": result.CreateTime,
	})
}

// GetMessages gets messages from a space
func (s *ChatService) GetMessages(ctx context.Context, spaceName string, pageToken *string, pageSize *int, threadKey *string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.Spaces.Messages.List(spaceName)

	if pageSize != nil {
		req.PageSize(int64(*pageSize))
	} else {
		req.PageSize(20)
	}

	if pageToken != nil && *pageToken != "" {
		req.PageToken(*pageToken)
	}

	result, err := req.Do()
	if err != nil {
		return ErrorResponse(err)
	}

	messages := make([]map[string]interface{}, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = map[string]interface{}{
			"name":       msg.Name,
			"text":       msg.Text,
			"createTime": msg.CreateTime,
			"sender":     msg.Sender,
		}
		if msg.Thread != nil {
			messages[i]["threadKey"] = msg.Thread.ThreadKey
		}
	}

	response := map[string]interface{}{
		"messages": messages,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// SendDm sends a direct message to a user by email
func (s *ChatService) SendDm(ctx context.Context, email string, text string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Find or create DM space
	space, err := client.Spaces.FindDirectMessage().Name(fmt.Sprintf("users/%s", email)).Do()
	if err != nil {
		// Space doesn't exist, create it
		space, err = client.Spaces.Setup(&chat.SetUpSpaceRequest{
			Space: &chat.Space{
				SpaceType:   "DIRECT_MESSAGE",
				SingleUserBotDm: false,
			},
			Memberships: []*chat.Membership{
				{
					Member: &chat.User{
						Name: fmt.Sprintf("users/%s", email),
						Type: "HUMAN",
					},
				},
			},
		}).Do()
		if err != nil {
			return ErrorResponse(err)
		}
	}

	// Send message
	message, err := client.Spaces.Messages.Create(space.Name, &chat.Message{
		Text: text,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"spaceName":  space.Name,
		"messageName": message.Name,
	})
}

// FindDmByEmail finds a DM space by user email
func (s *ChatService) FindDmByEmail(ctx context.Context, email string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	space, err := client.Spaces.FindDirectMessage().Name(fmt.Sprintf("users/%s", email)).Do()
	if err != nil {
		return JSONResponse(map[string]interface{}{
			"found": false,
		})
	}

	return JSONResponse(map[string]interface{}{
		"found":       true,
		"name":        space.Name,
		"displayName": space.DisplayName,
	})
}

// ListThreads lists threads from a space
func (s *ChatService) ListThreads(ctx context.Context, spaceName string, pageToken *string, pageSize *int) ToolResponse {
	// Note: Chat API doesn't have a direct ListThreads endpoint
	// We list messages and group by thread
	return s.GetMessages(ctx, spaceName, pageToken, pageSize, nil)
}

// SetUpSpace creates a new space
func (s *ChatService) SetUpSpace(ctx context.Context, displayName string, memberEmails []string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	memberships := make([]*chat.Membership, len(memberEmails))
	for i, email := range memberEmails {
		memberships[i] = &chat.Membership{
			Member: &chat.User{
				Name: fmt.Sprintf("users/%s", email),
				Type: "HUMAN",
			},
		}
	}

	result, err := client.Spaces.Setup(&chat.SetUpSpaceRequest{
		Space: &chat.Space{
			DisplayName: displayName,
			SpaceType:   "SPACE",
		},
		Memberships: memberships,
	}).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"name":        result.Name,
		"displayName": result.DisplayName,
	})
}

// AddReaction adds a reaction (emoji) to a message
func (s *ChatService) AddReaction(ctx context.Context, messageName string, emoji string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Extract parent (space name and message name)
	reaction := &chat.Reaction{
		Emoji: &chat.Emoji{
			Unicode: emoji,
		},
	}

	result, err := client.Spaces.Messages.Reactions.Create(messageName, reaction).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"name":  result.Name,
		"emoji": emoji,
	})
}

// RemoveReaction removes a reaction from a message
func (s *ChatService) RemoveReaction(ctx context.Context, reactionName string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	_, err = client.Spaces.Messages.Reactions.Delete(reactionName).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]interface{}{
		"status": "deleted",
	})
}

// ListReactions lists reactions on a message
func (s *ChatService) ListReactions(ctx context.Context, messageName string, pageToken *string, pageSize *int) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.Spaces.Messages.Reactions.List(messageName)

	if pageSize != nil {
		req.PageSize(int64(*pageSize))
	} else {
		req.PageSize(25)
	}

	if pageToken != nil && *pageToken != "" {
		req.PageToken(*pageToken)
	}

	result, err := req.Do()
	if err != nil {
		return ErrorResponse(err)
	}

	reactions := make([]map[string]interface{}, len(result.Reactions))
	for i, r := range result.Reactions {
		reactions[i] = map[string]interface{}{
			"name": r.Name,
		}
		if r.Emoji != nil {
			reactions[i]["emoji"] = r.Emoji.Unicode
		}
		if r.User != nil {
			reactions[i]["user"] = r.User.Name
		}
	}

	response := map[string]interface{}{
		"reactions": reactions,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}

// SendCardMessage sends a card message to a space
func (s *ChatService) SendCardMessage(ctx context.Context, spaceName string, cardJSON string, threadKey *string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	message := &chat.Message{
		CardsV2: []*chat.CardWithId{
			{
				CardId: "card1",
				Card: &chat.GoogleAppsCardV1Card{
					Header: &chat.GoogleAppsCardV1CardHeader{
						Title: "Card Message",
					},
					Sections: []*chat.GoogleAppsCardV1Section{
						{
							Widgets: []*chat.GoogleAppsCardV1Widget{
								{
									TextParagraph: &chat.GoogleAppsCardV1TextParagraph{
										Text: cardJSON,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if threadKey != nil && *threadKey != "" {
		message.Thread = &chat.Thread{
			ThreadKey: *threadKey,
		}
	}

	result, err := client.Spaces.Messages.Create(spaceName, message).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"name":       result.Name,
		"createTime": result.CreateTime,
	})
}

// SendRichCard sends a rich card message with header, sections, and buttons
func (s *ChatService) SendRichCard(ctx context.Context, spaceName string, title, subtitle, imageURL, text string, buttonText, buttonURL *string, threadKey *string) ToolResponse {
	client, err := s.getChatClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	card := &chat.GoogleAppsCardV1Card{
		Header: &chat.GoogleAppsCardV1CardHeader{
			Title:    title,
			Subtitle: subtitle,
		},
		Sections: []*chat.GoogleAppsCardV1Section{},
	}

	// Add image if provided
	if imageURL != "" {
		card.Header.ImageUrl = imageURL
		card.Header.ImageType = "CIRCLE"
	}

	// Add text section
	if text != "" {
		card.Sections = append(card.Sections, &chat.GoogleAppsCardV1Section{
			Widgets: []*chat.GoogleAppsCardV1Widget{
				{
					TextParagraph: &chat.GoogleAppsCardV1TextParagraph{
						Text: text,
					},
				},
			},
		})
	}

	// Add button if provided
	if buttonText != nil && *buttonText != "" && buttonURL != nil && *buttonURL != "" {
		card.Sections = append(card.Sections, &chat.GoogleAppsCardV1Section{
			Widgets: []*chat.GoogleAppsCardV1Widget{
				{
					ButtonList: &chat.GoogleAppsCardV1ButtonList{
						Buttons: []*chat.GoogleAppsCardV1Button{
							{
								Text: *buttonText,
								OnClick: &chat.GoogleAppsCardV1OnClick{
									OpenLink: &chat.GoogleAppsCardV1OpenLink{
										Url: *buttonURL,
									},
								},
							},
						},
					},
				},
			},
		})
	}

	message := &chat.Message{
		CardsV2: []*chat.CardWithId{
			{
				CardId: "richCard",
				Card:   card,
			},
		},
	}

	if threadKey != nil && *threadKey != "" {
		message.Thread = &chat.Thread{
			ThreadKey: *threadKey,
		}
	}

	result, err := client.Spaces.Messages.Create(spaceName, message).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return JSONResponse(map[string]string{
		"name":       result.Name,
		"createTime": result.CreateTime,
	})
}
