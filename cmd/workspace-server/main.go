// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/oowada/google-workspace-mcp/internal/auth"
	"github.com/oowada/google-workspace-mcp/internal/services"
	"github.com/oowada/google-workspace-mcp/internal/tools"
	"github.com/oowada/google-workspace-mcp/internal/util"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

// OAuth scopes required for Google Workspace APIs
var scopes = []string{
	"https://www.googleapis.com/auth/documents",
	"https://www.googleapis.com/auth/drive",
	"https://www.googleapis.com/auth/calendar",
	"https://www.googleapis.com/auth/chat.spaces",
	"https://www.googleapis.com/auth/chat.messages",
	"https://www.googleapis.com/auth/chat.memberships",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/gmail.modify",
	"https://www.googleapis.com/auth/gmail.settings.basic",
	"https://www.googleapis.com/auth/directory.readonly",
	"https://www.googleapis.com/auth/presentations",
	"https://www.googleapis.com/auth/spreadsheets",
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("workspace-server %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	if *debug {
		util.EnableDebug()
	}

	// Initialize auth manager
	authManager := auth.NewAuthManager(scopes)

	// Initialize services
	driveService := services.NewDriveService(authManager)
	svc := &tools.ServiceContainer{
		Drive:    driveService,
		Docs:     services.NewDocsService(authManager, driveService),
		Calendar: services.NewCalendarService(authManager),
		Chat:     services.NewChatService(authManager),
		Gmail:    services.NewGmailService(authManager),
		People:   services.NewPeopleService(authManager),
		Slides:   services.NewSlidesService(authManager),
		Sheets:   services.NewSheetsService(authManager),
		Time:     services.NewTimeService(),
	}

	// Create MCP server
	srv := server.NewMCPServer(
		"google-workspace-server",
		Version,
		server.WithToolCapabilities(true),
	)

	// Register all tools
	registrar := tools.NewToolRegistrar(srv, authManager, svc)
	registrar.RegisterAll()

	// Run server with stdio transport
	util.LogInfo("Google Workspace MCP Server v%s starting...", Version)
	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
