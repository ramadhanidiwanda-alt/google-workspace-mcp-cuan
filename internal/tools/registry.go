// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/oowada/google-workspace-mcp/internal/auth"
	"github.com/oowada/google-workspace-mcp/internal/services"
)

// ServiceContainer holds all service instances
type ServiceContainer struct {
	Docs     *services.DocsService
	Drive    *services.DriveService
	Calendar *services.CalendarService
	Chat     *services.ChatService
	Gmail    *services.GmailService
	People   *services.PeopleService
	Slides   *services.SlidesService
	Sheets   *services.SheetsService
	Time     *services.TimeService
	Tasks    *services.TasksService
	Forms    *services.FormsService
}

// ToolRegistrar registers all tools with the MCP server
type ToolRegistrar struct {
	server   *server.MCPServer
	auth     *auth.AuthManager
	services *ServiceContainer
}

// NewToolRegistrar creates a new ToolRegistrar instance
func NewToolRegistrar(srv *server.MCPServer, auth *auth.AuthManager, svc *ServiceContainer) *ToolRegistrar {
	return &ToolRegistrar{
		server:   srv,
		auth:     auth,
		services: svc,
	}
}

// RegisterAll registers all tools with the MCP server
func (r *ToolRegistrar) RegisterAll() {
	r.registerAuthTools()
	r.registerTimeTools()
	r.registerDriveTools()
	r.registerDocsTools()
	r.registerCalendarTools()
	r.registerGmailTools()
	r.registerChatTools()
	r.registerPeopleTools()
	r.registerSlidesTools()
	r.registerSheetsTools()
	r.registerTasksTools()
	r.registerFormsTools()
}
