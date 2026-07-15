# Agent Prompt: Telegram → Notion Task Creator (Google ADK for Go)

This project uses **Google Agent Development Kit for Go** (`google.golang.org/adk/v2`).
The agent is implemented in Go, not Python. The Telegram bot receives a message,
forwards it to an ADK `LlmAgent` (Gemini), which decides which function tool to call.
Tools write pages into the Notion Kanban / documentation databases and reply with a
short confirmation.

The Notion tools are a faithful Go port of `tools.py` (Python ADK `FunctionTool`s).

## System / role instruction (passed to the LLM agent)
Lives in `prompt.yaml` at the repository root (mirrors ADK's declarative
`instruction:` YAML key). Override the path with the `PROMPT_PATH` env var.

## Tools (ported from tools.py)
All registered as ADK function tools in `internal/agent/tools.go`:
- `create_task_in_kanban` — create a task in the Kanban DB
  (task_name, project_name, priority, complexity, task_type, notes, deadline)
- `create_project_in_kanban` — add a project to the Kanban "Project Name" multi-select
- `append_block_children_unrestricted` — append arbitrary Notion blocks to a page/block
- `get_markdown_blocks_json` — convert Markdown to Notion blocks (chunked)
- `create_notion_workflow_instruction` — generate a page-creation workflow
- `analyze_technical_requirements` — extract project + tasks from a ТЗ
- `save_tz_to_documentation` — save a ТЗ as a doc page
- `process_tz_to_kanban` — full pipeline: doc + project + tasks

The tool handlers call `notion.Client` methods, which talk to the Notion REST API
(`https://api.notion.com/v1`, `Notion-Version: 2022-06-28`) over raw HTTP.

## Wiring
- `internal/config` — loads `telegram.token`, `gemini.api_key`, `gemini.model`,
  `notion.api_key`, `notion.kanban_database_id`, `notion.docs_database_id`.
- `internal/notion` — Notion REST client (raw HTTP, no external Notion SDK).
- `internal/agent` — ADK `LlmAgent` + `runner.Runner` + the Notion function tools.
  The agent instruction lives in `prompt.yaml` at the repository root.
- `internal/telegram` — go-telegram/bot handler that calls
  `TaskAgent.ProcessMessage` and sends the reply back.
- `cmd/bot/main.go` — composition root.

## How to extend
1. Add more tools via `functiontool.New` in `internal/agent/tools.go` and append to
   the `newNotionTools` slice.
2. Adjust the instruction in `prompt.yaml`.
3. Run with env vars from `config.example.yaml` (use `os.ExpandEnv` /
   real env vars for secrets).
