package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gliedabrennung/notionist/internal/notion"
	adkagent "google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func newNotionTools(notionClient *notion.Client) ([]tool.Tool, error) {
	builders := []func(*notion.Client) (tool.Tool, error){
		newCreateTaskInKanbanTool,
		newCreateProjectInKanbanTool,
		newAppendBlockChildrenTool,
		newGetMarkdownBlocksJSONTool,
		newCreateNotionWorkflowInstructionTool,
		newAnalyzeTechnicalRequirementsTool,
		newSaveTZToDocumentationTool,
		newProcessTZToKanbanTool,
	}

	tools := make([]tool.Tool, 0, len(builders))
	for _, b := range builders {
		t, err := b(notionClient)
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, nil
}

type createTaskInKanbanArgs struct {
	TaskName    string `json:"task_name" jsonschema:"Name of the task."`
	ProjectName string `json:"project_name,omitempty" jsonschema:"Name of the project this task belongs to."`
	Priority    string `json:"priority,omitempty" jsonschema:"Priority: 'High Priority', 'Medium Priority' or 'Low Priority'."`
	Complexity  string `json:"complexity,omitempty" jsonschema:"Complexity: 'Hard Complexity', 'Normal Complexity' or 'Easy Complexity'."`
	TaskType    string `json:"task_type,omitempty" jsonschema:"Task type, e.g. 'PROJECT TASK', 'DESIGN', 'DOCUMENTATION'."`
	Notes       string `json:"notes,omitempty" jsonschema:"Additional notes for the task."`
	Deadline    string `json:"deadline,omitempty" jsonschema:"Deadline as YYYY-MM-DD."`
}

func newCreateTaskInKanbanTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "create_task_in_kanban",
		Description: "Create a task in the Notion Kanban database. Use this whenever the user wants to remember, track or do something.",
	}
	handler := func(ctx adkagent.Context, args createTaskInKanbanArgs) (notion.TaskResult, error) {
		task := notion.Task{
			Name:        args.TaskName,
			ProjectName: args.ProjectName,
			Priority:    args.Priority,
			Complexity:  args.Complexity,
			TaskType:    args.TaskType,
			Notes:       args.Notes,
		}
		if args.Deadline != "" {
			d, err := parseDate(args.Deadline)
			if err != nil {
				return notion.TaskResult{}, err
			}
			task.Deadline = &d
		}
		res, err := c.CreateTask(ctx, task)
		if err != nil {
			return notion.TaskResult{}, fmt.Errorf("creating Notion task: %w", err)
		}
		return *res, nil
	}
	return functiontool.New(cfg, handler)
}

type createProjectInKanbanArgs struct {
	ProjectName string `json:"project_name" jsonschema:"Name of the project to add to the Kanban board."`
}

func newCreateProjectInKanbanTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "create_project_in_kanban",
		Description: "Add a new project to the 'Project Name' multi-select of the Kanban database.",
	}
	handler := func(ctx adkagent.Context, args createProjectInKanbanArgs) (notion.ProjectResult, error) {
		res, err := c.CreateProject(ctx, args.ProjectName)
		if err != nil {
			return notion.ProjectResult{}, fmt.Errorf("creating project: %w", err)
		}
		return *res, nil
	}
	return functiontool.New(cfg, handler)
}

type appendBlockChildrenArgs struct {
	BlockID  string           `json:"block_id" jsonschema:"Identifier of the parent block or page to append children to."`
	Children []map[string]any `json:"children" jsonschema:"List of Notion block objects to append."`
}

func newAppendBlockChildrenTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "append_block_children_unrestricted",
		Description: "Append a list of Notion block objects to a parent block or page. Supports any block type (headings, code, tables, callouts, etc.).",
	}
	handler := func(ctx adkagent.Context, args appendBlockChildrenArgs) (notion.StatusResult, error) {
		res, err := c.AppendBlockChildren(ctx, args.BlockID, args.Children)
		if err != nil {
			return notion.StatusResult{}, fmt.Errorf("appending blocks: %w", err)
		}
		return *res, nil
	}
	return functiontool.New(cfg, handler)
}

