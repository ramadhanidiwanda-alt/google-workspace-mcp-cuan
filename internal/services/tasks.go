// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"time"

	"google.golang.org/api/tasks/v1"
)

// TasksService provides Google Tasks operations
type TasksService struct {
	auth AuthProvider
}

// NewTasksService creates a new TasksService
func NewTasksService(auth AuthProvider) *TasksService {
	return &TasksService{auth: auth}
}

// getClient creates a new Tasks API client
func (s *TasksService) getClient(ctx context.Context) (*tasks.Service, error) {
	opt, err := s.auth.GetClientOption(ctx)
	if err != nil {
		return nil, err
	}
	return tasks.NewService(ctx, opt)
}

// TaskListInfo represents a task list
type TaskListInfo struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Updated string `json:"updated,omitempty"`
}

// TaskInfo represents a task
type TaskInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Notes     string `json:"notes,omitempty"`
	Status    string `json:"status"` // needsAction or completed
	Due       string `json:"due,omitempty"`
	Completed string `json:"completed,omitempty"`
	Parent    string `json:"parent,omitempty"`
	Position  string `json:"position,omitempty"`
	Updated   string `json:"updated,omitempty"`
}

// ListTaskLists returns all task lists
func (s *TasksService) ListTaskLists(ctx context.Context) ([]TaskListInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := client.Tasklists.List().MaxResults(100).Do()
	if err != nil {
		return nil, err
	}

	var lists []TaskListInfo
	for _, tl := range resp.Items {
		lists = append(lists, TaskListInfo{
			ID:      tl.Id,
			Title:   tl.Title,
			Updated: tl.Updated,
		})
	}
	return lists, nil
}

// GetTaskList returns a specific task list
func (s *TasksService) GetTaskList(ctx context.Context, taskListID string) (*TaskListInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	tl, err := client.Tasklists.Get(taskListID).Do()
	if err != nil {
		return nil, err
	}

	return &TaskListInfo{
		ID:      tl.Id,
		Title:   tl.Title,
		Updated: tl.Updated,
	}, nil
}

// CreateTaskList creates a new task list
func (s *TasksService) CreateTaskList(ctx context.Context, title string) (*TaskListInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	tl, err := client.Tasklists.Insert(&tasks.TaskList{
		Title: title,
	}).Do()
	if err != nil {
		return nil, err
	}

	return &TaskListInfo{
		ID:      tl.Id,
		Title:   tl.Title,
		Updated: tl.Updated,
	}, nil
}

// UpdateTaskList updates a task list title
func (s *TasksService) UpdateTaskList(ctx context.Context, taskListID, title string) (*TaskListInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	tl, err := client.Tasklists.Update(taskListID, &tasks.TaskList{
		Title: title,
	}).Do()
	if err != nil {
		return nil, err
	}

	return &TaskListInfo{
		ID:      tl.Id,
		Title:   tl.Title,
		Updated: tl.Updated,
	}, nil
}

// DeleteTaskList deletes a task list
func (s *TasksService) DeleteTaskList(ctx context.Context, taskListID string) error {
	client, err := s.getClient(ctx)
	if err != nil {
		return err
	}

	return client.Tasklists.Delete(taskListID).Do()
}

// ListTasks returns tasks in a task list
func (s *TasksService) ListTasks(ctx context.Context, taskListID string, showCompleted, showHidden bool) ([]TaskInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	call := client.Tasks.List(taskListID).MaxResults(100)
	if showCompleted {
		call = call.ShowCompleted(true)
	}
	if showHidden {
		call = call.ShowHidden(true)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, err
	}

	var taskList []TaskInfo
	for _, t := range resp.Items {
		taskList = append(taskList, taskToInfo(t))
	}
	return taskList, nil
}

// taskToInfo converts a tasks.Task to TaskInfo
func taskToInfo(t *tasks.Task) TaskInfo {
	info := TaskInfo{
		ID:       t.Id,
		Title:    t.Title,
		Notes:    t.Notes,
		Status:   t.Status,
		Due:      t.Due,
		Parent:   t.Parent,
		Position: t.Position,
		Updated:  t.Updated,
	}
	if t.Completed != nil {
		info.Completed = *t.Completed
	}
	return info
}

// GetTask returns a specific task
func (s *TasksService) GetTask(ctx context.Context, taskListID, taskID string) (*TaskInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	t, err := client.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, err
	}

	info := taskToInfo(t)
	return &info, nil
}

// CreateTask creates a new task
func (s *TasksService) CreateTask(ctx context.Context, taskListID, title, notes, due, parent string) (*TaskInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	task := &tasks.Task{
		Title: title,
		Notes: notes,
	}
	if due != "" {
		task.Due = due
	}

	call := client.Tasks.Insert(taskListID, task)
	if parent != "" {
		call = call.Parent(parent)
	}

	t, err := call.Do()
	if err != nil {
		return nil, err
	}

	info := taskToInfo(t)
	return &info, nil
}

// UpdateTask updates a task
func (s *TasksService) UpdateTask(ctx context.Context, taskListID, taskID, title, notes, due, status string) (*TaskInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	// First get the existing task
	existing, err := client.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, err
	}

	// Update fields
	if title != "" {
		existing.Title = title
	}
	if notes != "" {
		existing.Notes = notes
	}
	if due != "" {
		existing.Due = due
	}
	if status != "" {
		existing.Status = status
		if status == "completed" {
			completedTime := time.Now().Format(time.RFC3339)
			existing.Completed = &completedTime
		} else {
			existing.Completed = nil
		}
	}

	t, err := client.Tasks.Update(taskListID, taskID, existing).Do()
	if err != nil {
		return nil, err
	}

	info := taskToInfo(t)
	return &info, nil
}

// CompleteTask marks a task as completed
func (s *TasksService) CompleteTask(ctx context.Context, taskListID, taskID string) (*TaskInfo, error) {
	return s.UpdateTask(ctx, taskListID, taskID, "", "", "", "completed")
}

// DeleteTask deletes a task
func (s *TasksService) DeleteTask(ctx context.Context, taskListID, taskID string) error {
	client, err := s.getClient(ctx)
	if err != nil {
		return err
	}

	return client.Tasks.Delete(taskListID, taskID).Do()
}

// ClearCompletedTasks clears all completed tasks from a list
func (s *TasksService) ClearCompletedTasks(ctx context.Context, taskListID string) error {
	client, err := s.getClient(ctx)
	if err != nil {
		return err
	}

	return client.Tasks.Clear(taskListID).Do()
}

// MoveTask moves a task to a different position
func (s *TasksService) MoveTask(ctx context.Context, taskListID, taskID, parent, previous string) (*TaskInfo, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	call := client.Tasks.Move(taskListID, taskID)
	if parent != "" {
		call = call.Parent(parent)
	}
	if previous != "" {
		call = call.Previous(previous)
	}

	t, err := call.Do()
	if err != nil {
		return nil, err
	}

	info := taskToInfo(t)
	return &info, nil
}
