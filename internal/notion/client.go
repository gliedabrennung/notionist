package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gliedabrennung/notionist/internal/config"
)

const (
	notionVersion = "2022-06-28"
	notionBaseURL = "https://api.notion.com/v1"

	methodGet    = "GET"
	methodPost   = "POST"
	methodPatch  = "PATCH"
	methodDelete = "DELETE"
)

type Client struct {
	apiKey   string
	kanbanDB string
	docsDB   string
	http     *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		apiKey:   cfg.Notion.APIKey,
		kanbanDB: cfg.Notion.KanbanDatabaseID,
		docsDB:   cfg.Notion.DocsDatabaseID,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

type notionError struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshalling request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, notionBaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("notion request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ne notionError
		if json.Unmarshal(data, &ne) == nil && ne.Message != "" {
			return fmt.Errorf("notion API error: %d - %s", resp.StatusCode, ne.Message)
		}
		return fmt.Errorf("notion API error: %d - %s", resp.StatusCode, string(data))
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

func titleProp(content string) map[string]any {
	return map[string]any{
		"title": []map[string]any{
			{"text": map[string]any{"content": content}},
		},
	}
}

func richTextProp(content string) map[string]any {
	return map[string]any{
		"rich_text": []map[string]any{
			{"text": map[string]any{"content": content}},
		},
	}
}

func selectProp(name string) map[string]any {
	return map[string]any{"select": map[string]any{"name": name}}
}

func statusProp(name string) map[string]any {
	return map[string]any{"status": map[string]any{"name": name}}
}

func multiSelectProp(names ...string) map[string]any {
	opts := make([]map[string]any, 0, len(names))
	for _, n := range names {
		if n == "" {
			continue
		}
		opts = append(opts, map[string]any{"name": n})
	}
	return map[string]any{"multi_select": opts}
}
