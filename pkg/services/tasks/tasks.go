package tasks

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	tasksapi "google.golang.org/api/tasks/v1"
)

// Service wraps the Google Tasks API.
type Service struct {
	srv *tasksapi.Service
}

// New creates a new Service.
func New(ctx context.Context, opts ...option.ClientOption) (*Service, error) {
	srv, err := tasksapi.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Tasks client: %w", err)
	}
	return &Service{srv: srv}, nil
}

// ListTaskLists returns the authenticated user's task lists.
// Call this first so the AI can pick the correct task list ID for subsequent operations.
func (s *Service) ListTaskLists(maxResults int64) ([]*tasksapi.TaskList, error) {
	if maxResults <= 0 {
		maxResults = 100
	}
	resp, err := s.srv.Tasklists.List().MaxResults(maxResults).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list task lists: %w", err)
	}
	return resp.Items, nil
}

// ListTasksOptions configures how tasks are listed.
type ListTasksOptions struct {
	ShowCompleted bool  // Include completed tasks (default: false to reduce output)
	MaxResults    int64 // Max tasks per page (default: 20, max: 100)
}

// ListTasks returns tasks in the given task list.
// Use ShowCompleted: true to include completed tasks; false keeps output smaller.
func (s *Service) ListTasks(taskListID string, opts ListTasksOptions) ([]*tasksapi.Task, error) {
	if taskListID == "" {
		return nil, fmt.Errorf("task_list_id is required")
	}
	if opts.MaxResults <= 0 {
		opts.MaxResults = 20
	}
	if opts.MaxResults > 100 {
		opts.MaxResults = 100
	}

	call := s.srv.Tasks.List(taskListID).
		ShowCompleted(opts.ShowCompleted).
		MaxResults(opts.MaxResults)

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list tasks: %w", err)
	}
	return resp.Items, nil
}

// InsertTask creates a new task in the given task list.
func (s *Service) InsertTask(taskListID string, title string, notes string, due string) (*tasksapi.Task, error) {
	if taskListID == "" {
		return nil, fmt.Errorf("task_list_id is required")
	}
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	task := &tasksapi.Task{Title: title}
	if notes != "" {
		task.Notes = notes
	}
	if due != "" {
		task.Due = due
	}

	t, err := s.srv.Tasks.Insert(taskListID, task).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to insert task: %w", err)
	}
	return t, nil
}

// UpdateTaskInput holds optional fields for updating a task.
type UpdateTaskInput struct {
	Title  *string
	Notes  *string
	Due    *string
	Status *string // "needsAction" or "completed"
}

// UpdateTask updates an existing task. Only non-nil fields are applied.
func (s *Service) UpdateTask(taskListID string, taskID string, in UpdateTaskInput) (*tasksapi.Task, error) {
	if taskListID == "" || taskID == "" {
		return nil, fmt.Errorf("task_list_id and task_id are required")
	}

	existing, err := s.srv.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get task: %w", err)
	}

	if in.Title != nil {
		existing.Title = *in.Title
	}
	if in.Notes != nil {
		existing.Notes = *in.Notes
	}
	if in.Due != nil {
		existing.Due = *in.Due
	}
	if in.Status != nil {
		existing.Status = *in.Status
	}

	t, err := s.srv.Tasks.Update(taskListID, taskID, existing).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to update task: %w", err)
	}
	return t, nil
}

// DeleteTask removes a task from the task list.
func (s *Service) DeleteTask(taskListID string, taskID string) error {
	if taskListID == "" || taskID == "" {
		return fmt.Errorf("task_list_id and task_id are required")
	}
	return s.srv.Tasks.Delete(taskListID, taskID).Do()
}
