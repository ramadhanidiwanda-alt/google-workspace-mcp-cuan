// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oowada/google-workspace-mcp/internal/services"
)

// registerTasksTools registers Google Tasks tools
func (r *ToolRegistrar) registerTasksTools() {
	// tasks.listTaskLists - List all task lists
	r.server.AddTool(
		mcp.NewTool("tasks_listTaskLists",
			mcp.WithDescription("Lists all task lists for the user."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			lists, err := r.services.Tasks.ListTaskLists(ctx)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(lists).Content[0].Text), nil
		},
	)

	// tasks.getTaskList - Get a specific task list
	r.server.AddTool(
		mcp.NewTool("tasks_getTaskList",
			mcp.WithDescription("Gets a specific task list by ID."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			list, err := r.services.Tasks.GetTaskList(ctx, taskListID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(list).Content[0].Text), nil
		},
	)

	// tasks.createTaskList - Create a new task list
	r.server.AddTool(
		mcp.NewTool("tasks_createTaskList",
			mcp.WithDescription("Creates a new task list."),
			mcp.WithString("title", mcp.Required(), mcp.Description("The title of the task list")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			title := args["title"].(string)
			list, err := r.services.Tasks.CreateTaskList(ctx, title)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(list).Content[0].Text), nil
		},
	)

	// tasks.updateTaskList - Update a task list
	r.server.AddTool(
		mcp.NewTool("tasks_updateTaskList",
			mcp.WithDescription("Updates a task list title."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
			mcp.WithString("title", mcp.Required(), mcp.Description("The new title")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			title := args["title"].(string)
			list, err := r.services.Tasks.UpdateTaskList(ctx, taskListID, title)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(list).Content[0].Text), nil
		},
	)

	// tasks.deleteTaskList - Delete a task list
	r.server.AddTool(
		mcp.NewTool("tasks_deleteTaskList",
			mcp.WithDescription("Deletes a task list."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			err := r.services.Tasks.DeleteTaskList(ctx, taskListID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(map[string]string{"status": "deleted"}).Content[0].Text), nil
		},
	)

	// tasks.listTasks - List tasks in a task list
	r.server.AddTool(
		mcp.NewTool("tasks_listTasks",
			mcp.WithDescription("Lists all tasks in a task list."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list (use '@default' for primary list)")),
			mcp.WithBoolean("showCompleted", mcp.Description("Include completed tasks (default: false)")),
			mcp.WithBoolean("showHidden", mcp.Description("Include hidden tasks (default: false)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			showCompleted := false
			showHidden := false
			if v, ok := args["showCompleted"].(bool); ok {
				showCompleted = v
			}
			if v, ok := args["showHidden"].(bool); ok {
				showHidden = v
			}
			tasks, err := r.services.Tasks.ListTasks(ctx, taskListID, showCompleted, showHidden)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(tasks).Content[0].Text), nil
		},
	)

	// tasks.getTask - Get a specific task
	r.server.AddTool(
		mcp.NewTool("tasks_getTask",
			mcp.WithDescription("Gets a specific task."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
			mcp.WithString("taskId", mcp.Required(), mcp.Description("The ID of the task")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			taskID := args["taskId"].(string)
			task, err := r.services.Tasks.GetTask(ctx, taskListID, taskID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(task).Content[0].Text), nil
		},
	)

	// tasks.createTask - Create a new task
	r.server.AddTool(
		mcp.NewTool("tasks_createTask",
			mcp.WithDescription("Creates a new task."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list (use '@default' for primary list)")),
			mcp.WithString("title", mcp.Required(), mcp.Description("The title of the task")),
			mcp.WithString("notes", mcp.Description("Notes/description for the task")),
			mcp.WithString("due", mcp.Description("Due date in RFC3339 format (e.g., 2024-12-31T00:00:00Z)")),
			mcp.WithString("parent", mcp.Description("Parent task ID for subtasks")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			title := args["title"].(string)
			notes := ""
			due := ""
			parent := ""
			if v, ok := args["notes"].(string); ok {
				notes = v
			}
			if v, ok := args["due"].(string); ok {
				due = v
			}
			if v, ok := args["parent"].(string); ok {
				parent = v
			}
			task, err := r.services.Tasks.CreateTask(ctx, taskListID, title, notes, due, parent)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(task).Content[0].Text), nil
		},
	)

	// tasks.updateTask - Update a task
	r.server.AddTool(
		mcp.NewTool("tasks_updateTask",
			mcp.WithDescription("Updates an existing task."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
			mcp.WithString("taskId", mcp.Required(), mcp.Description("The ID of the task")),
			mcp.WithString("title", mcp.Description("New title")),
			mcp.WithString("notes", mcp.Description("New notes")),
			mcp.WithString("due", mcp.Description("New due date in RFC3339 format")),
			mcp.WithString("status", mcp.Description("Status: needsAction or completed")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			taskID := args["taskId"].(string)
			title := ""
			notes := ""
			due := ""
			status := ""
			if v, ok := args["title"].(string); ok {
				title = v
			}
			if v, ok := args["notes"].(string); ok {
				notes = v
			}
			if v, ok := args["due"].(string); ok {
				due = v
			}
			if v, ok := args["status"].(string); ok {
				status = v
			}
			task, err := r.services.Tasks.UpdateTask(ctx, taskListID, taskID, title, notes, due, status)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(task).Content[0].Text), nil
		},
	)

	// tasks.completeTask - Mark a task as completed
	r.server.AddTool(
		mcp.NewTool("tasks_completeTask",
			mcp.WithDescription("Marks a task as completed."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
			mcp.WithString("taskId", mcp.Required(), mcp.Description("The ID of the task")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			taskID := args["taskId"].(string)
			task, err := r.services.Tasks.CompleteTask(ctx, taskListID, taskID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(task).Content[0].Text), nil
		},
	)

	// tasks.deleteTask - Delete a task
	r.server.AddTool(
		mcp.NewTool("tasks_deleteTask",
			mcp.WithDescription("Deletes a task."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
			mcp.WithString("taskId", mcp.Required(), mcp.Description("The ID of the task")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			taskID := args["taskId"].(string)
			err := r.services.Tasks.DeleteTask(ctx, taskListID, taskID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(map[string]string{"status": "deleted"}).Content[0].Text), nil
		},
	)

	// tasks.clearCompleted - Clear all completed tasks
	r.server.AddTool(
		mcp.NewTool("tasks_clearCompleted",
			mcp.WithDescription("Clears all completed tasks from a task list."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			err := r.services.Tasks.ClearCompletedTasks(ctx, taskListID)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(map[string]string{"status": "cleared"}).Content[0].Text), nil
		},
	)

	// tasks.moveTask - Move a task
	r.server.AddTool(
		mcp.NewTool("tasks_moveTask",
			mcp.WithDescription("Moves a task to a different position or makes it a subtask."),
			mcp.WithString("taskListId", mcp.Required(), mcp.Description("The ID of the task list")),
			mcp.WithString("taskId", mcp.Required(), mcp.Description("The ID of the task")),
			mcp.WithString("parent", mcp.Description("Parent task ID to make this a subtask")),
			mcp.WithString("previous", mcp.Description("Previous sibling task ID for positioning")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments.(map[string]interface{})
			taskListID := args["taskListId"].(string)
			taskID := args["taskId"].(string)
			parent := ""
			previous := ""
			if v, ok := args["parent"].(string); ok {
				parent = v
			}
			if v, ok := args["previous"].(string); ok {
				previous = v
			}
			task, err := r.services.Tasks.MoveTask(ctx, taskListID, taskID, parent, previous)
			if err != nil {
				return mcp.NewToolResultText(services.ErrorResponse(err).Content[0].Text), nil
			}
			return mcp.NewToolResultText(services.JSONResponse(task).Content[0].Text), nil
		},
	)
}
