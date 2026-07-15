package notion

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type Block = map[string]any

func MarkdownToNotionBlocks(markdown string) []Block {
	blocks := make([]Block, 0)
	lines := strings.Split(markdown, "\n")

	numberedRe := regexp.MustCompile(`^\d+\. `)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "# "):
			blocks = append(blocks, heading("heading_1", line[2:]))
		case strings.HasPrefix(line, "## "):
			blocks = append(blocks, heading("heading_2", line[3:]))
		case strings.HasPrefix(line, "### "):
			blocks = append(blocks, heading("heading_3", line[4:]))
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			blocks = append(blocks, listItem("bulleted_list_item", line[2:]))
		case strings.HasPrefix(line, "1. ") || numberedRe.MatchString(line):
			content := regexp.MustCompile(`^\d+\. `).ReplaceAllString(line, "")
			blocks = append(blocks, listItem("numbered_list_item", content))
		case strings.HasPrefix(line, "```"):
			codeLines := make([]string, 0)
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				codeLines = append(codeLines, lines[i])
				i++
			}
			blocks = append(blocks, Block{
				"type": "code",
				"code": map[string]any{
					"rich_text": []map[string]any{textObject(strings.Join(codeLines, "\n"))},
					"language":  "plain_text",
				},
			})
		case strings.HasPrefix(line, "|") && strings.ContainsRune(line[1:], '|'):

			blocks = append(blocks, paragraph(line))
		default:
			blocks = append(blocks, paragraph(line))
		}
	}

	return blocks
}

func ChunkBlocks(blocks []Block, chunkSize int) [][]Block {
	if chunkSize <= 0 {
		chunkSize = 100
	}
	chunks := make([][]Block, 0)
	for i := 0; i < len(blocks); i += chunkSize {
		end := i + chunkSize
		if end > len(blocks) {
			end = len(blocks)
		}
		chunks = append(chunks, blocks[i:end])
	}
	return chunks
}

func (c *Client) AppendBlockChildren(ctx context.Context, blockID string, children []Block) (*StatusResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("NOTION_API_KEY is not set")
	}
	if len(children) == 0 {
		return &StatusResult{Status: "success", Message: "No blocks to append."}, nil
	}

	var resp struct {
		Results []map[string]any `json:"results"`
	}
	err := c.do(ctx, methodPatch, "/blocks/"+blockID+"/children", map[string]any{
		"children": children,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("appending blocks: %w", err)
	}

	return &StatusResult{
		Status:  "success",
		Message: fmt.Sprintf("Successfully appended %d blocks.", len(resp.Results)),
	}, nil
}

func heading(t, content string) Block {
	return Block{
		"type": t,
		t: map[string]any{
			"rich_text": []map[string]any{textObject(content)},
		},
	}
}

func listItem(t, content string) Block {
	return Block{
		"type": t,
		t: map[string]any{
			"rich_text": []map[string]any{textObject(content)},
		},
	}
}

func paragraph(content string) Block {
	return Block{
		"type": "paragraph",
		"paragraph": map[string]any{
			"rich_text": []map[string]any{textObject(content)},
		},
	}
}

func textObject(content string) map[string]any {
	return map[string]any{"type": "text", "text": map[string]any{"content": content}}
}

type StatusResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
