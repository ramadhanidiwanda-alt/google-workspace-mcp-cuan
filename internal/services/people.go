// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"strings"

	"google.golang.org/api/people/v1"
)

// PeopleService provides Google People API operations
type PeopleService struct {
	auth AuthProvider
}

// NewPeopleService creates a new PeopleService instance
func NewPeopleService(auth AuthProvider) *PeopleService {
	return &PeopleService{auth: auth}
}

// getPeopleClient returns an authenticated People client
func (s *PeopleService) getPeopleClient(ctx context.Context) (*people.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return people.NewService(ctx, opt)
}

const personFields = "names,emailAddresses,phoneNumbers,organizations,photos,relations,userDefined"

// GetUserProfile gets a user's profile by ID, email, or name
func (s *PeopleService) GetUserProfile(ctx context.Context, identifier string) ToolResponse {
	client, err := s.getPeopleClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Determine if identifier is an email, resource name, or search term
	var resourceName string
	if strings.HasPrefix(identifier, "people/") {
		resourceName = identifier
	} else if strings.Contains(identifier, "@") {
		// Search by email
		result, err := client.People.SearchDirectoryPeople().
			Query(identifier).
			ReadMask(personFields).
			Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").
			PageSize(1).
			Do()
		if err != nil {
			return ErrorResponse(err)
		}
		if len(result.People) == 0 {
			return JSONResponse(map[string]interface{}{
				"found": false,
			})
		}
		resourceName = result.People[0].ResourceName
	} else {
		// Search by name
		result, err := client.People.SearchDirectoryPeople().
			Query(identifier).
			ReadMask(personFields).
			Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").
			PageSize(1).
			Do()
		if err != nil {
			return ErrorResponse(err)
		}
		if len(result.People) == 0 {
			return JSONResponse(map[string]interface{}{
				"found": false,
			})
		}
		resourceName = result.People[0].ResourceName
	}

	// Get full profile
	person, err := client.People.Get(resourceName).PersonFields(personFields).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return formatPerson(person)
}

// GetMe gets the authenticated user's profile
func (s *PeopleService) GetMe(ctx context.Context) ToolResponse {
	client, err := s.getPeopleClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	person, err := client.People.Get("people/me").PersonFields(personFields).Do()
	if err != nil {
		return ErrorResponse(err)
	}

	return formatPerson(person)
}

// GetUserRelations gets a user's relations (manager, spouse, assistant, etc.)
func (s *PeopleService) GetUserRelations(ctx context.Context, identifier string, relationType *string) ToolResponse {
	client, err := s.getPeopleClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	// Get person with relations
	var resourceName string
	if strings.HasPrefix(identifier, "people/") {
		resourceName = identifier
	} else if identifier == "me" {
		resourceName = "people/me"
	} else {
		// Search by email or name
		result, err := client.People.SearchDirectoryPeople().
			Query(identifier).
			ReadMask("relations").
			Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").
			PageSize(1).
			Do()
		if err != nil {
			return ErrorResponse(err)
		}
		if len(result.People) == 0 {
			return JSONResponse(map[string]interface{}{
				"found": false,
			})
		}
		resourceName = result.People[0].ResourceName
	}

	person, err := client.People.Get(resourceName).PersonFields("relations").Do()
	if err != nil {
		return ErrorResponse(err)
	}

	relations := make([]map[string]string, 0)
	for _, rel := range person.Relations {
		if relationType != nil && *relationType != "" {
			if !strings.EqualFold(rel.Type, *relationType) {
				continue
			}
		}
		relations = append(relations, map[string]string{
			"person": rel.Person,
			"type":   rel.Type,
		})
	}

	return JSONResponse(map[string]interface{}{
		"resourceName": person.ResourceName,
		"relations":    relations,
	})
}

// formatPerson formats a person resource for response
func formatPerson(person *people.Person) ToolResponse {
	result := map[string]interface{}{
		"found":        true,
		"resourceName": person.ResourceName,
	}

	// Names
	if len(person.Names) > 0 {
		name := person.Names[0]
		result["name"] = map[string]string{
			"displayName": name.DisplayName,
			"givenName":   name.GivenName,
			"familyName":  name.FamilyName,
		}
	}

	// Email addresses
	if len(person.EmailAddresses) > 0 {
		emails := make([]map[string]string, len(person.EmailAddresses))
		for i, email := range person.EmailAddresses {
			emails[i] = map[string]string{
				"value": email.Value,
				"type":  email.Type,
			}
		}
		result["emailAddresses"] = emails
	}

	// Phone numbers
	if len(person.PhoneNumbers) > 0 {
		phones := make([]map[string]string, len(person.PhoneNumbers))
		for i, phone := range person.PhoneNumbers {
			phones[i] = map[string]string{
				"value": phone.Value,
				"type":  phone.Type,
			}
		}
		result["phoneNumbers"] = phones
	}

	// Organizations
	if len(person.Organizations) > 0 {
		orgs := make([]map[string]string, len(person.Organizations))
		for i, org := range person.Organizations {
			orgs[i] = map[string]string{
				"name":       org.Name,
				"title":      org.Title,
				"department": org.Department,
			}
		}
		result["organizations"] = orgs
	}

	// Photos
	if len(person.Photos) > 0 {
		result["photoUrl"] = person.Photos[0].Url
	}

	// Relations
	if len(person.Relations) > 0 {
		relations := make([]map[string]string, len(person.Relations))
		for i, rel := range person.Relations {
			relations[i] = map[string]string{
				"person": rel.Person,
				"type":   rel.Type,
			}
		}
		result["relations"] = relations
	}

	return JSONResponse(result)
}

// SearchDirectory searches the directory for people
func (s *PeopleService) SearchDirectory(ctx context.Context, query string, pageToken *string, pageSize *int) ToolResponse {
	client, err := s.getPeopleClient(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	req := client.People.SearchDirectoryPeople().
		Query(query).
		ReadMask(personFields).
		Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE")

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

	people := make([]map[string]interface{}, len(result.People))
	for i, person := range result.People {
		personData := map[string]interface{}{
			"resourceName": person.ResourceName,
		}
		if len(person.Names) > 0 {
			personData["displayName"] = person.Names[0].DisplayName
		}
		if len(person.EmailAddresses) > 0 {
			personData["email"] = person.EmailAddresses[0].Value
		}
		people[i] = personData
	}

	response := map[string]interface{}{
		"people": people,
	}
	if result.NextPageToken != "" {
		response["nextPageToken"] = result.NextPageToken
	}

	return JSONResponse(response)
}
