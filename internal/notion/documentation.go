package notion

import (
	"context"
	"fmt"
	"strings"
)

type DocumentationResult struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	PageID       string `json:"page_id,omitempty"`
	PageURL      string `json:"page_url,omitempty"`
	DocTitle     string `json:"doc_title,omitempty"`
	Category     string `json:"category,omitempty"`
	BlocksAdded  int    `json:"blocks_added,omitempty"`
	TotalBlocks  int    `json:"total_blocks,omitempty"`
	TotalChunks  int    `json:"total_chunks,omitempty"`
	ChunksAdded  int    `json:"chunks_added,omitempty"`
	FailedChunks int    `json:"failed_chunks,omitempty"`
	SuccessRate  string `json:"success_rate,omitempty"`
	Summary      string `json:"summary,omitempty"`
}

func extractDocTitle(tz string) string {
	docTitle := "Техническое задание"
	for _, line := range strings.Split(tz, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "название проекта") || strings.Contains(lower, "проект:") {
			if i := strings.Index(line, "**"); i >= 0 {
				parts := strings.Split(line, "**")
				if len(parts) > 1 {
					docTitle = strings.TrimSpace(parts[1]) + " - ТЗ"
				}
			} else if i := strings.Index(line, ":"); i >= 0 {
				docTitle = strings.TrimSpace(line[i+1:]) + " - ТЗ"
			}
			break
		}
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "##") {
			docTitle = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if strings.ToLower(docTitle) != "техническое задание" {
				docTitle += " - ТЗ"
			}
			break
		}
	}
	return docTitle
}

func (c *Client) SaveTZToDocumentation(ctx context.Context, tz string, databaseID string) (*DocumentationResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("NOTION_API_KEY is not set")
	}
	if databaseID == "" {
		databaseID = c.docsDB
	}
	if databaseID == "" {
		return nil, fmt.Errorf("notion docs_database_id is not configured")
	}

	docTitle := extractDocTitle(tz)

	blocks := MarkdownToNotionBlocks(tz)
	chunks := ChunkBlocks(blocks, 80)
	var firstChunk []Block
	if len(chunks) > 0 {
		firstChunk = chunks[0]
	}

	page, err := c.createDocPage(ctx, databaseID, docTitle, firstChunk)
	if err != nil {
		return nil, err
	}

	return &DocumentationResult{
		Status:      "success",
		Message:     "ТЗ успешно сохранено как документ",
		PageID:      page.ID,
		PageURL:     page.URL,
		DocTitle:    docTitle,
		Category:    "Planning",
		BlocksAdded: len(firstChunk),
		TotalBlocks: len(blocks),
		Summary:     fmt.Sprintf("✅ Создан документ '%s' в базе данных", docTitle),
	}, nil
}

func (c *Client) SaveTZToDocumentationComplete(ctx context.Context, tz string, databaseID string) (*DocumentationResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("NOTION_API_KEY is not set")
	}
	if databaseID == "" {
		databaseID = c.docsDB
	}
	if databaseID == "" {
		return nil, fmt.Errorf("notion docs_database_id is not configured")
	}

	docTitle := extractDocTitle(tz)

	blocks := MarkdownToNotionBlocks(tz)
	chunks := ChunkBlocks(blocks, 80)
	totalChunks := len(chunks)
	if totalChunks == 0 {
		totalChunks = 1
	}
	var firstChunk []Block
	if len(chunks) > 0 {
		firstChunk = chunks[0]
	}
	remaining := chunks[1:]

	page, err := c.createDocPage(ctx, databaseID, docTitle, firstChunk)
	if err != nil {
		return nil, err
	}

	chunksAdded := 1
	failed := 0
	for i, chunk := range remaining {
		if _, aerr := c.AppendBlockChildren(ctx, page.ID, chunk); aerr != nil {
			failed++
			fmt.Printf("⚠️ Ошибка добавления чанка %d: %v\n", i+2, aerr)
		} else {
			chunksAdded++
		}
	}

	successRate := float64(chunksAdded) / float64(totalChunks)
	status := "success"
	if successRate < 0.8 {
		status = "partial_success"
	}

	return &DocumentationResult{
		Status:       status,
		Message:      fmt.Sprintf("ТЗ сохранено с %d/%d чанками", chunksAdded, totalChunks),
		PageID:       page.ID,
		PageURL:      page.URL,
		DocTitle:     docTitle,
		Category:     "Planning",
		TotalChunks:  totalChunks,
		ChunksAdded:  chunksAdded,
		FailedChunks: failed,
		TotalBlocks:  len(blocks),
		SuccessRate:  fmt.Sprintf("%.1f%%", successRate*100),
		Summary:      fmt.Sprintf("✅ Создан документ '%s' с %d/%d чанками (%.1f%% успешно)", docTitle, chunksAdded, totalChunks, successRate*100),
	}, nil
}

func (c *Client) createDocPage(ctx context.Context, databaseID, docTitle string, children []Block) (*pageResponse, error) {
	var resp pageResponse
	err := c.do(ctx, methodPost, "/pages", map[string]any{
		"parent": map[string]any{"database_id": databaseID},
		"properties": map[string]any{
			"Doc name": titleProp(docTitle),
			"Category": multiSelectProp("Planning"),
		},
		"children": children,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("creating documentation page: %w", err)
	}
	return &resp, nil
}
