package notion

import (
	"context"
	"encoding/json"
	"fmt"
)

type ProjectResult struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	ProjectID   string `json:"project_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	Color       string `json:"color,omitempty"`
}

type databaseResponse struct {
	Properties map[string]any `json:"properties"`
}

type multiSelectProperty struct {
	Options []map[string]any `json:"multi_select"`
}

func (c *Client) CreateProject(ctx context.Context, projectName string) (*ProjectResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("NOTION_API_KEY is not set")
	}
	if c.kanbanDB == "" {
		return nil, fmt.Errorf("notion kanban_database_id is not configured")
	}

	var db databaseResponse
	if err := c.do(ctx, methodGet, "/databases/"+c.kanbanDB, nil, &db); err != nil {
		return nil, fmt.Errorf("getting database: %w", err)
	}

	propRaw, ok := db.Properties["Project Name"]
	if !ok {
		return nil, fmt.Errorf("database has no 'Project Name' property")
	}
	propBytes, err := json.Marshal(propRaw)
	if err != nil {
		return nil, err
	}
	var ms multiSelectProperty
	if err := json.Unmarshal(propBytes, &ms); err != nil {
		return nil, fmt.Errorf("decoding Project Name property: %w", err)
	}

	for _, opt := range ms.Options {
		if name, _ := opt["name"].(string); name == projectName {
			return &ProjectResult{
				Status:      "success",
				Message:     fmt.Sprintf("Проект '%s' уже существует", projectName),
				ProjectID:   fmt.Sprintf("%v", opt["id"]),
				ProjectName: name,
			}, nil
		}
	}

	colors := []string{"blue", "brown", "default", "gray", "green", "orange", "pink", "purple", "red", "yellow"}
	used := make(map[string]bool, len(ms.Options))
	for _, opt := range ms.Options {
		if col, _ := opt["color"].(string); col != "" {
			used[col] = true
		}
	}
	newColor := "default"
	for _, col := range colors {
		if !used[col] {
			newColor = col
			break
		}
	}

	newOption := map[string]any{"name": projectName, "color": newColor}
	ms.Options = append(ms.Options, newOption)

	update := map[string]any{
		"properties": map[string]any{
			"Project Name": map[string]any{"multi_select": map[string]any{"options": ms.Options}},
		},
	}
	if err := c.do(ctx, methodPatch, "/databases/"+c.kanbanDB, update, nil); err != nil {
		return nil, fmt.Errorf("updating database: %w", err)
	}

	return &ProjectResult{
		Status:      "success",
		Message:     fmt.Sprintf("Проект '%s' успешно добавлен", projectName),
		ProjectName: projectName,
		Color:       newColor,
	}, nil
}
