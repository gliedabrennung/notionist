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
- **Markdown-formatted replies** back to Telegram.

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

## Setting up Notion

The bot talks to Notion through an **internal integration** and writes into two
databases you create. No external library is used — the app calls the Notion REST
API (`Notion-Version: 2022-06-28`) directly.

### 1. Create an integration and get the token

1. Open <https://www.notion.so/my-integrations> and click **New integration**.
2. Give it a name, pick the workspace, and keep the default capabilities
   (the integration needs **Read**, **Update** and **Insert** content).
3. Copy the **Internal Integration Secret** — it looks like `ntn_...`
   (older tokens start with `secret_`). This is your `NOTION_API_KEY`.

### 2. Create the two databases

Create two databases (or reuse existing ones) in your workspace:

- **Kanban database** — where tasks and projects live.
- **Documentation database** — where ТЗ / technical specs are stored.

For each database, open it, click **··· → Connections**, and add the integration
you just created. The integration needs **edit** access. Without this connection,
every API call returns a permission error.

### 3. Get the database IDs

Open each database as a page and copy the ID from the URL:

```
https://www.notion.so/<workspace>/<32-char-hex-id>?v=...
```

The `<32-char-hex-id>` is the database ID — it must contain the hyphens, e.g.
`1c33b84f-1fac-8055-a0f3-xxxxxxxxxxxx`. Put them into
`NOTION_KANBAN_DATABASE_ID` and `NOTION_DOCS_DATABASE_ID` (or the matching
`config.yaml` keys).

### 4. Configure the Kanban database schema

The Kanban database **must** have the following properties (names are matched
exactly as written):

| Property | Type | Required | Notes / Options |
| --- | --- | --- | --- |
| `Name` | Title | yes | Task name (always set). |
| `Project Name` | Multi-select | yes | Project the task belongs to. |
| `Status` | **Status** | yes | Must be the dedicated **Status** property type, not *Select*. The app sets it to `To-do`. Suggested options: `To-do`, `In-progress`, `Done`. |
| `Priority` | Select | yes | Suggested options: `High Priority`, `Medium Priority`, `Low Priority` (default). |
| `Complexity` | Select | yes | Suggested options: `Hard Complexity`, `Normal Complexity` (default), `Easy Complexity`. |
| `Task type` | Multi-select | yes | Suggested options: `PROJECT TASK` (default), `DESIGN`, `DOCUMENTATION`. |
| `Deadline` | Date | no | Set only when the task has a deadline. |
| `Notes` | Text (rich text) | no | Extra context; set only when provided. |

> `Status` must be a **Status** property, not a plain **Select** — Notion models
> them differently and the app writes a status group name, not a select option.

### 5. Configure the documentation database schema

The documentation database is more permissive:

- It must have **one Title property** — the app auto-detects its real name
  (it does not assume `Doc name`), so you can name it whatever you like.
- A `Category` **Multi-select** property is **optional**. If it exists it is
  filled in; if it does not, it is silently skipped (no error).

### 6. Provide the credentials

Export the three values, or put them (expanded from `${VAR}`) into `config.yaml`:

```bash
export NOTION_API_KEY="ntn_..."
export KANBAN_DATABASE_ID="1c33b84f-1fac-8055-a0f3-xxxxxxxxxxxx"
export DOCS_DATABASE_ID="2d44c95f-2fbd-9166-b1f4-yyyyyyyyyyyy"
```

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
