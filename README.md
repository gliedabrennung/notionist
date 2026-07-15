# notionist

Telegram → Notion assistant built with the **Google Agent Development Kit for Go**
(`google.golang.org/adk/v2`). A Telegram bot receives a message, forwards it to an
ADK `LlmAgent` (Gemini), which decides which Notion tool to call — creating tasks
and documentation in your Notion workspace, with a focus on turning technical
specifications (ТЗ) into structured Kanban workflows.

## Features

- **Task creation** in a Notion Kanban board with priority, complexity, task type,
  notes and deadline.
- **Project management** — adds projects to the board's `Project Name` multi-select.
- **Technical-specification processing** — analyzes a ТЗ, saves it as a documentation
  page, and creates the project + all tasks in one automated pipeline.
- **Markdown → Notion blocks** conversion and direct block appending.
- **HTML-formatted replies** back to Telegram (per the agent instruction).

## Architecture

```
cmd/bot/main.go        Composition root: loads config, builds the agent, starts the bot.
internal/config        Loads config.yaml (env vars expanded via ${VAR}).
internal/notion        Notion REST client (raw HTTP, Notion-Version 2022-06-28).
internal/agent         ADK LlmAgent + Runner + the Notion function tools.
internal/telegram      go-telegram/bot handler that forwards messages to the agent.
prompt.yaml            Agent instruction (mirrors ADK's declarative `instruction:` key).
```

The Notion tools are a Go port of the Python `tools.py` (Google ADK `FunctionTool`s).

### Notion tools (registered as ADK function tools)

- `create_task_in_kanban` — create a task in the Kanban database.
- `create_project_in_kanban` — add a project to the Kanban `Project Name` multi-select.
- `process_tz_to_kanban` — full pipeline: documentation + project + tasks.
- `analyze_technical_requirements` — extract project + tasks from a ТЗ.
- `save_tz_to_documentation` — save a ТЗ as a documentation page.
- `get_markdown_blocks_json` — convert Markdown to Notion blocks (chunked).
- `append_block_children_unrestricted` — append arbitrary blocks to a page/block.
- `create_notion_workflow_instruction` — generate a page-creation workflow.

## Configuration

Copy `config.example.yaml` to `config.yaml` and fill in your values. Values support
environment-variable expansion (`${VAR}`).

```yaml
telegram:
  token: "${TELEGRAM_BOT_TOKEN}"

gemini:
  api_key: "${GEMINI_API_KEY}"
  model: "gemini-2.5-flash"      # optional, has a default

notion:
  api_key: "${NOTION_API_KEY}"
  kanban_database_id: "${KANBAN_DATABASE_ID}"
  docs_database_id: "${DOCS_DATABASE_ID}"

database:
  url: "sqlite:///sessions.db"    # reserved for future session persistence
```

| Key | Description |
| --- | --- |
| `telegram.token` | Telegram bot token from @BotFather. |
| `gemini.api_key` | Google AI / Gemini API key. |
| `gemini.model` | Gemini model name (defaults to `gemini-2.5-flash`). |
| `notion.api_key` | Notion integration token. |
| `notion.kanban_database_id` | ID of the Kanban tasks database. |
| `notion.docs_database_id` | ID of the documentation database. |

The agent instruction is loaded from `prompt.yaml` in the repository root. Override
the path with the `PROMPT_PATH` env var.

## Running

```bash
go run ./cmd/bot
```

Make sure the required environment variables (or literal values in `config.yaml`)
are set, then send a message to your Telegram bot:

- A plain task → the agent creates a Kanban task.
- A technical specification (ТЗ) → use `process_tz_to_kanban` to document it and
  create the project with all tasks.

### Docker

`config.yaml` holds secrets and is gitignored, so it is excluded from the image
build (see `.dockerignore`). Provide it at runtime by mounting the file:

```bash
docker build -t notionist .
docker run --rm -v "$PWD/config.yaml:/app/config.yaml" notionist
```

`prompt.yaml` is baked into the image; override its path with the `PROMPT_PATH`
env var if needed.

## Project layout

```
.
├── cmd/bot/main.go
├── internal/
│   ├── agent/      # ADK agent, tools, prompt loader
│   ├── config/     # config loading
│   ├── notion/     # Notion REST client + tools
│   └── telegram/   # Telegram bot handler
├── prompt.yaml     # agent instruction
├── config.example.yaml
└── config.yaml     # gitignored, your secrets
```
