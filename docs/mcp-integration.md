# MCP Integration

Integrate Knowns with AI assistants via Model Context Protocol (MCP).

## What is MCP?

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) is a standard for connecting AI assistants to external tools and data sources. Knowns implements an MCP server that allows AI assistants to read and manage your tasks and documentation directly.

## Supported Platforms

| Platform | Config File | Scope | Auto-discover |
|----------|-------------|-------|---------------|
| **Claude Code** | `.mcp.json` | Per-project | ✅ |
| **Gemini CLI** | platform-managed | Global | ✅ |
| **Cursor** | `.cursor/mcp.json` | Per-project | ⚠️ Manual |
| **Claude Desktop** | app config file | Global | ⚠️ Manual |

## Session Initialization (CRITICAL for Global Configs)

For platforms with **global MCP config** (for example Gemini CLI or Claude Desktop), the MCP server doesn't know which project to work with. **Run these tools at session start:**

```json
// 1. Detect available Knowns projects
mcp__knowns__detect_projects({})

// 2. Set the active project
mcp__knowns__set_project({ "projectRoot": "/path/to/project" })

// 3. Verify project is set
mcp__knowns__get_current_project({})
```

> **Note:** Claude Code usually uses per-project `.mcp.json`, so session initialization is typically not required there.

## Setup

### Option A: Auto Setup (Recommended)

The easiest way to configure MCP:

```bash
# Setup MCP in Claude Code config
knowns mcp setup
```

`knowns mcp setup` exists in the current CLI, but the help output does not advertise extra flags. Keep this doc aligned with the shipped help unless new options are added to the binary.

**Alternative: During `knowns init`**

When you select **MCP** as the AI Guidelines type during init, Knowns automatically creates `.mcp.json`:

```
? AI Guidelines type: MCP
✓ Created .mcp.json for Claude Code MCP auto-discovery
```

### Option B: Manual Setup

### 1. Install Knowns

```bash
# Homebrew (macOS/Linux)
brew install knowns-dev/tap/knowns

# npm
npm install -g knowns

# bun
bun install -g knowns
```

### 2. Configure Your Platform

#### Claude Code (Per-project)

Create `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "knowns": {
      "command": "npx",
      "args": ["-y", "knowns", "mcp", "--stdio"]
    }
  }
}
```

#### Gemini CLI / Other Global MCP Clients

Use the same server command pattern in the client-specific config:

```json
{
  "mcpServers": {
    "knowns": {
      "command": "npx",
      "args": ["-y", "knowns", "mcp", "--stdio"]
    }
  }
}
```

> **Note:** For global MCP configs, use `detect_projects` and `set_project` at session start.

#### Claude Desktop (Global)

Edit Claude's configuration file:

**macOS/Linux (current CLI setup path):** `~/.claude/claude_desktop_config.json`

**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp", "--stdio"],
      "cwd": "/path/to/your/project"
    }
  }
}
```

> **Note**: Replace `/path/to/your/project` with your actual project path where `.knowns/` folder exists.

#### Cursor (Per-project)

Create `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "knowns": {
      "command": "npx",
      "args": ["-y", "knowns", "mcp", "--stdio"]
    }
  }
}
```

### 3. Restart Claude Desktop

Close and reopen Claude Desktop to load the new configuration.

### 4. Sync MCP Guidelines (Optional)

Sync your instruction files and skills with the current CLI-generated guidance:

```bash
# Sync everything
knowns sync

# Sync instruction files only
knowns sync --instructions

# Sync a single platform's instruction file
knowns sync --instructions --platform claude

# Sync skills only
knowns sync --skills
```

This updates generated instruction files to match the current binary. For Claude Desktop or Claude Code MCP usage, prefer `knowns mcp --stdio` when configuring the server process.

## Usage

Once configured, Claude can access your Knowns data automatically.

For any global MCP client, remember that most Knowns tools return `No project set. Call set_project first.` until a project has been selected.

### Example Conversation

```
You: "Start working on @task-pdyd2e"

Claude: [Reads task via MCP]
"I see task 'pdyd2e' 'Add user authentication' with these acceptance criteria:
- User can login with email/password
- JWT token is returned on success
- Invalid credentials return 401

The description references @doc/patterns/auth. Let me check that..."

[Fetches patterns/auth.md via MCP]

"Got it! The auth pattern uses JWT with 15-minute access tokens and
7-day refresh tokens. I'll implement following this pattern.

Starting timer and beginning implementation..."

