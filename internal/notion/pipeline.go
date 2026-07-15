package notion

import (
	"context"
	"fmt"
)

type ProcessResult struct {
	Status               string            `json:"status"`
	Message              string            `json:"message,omitempty"`
	ProjectName          string            `json:"project_name"`
	DocumentationCreated bool              `json:"documentation_created"`
	DocumentationInfo    map[string]string `json:"documentation_info,omitempty"`
	DocumentationError   string            `json:"documentation_error,omitempty"`
	ProjectCreated       string            `json:"project_created,omitempty"`
	TotalTasks           int               `json:"total_tasks"`
	CreatedTasks         int               `json:"created_tasks"`
	FailedTasks          int               `json:"failed_tasks"`
	TasksDetails         []TaskInfo        `json:"tasks_details,omitempty"`
	FailedTasksDetails   []TaskInfo        `json:"failed_tasks_details,omitempty"`
	Summary              string            `json:"summary"`
}

func (c *Client) ProcessTZToKanban(ctx context.Context, tz string, projectNameOverride string) (*ProcessResult, error) {
	result := &ProcessResult{}

	docResult, docErr := c.SaveTZToDocumentation(ctx, tz, "")
	if docErr == nil && docResult.Status == "success" {
		result.DocumentationCreated = true
		result.DocumentationInfo = map[string]string{
			"title": docResult.DocTitle,
			"url":   docResult.PageURL,
		}
	} else if docErr != nil {
		result.DocumentationError = docErr.Error()
	} else {
		result.DocumentationError = docResult.Message
	}

	analysis, err := AnalyzeTechnicalRequirements(tz)
	if err != nil {
		return nil, fmt.Errorf("analyzing ТЗ: %w", err)
	}
	if analysis.Status != "success" {
		return nil, fmt.Errorf("analysis failed: %s", analysis.Message)
	}

	projectName := orDefault(projectNameOverride, analysis.ProjectName)

	projResult, err := c.CreateProject(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}
	result.ProjectCreated = projResult.Message

	created := make([]TaskInfo, 0)
	failed := make([]TaskInfo, 0)
	for _, t := range analysis.Tasks {
		_, terr := c.CreateTask(ctx, Task{
			Name:        t.Name,
			ProjectName: projectName,
			Priority:    t.Priority,
			Complexity:  t.Complexity,
			TaskType:    t.TaskType,
			Notes:       t.Notes,
		})
		if terr != nil {
			failed = append(failed, t)
		} else {
			created = append(created, t)
		}
	}

	result.ProjectName = projectName
	result.TotalTasks = analysis.TotalTasks
	result.CreatedTasks = len(created)
	result.FailedTasks = len(failed)
	result.TasksDetails = created
	result.FailedTasksDetails = failed

	summary := fmt.Sprintf("✅ Проект '%s' создан с %d задачами", projectName, len(created))
	result.Summary = summary

	return result, nil
}