type getMarkdownBlocksJSONArgs struct {
	MarkdownContent string `json:"markdown_content" jsonschema:"The Markdown content to convert to Notion blocks."`
}

type markdownBlocksResult struct {
	Status          string           `json:"status"`
	Message         string           `json:"message,omitempty"`
	TotalBlocks     int              `json:"total_blocks"`
	Chunks          int              `json:"chunks"`
	FirstChunk      []notion.Block   `json:"first_chunk"`
	RemainingChunks [][]notion.Block `json:"remaining_chunks"`
}

func newGetMarkdownBlocksJSONTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "get_markdown_blocks_json",
		Description: "Convert Markdown content into Notion block JSON, split into API-safe chunks.",
	}
	handler := func(ctx adkagent.Context, args getMarkdownBlocksJSONArgs) (markdownBlocksResult, error) {
		blocks := notion.MarkdownToNotionBlocks(args.MarkdownContent)
		chunks := notion.ChunkBlocks(blocks, 80)
		var first []notion.Block
		var remaining [][]notion.Block
		if len(chunks) > 0 {
			first = chunks[0]
			remaining = chunks[1:]
		}
		return markdownBlocksResult{
			Status:          "success",
			TotalBlocks:     len(blocks),
			Chunks:          len(chunks),
			FirstChunk:      first,
			RemainingChunks: remaining,
		}, nil
	}
	return functiontool.New(cfg, handler)
}

type createNotionWorkflowInstructionArgs struct {
	DatabaseName    string `json:"database_name" jsonschema:"Name of the database to create the page in."`
	PageTitle       string `json:"page_title" jsonschema:"Title for the new page."`
	MarkdownContent string `json:"markdown_content" jsonschema:"The Markdown content."`
}

type workflowInstructionResult struct {
	Status       string `json:"status"`
	Instructions string `json:"instructions"`
	TotalChunks  int    `json:"total_chunks"`
	TotalBlocks  int    `json:"total_blocks"`
}

func newCreateNotionWorkflowInstructionTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "create_notion_workflow_instruction",
		Description: "Generate step-by-step instructions for creating a Notion page from Markdown using the available tools.",
	}
	handler := func(ctx adkagent.Context, args createNotionWorkflowInstructionArgs) (workflowInstructionResult, error) {
		blocks := notion.MarkdownToNotionBlocks(args.MarkdownContent)
		chunks := notion.ChunkBlocks(blocks, 100)

		var b strings.Builder
		b.WriteString("NOTION PAGE CREATION WORKFLOW\n")
		b.WriteString("=============================\n\n")
		fmt.Fprintf(&b, "OBJECTIVE: Create page %q in database %q\n\n", args.PageTitle, args.DatabaseName)
		b.WriteString("STEP 1: Find Database\n--------------------\n")
		b.WriteString("Use tool: API-post-search\n")
		fmt.Fprintf(&b, "Parameters: {\"query\": %q, \"filter\": {\"value\": \"database\", \"property\": \"object\"}}\n", args.DatabaseName)
		b.WriteString("→ Extract database ID from the first result\n\n")
		b.WriteString("STEP 2: Create Empty Page\n-------------------------\n")
		b.WriteString("Use tool: API-post-page\n")
		b.WriteString("Parameters: {\"parent\": {\"database_id\": \"DATABASE_ID_FROM_STEP_1\"}, \"properties\": {\"Doc name\": {\"title\": [{\"text\": {\"content\": \"" + args.PageTitle + "\"}}]}}}\n")
		b.WriteString("→ Extract page ID from the response\n\n")
		b.WriteString("STEP 3: Convert Markdown to Blocks\n----------------------------------\n")
		b.WriteString("Use tool: get_markdown_blocks_json\n")
		fmt.Fprintf(&b, "Parameters: %s\n→ Extract blocks from the response\n\n", args.MarkdownContent)
		fmt.Fprintf(&b, "STEP 4: Add Content (%d chunks, %d total blocks)\n------------------------------------------------------------------\n", len(chunks), len(blocks))

		for i, chunk := range chunks {
			fmt.Fprintf(&b, "\nChunk %d/%d - Use tool: append_block_children_unrestricted\n", i+1, len(chunks))
			fmt.Fprintf(&b, "Parameters: {\"block_id\": \"PAGE_ID_FROM_STEP_2\", \"children\": %s}\n", mustJSON(chunk))
		}
		b.WriteString("\nCOMPLETION\n----------\nAll content has been added to the Notion page successfully using the append_block_children_unrestricted tool.\n")

		return workflowInstructionResult{
			Status:       "success",
			Instructions: b.String(),
			TotalChunks:  len(chunks),
			TotalBlocks:  len(blocks),
		}, nil
	}
	return functiontool.New(cfg, handler)
}

