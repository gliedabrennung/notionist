package notion

import (
	"context"
	"fmt"
	"time"
)

type Task struct {
	Name        string
	ProjectName string
	Priority    string
	Complexity  string
	TaskType    string
	Notes       string
	Deadline    *time.Time
}

type TaskResult struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	PageID      string `json:"page_id,omitempty"`
	TaskName    string `json:"task_name,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	URL         string `json:"url,omitempty"`
}

type pageResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (c *Client) CreateTask(ctx context.Context, task Task) (*TaskResult, error) {
	if c.kanbanDB == "" {
		return nil, fmt.Errorf("notion kanban_database_id is not configured")
	}

	priority := task.Priority
	if priority == "" {
		priority = "Medium Priority"
	}
	complexity := task.Complexity
	if complexity == "" {
		complexity = "Normal Complexity"
	}
	taskType := task.TaskType
	if taskType == "" {
		taskType = "PROJECT TASK"
	}

	props := map[string]any{
		"Name":         titleProp(task.Name),
		"Project Name": multiSelectProp(task.ProjectName),
		"Status":       selectProp("To-do"),
		"Priority":     selectProp(priority),
		"Complexity":   selectProp(complexity),
		"Task type":    multiSelectProp(taskType),
	}

	if task.Notes != "" {
		props["Notes"] = richTextProp(task.Notes)
	}
	if task.Deadline != nil {
		props["Deadline"] = map[string]any{
			"date": map[string]any{"start": task.Deadline.Format("2006-01-02")},
		}
	}

	var resp pageResponse
	err := c.do(ctx, methodPost, "/pages", map[string]any{
		"parent":     map[string]any{"database_id": c.kanbanDB},
		"properties": props,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("creating Notion task: %w", err)
	}

	return &TaskResult{
		Status:      "success",
		Message:     fmt.Sprintf("Задача '%s' успешно создана", task.Name),
		PageID:      resp.ID,
		TaskName:    task.Name,
		ProjectName: task.ProjectName,
		URL:         resp.URL,
	}, nil
}