[Starts time tracking via MCP]
```

## Available MCP Tools

### Project Tools (Session Initialization)

| Tool                  | Description             | Parameters                |
| --------------------- | ----------------------- | ------------------------- |
| `detect_projects`     | Scan for Knowns projects| `additionalPaths?` |
| `set_project`         | Set active project      | `projectRoot`             |
| `get_current_project` | Get current project     | -                         |

> **Required for global MCP configs**. Call at session start.

### Task Management

| Tool           | Description             | Parameters                                                                         |
| -------------- | ----------------------- | ---------------------------------------------------------------------------------- |
| `create_task`  | Create a new task       | `title`, `description?`, `status?`, `priority?`, `labels?`, `assignee?`, `parent?`, `fulfills?`, `spec?` |
| `get_task`     | Get task by ID          | `taskId`                                                                           |
| `update_task`  | Update task fields      | `taskId`, `status?`, `priority?`, `assignee?`, `labels?`, `addAc?`, `checkAc?`, `uncheckAc?`, `removeAc?`, `plan?`, `notes?`, `appendNotes?` |
| `list_tasks`   | List tasks with filters | `status?`, `priority?`, `assignee?`, `label?`                                      |

**update_task extended fields:**

| Field | Description |
| ----- | ----------- |
| `addAc` | Add acceptance criteria (array of strings) |
| `checkAc` | Check AC by index (1-based, array of numbers) |
| `uncheckAc` | Uncheck AC by index (1-based) |
| `removeAc` | Remove AC by index (1-based) |
| `plan` | Set implementation plan |
| `notes` | Replace implementation notes |
| `appendNotes` | Append to implementation notes |

### Time Tracking

| Tool              | Description          | Parameters                             |
| ----------------- | -------------------- | -------------------------------------- |
| `start_time`      | Start timer for task | `taskId`                               |
| `stop_time`       | Stop active timer    | `taskId`                               |
| `add_time`        | Manual time entry    | `taskId`, `duration`, `note?`, `date?` |
| `get_time_report` | Generate time report | `from?`, `to?`, `groupBy?`             |

### Documentation

| Tool          | Description     | Parameters                                                                          |
| ------------- | --------------- | ----------------------------------------------------------------------------------- |
| `list_docs`   | List all docs   | `tag?`                                                                              |
| `get_doc`     | Get doc content | `path`, `smart?`, `info?`, `toc?`, `section?`                                       |
| `create_doc`  | Create new doc  | `title`, `description?`, `content?`, `tags?`, `folder?`                             |
| `update_doc`  | Update doc      | `path`, `title?`, `description?`, `content?`, `appendContent?`, `tags?`, `section?` |

### Unified Search

| Tool     | Description                     | Parameters                                           |
| -------- | ------------------------------- | ---------------------------------------------------- |
| `search` | Search tasks + docs             | `query`, `type?` (all/task/doc), `mode?` (hybrid/semantic/keyword), `status?`, `priority?`, `label?`, `tag?`, `assignee?` |

**Large Document Workflow:**

```json
// Step 1: Check size
{ "path": "readme", "info": true }  // → estimatedTokens: 12132

// Step 2: Get TOC (if >2000 tokens)
{ "path": "readme", "toc": true }

// Step 3: Read/Edit specific section
{ "path": "readme", "section": "2" }  // Read section
// update_doc with section + content replaces only that section
```

### Templates

| Tool              | Description           | Parameters                            |
| ----------------- | --------------------- | ------------------------------------- |
| `list_templates`  | List all templates    | -                                     |
| `get_template`    | Get template config   | `name`                                |
| `run_template`    | Run template          | `name`, `variables`, `dryRun?`        |
| `create_template` | Create new template   | `name`, `description?`, `doc?`        |

**Template Workflow:**

```json
// Step 1: List available templates
mcp__knowns__list_templates({})

// Step 2: Get template details
mcp__knowns__get_template({ "name": "react-component" })

// Step 3: Preview (dry run)
mcp__knowns__run_template({
  "name": "react-component",
  "variables": { "name": "UserProfile", "withTest": true },
  "dryRun": true
})

// Step 4: Generate files
mcp__knowns__run_template({
  "name": "react-component",
  "variables": { "name": "UserProfile", "withTest": true },
  "dryRun": false
})
```

### Validation

| Tool       | Description                        | Parameters        |
| ---------- | ---------------------------------- | ----------------- |
| `validate` | Validate refs and file integrity   | `scope?`, `fix?`, `entity?`, `strict?`  |

**validate parameters:**

| Parameter | Description |
| --------- | ----------- |
| `scope`   | `all`, `tasks`, `docs`, `templates`, or `sdd` |
| `fix`     | Attempt to auto-fix issues (default: `false`) |

**Example:**

```json
// Validate all refs
mcp__knowns__validate({})

// Validate only tasks
mcp__knowns__validate({ "scope": "tasks" })
```

### Board

| Tool        | Description            | Parameters |
| ----------- | ---------------------- | ---------- |
| `get_board` | Get kanban board state | -          |

## Benefits

### Zero Context Loss

- AI reads your docs directly — no copy-paste
- References (`@doc/...`, `@task-<id>`) are resolved automatically
- Project patterns are always available

### Consistent Implementations

- AI follows your documented patterns every time
- Same conventions across all sessions
- No more "how does your auth work?" questions

### Perfect Memory

- Documentation persists between sessions
- AI can reference any past decision
- Knowledge doesn't live in chat history

### Time Tracking Integration

- AI can start/stop timers automatically
- Track time spent on each task
- Generate reports for retrospectives

## Troubleshooting

### Claude doesn't see Knowns

1. Verify installation: `knowns --version`
2. Check config file path is correct
3. Ensure JSON syntax is valid
4. Verify `cwd` points to a valid project with `.knowns/` folder
5. Restart Claude Desktop completely

### MCP server not starting

Run manually to check for errors:

```bash
knowns mcp --stdio
```

### Tools not appearing

Check if project is initialized:

```bash
cd your-project
knowns init  # if not already done
```

## Alternative: --plain Output

If you're not using Claude Desktop, use `--plain` flag for AI-readable output:

```bash
knowns task pdyd2e --plain | pbcopy  # Copy to clipboard
knowns doc "auth-pattern" --plain
```

Then paste into any AI assistant.

## Resources

- [MCP Protocol Specification](https://modelcontextprotocol.io/specification/2025-11-25)
- [Developer Guide](./developer-guide.md)
- [Claude Desktop Configuration](https://modelcontextprotocol.io/quickstart/user)