type analyzeTechnicalRequirementsArgs struct {
	TZContent string `json:"tz_content" jsonschema:"The technical specification content to analyze."`
}

func newAnalyzeTechnicalRequirementsTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "analyze_technical_requirements",
		Description: "Analyze a technical specification (ТЗ) and extract the project name and a list of tasks with priorities, complexity and types.",
	}
	handler := func(ctx adkagent.Context, args analyzeTechnicalRequirementsArgs) (notion.AnalysisResult, error) {
		res, err := notion.AnalyzeTechnicalRequirements(args.TZContent)
		if err != nil {
			return notion.AnalysisResult{}, fmt.Errorf("analyzing ТЗ: %w", err)
		}
		return *res, nil
	}
	return functiontool.New(cfg, handler)
}

type saveTZToDocumentationArgs struct {
	TZContent  string `json:"tz_content" jsonschema:"The technical specification content."`
	DatabaseID string `json:"database_id,omitempty" jsonschema:"Optional docs database ID; defaults to the configured one."`
}

func newSaveTZToDocumentationTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "save_tz_to_documentation",
		Description: "Save a technical specification as a document page in the Notion documentation database.",
	}
	handler := func(ctx adkagent.Context, args saveTZToDocumentationArgs) (notion.DocumentationResult, error) {
		res, err := c.SaveTZToDocumentation(ctx, args.TZContent, args.DatabaseID)
		if err != nil {
			return notion.DocumentationResult{}, fmt.Errorf("saving documentation: %w", err)
		}
		return *res, nil
	}
	return functiontool.New(cfg, handler)
}

type processTZToKanbanArgs struct {
	TZContent           string `json:"tz_content" jsonschema:"The technical specification content to process."`
	ProjectNameOverride string `json:"project_name_override,omitempty" jsonschema:"Optional override for the project name."`
}

func newProcessTZToKanbanTool(c *notion.Client) (tool.Tool, error) {
	cfg := functiontool.Config{
		Name:        "process_tz_to_kanban",
		Description: "Full pipeline: save the ТЗ as documentation, analyze it, create the project and create all tasks in the Kanban board.",
	}
	handler := func(ctx adkagent.Context, args processTZToKanbanArgs) (notion.ProcessResult, error) {
		res, err := c.ProcessTZToKanban(ctx, args.TZContent, args.ProjectNameOverride)
		if err != nil {
			return notion.ProcessResult{}, fmt.Errorf("processing ТЗ: %w", err)
		}
		return *res, nil
	}
	return functiontool.New(cfg, handler)
}

func parseDate(s string) (time.Time, error) {
	now := time.Now()
	switch s {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	case "tomorrow":
		t := now.AddDate(0, 0, 1)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location()), nil
	case "next week":
		t := now.AddDate(0, 0, 7)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	for _, layout := range []string{"2006-01-02", "02.01.2006", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported due date format: %q", s)
}
