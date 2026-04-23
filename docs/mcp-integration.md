# MCP Integration

Integrate Knowns with AI assistants via Model Context Protocol (MCP).

<p align="center">
  <img src="../images/mcp-integration.svg" alt="MCP Integration Architecture" width="100%">
</p>

## What is MCP?

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) is a standard for connecting AI assistants to external tools and data sources. Knowns implements an MCP server that allows AI assistants to read and manage your tasks and documentation directly.

## Supported Platforms

| Platform | Config File | Scope | Auto-discover |
|----------|-------------|-------|---------------|
| **Claude Code** | `.mcp.json` | Per-project | ✅ |
| **Kiro IDE** | `.kiro/settings/mcp.json` | Per-project | ✅ |
| **Gemini CLI** | platform-managed | Global | ✅ |
| **OpenCode** | `opencode.json` | Per-project | ✅ |
| **Codex** | `.codex/config.toml` | Per-project | ⚠️ Manual |
| **Cursor** | `.cursor/mcp.json` | Per-project | ⚠️ Manual |
| **Claude Desktop** | app config file | Global | ⚠️ Manual |

## Session Initialization (CRITICAL for Global Configs)

For platforms with **global MCP config** (for example Gemini CLI or Claude Desktop), the MCP server doesn't know which project to work with. **Run these tools at session start:**

```json
// 1. Detect available Knowns projects
mcp__knowns__project({ "action": "detect" })

// 2. Set the active project
mcp__knowns__project({ "action": "set", "projectRoot": "/path/to/project" })

// 3. Verify project is set
mcp__knowns__project({ "action": "current" })
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

> **Note:** For global MCP configs, use `project({ action: "detect" })` and `project({ action: "set" })` at session start.

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

#### Kiro IDE (Per-project)

`knowns init` automatically creates `.kiro/steering/knowns.md` (references `KNOWNS.md` via `#[[file:KNOWNS.md]]`) and `.kiro/settings/mcp.json`:

```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp", "--stdio"],
      "disabled": false,
      "autoApprove": ["*"]
    }
  }
}
```

#### OpenCode (Per-project)

`knowns init` creates `opencode.json`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "knowns": {
      "type": "local",
      "command": ["knowns", "mcp", "--stdio"],
      "enabled": true
    }
  }
}
```

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

For any global MCP client, remember that most Knowns tools return `No project set` until a project has been selected via `project({ action: "set" })`.

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

| Tool | Action | Description | Parameters |
|------|--------|-------------|------------|
| `project` | `detect` | Scan for Knowns projects | `additionalPaths?` |
| `project` | `set` | Set active project | `projectRoot` |
| `project` | `current` | Get current project | - |
| `project` | `status` | Check project readiness | - |

> **Required for global MCP configs**. Call at session start.

### Task Management

All task operations use the `tasks` tool with an `action` parameter.

| Action | Description | Required Params | Optional Params |
|--------|-------------|-----------------|-----------------|
| `create` | Create a new task | `title` | `description`, `status`, `priority`, `labels`, `assignee`, `parent`, `fulfills`, `spec` |
| `get` | Get task by ID | `taskId` | — |
| `update` | Update task fields | `taskId` | `status`, `priority`, `assignee`, `labels`, `addAc`, `checkAc`, `uncheckAc`, `removeAc`, `plan`, `notes`, `appendNotes` |
| `list` | List tasks with filters | — | `status`, `priority`, `assignee`, `label` |
| `delete` | Delete a task (dry-run by default) | `taskId` | `dryRun` |
| `history` | Get version history | `taskId` | — |
| `board` | Get kanban board state | — | — |

**`update` action extended fields:**

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

All time operations use the `time` tool with an `action` parameter.

| Action | Description | Required Params | Optional Params |
|--------|-------------|-----------------|-----------------|
| `start` | Start timer for task | `taskId` | — |
| `stop` | Stop active timer | `taskId` | — |
| `add` | Manual time entry | `taskId`, `duration` | `note`, `date` |
| `report` | Generate time report | — | `from`, `to`, `groupBy` |

### Documentation

All doc operations use the `docs` tool with an `action` parameter.

| Action | Description | Required Params | Optional Params |
|--------|-------------|-----------------|-----------------|
| `list` | List all docs | — | `tag` |
| `get` | Get doc content | `path` | `smart`, `info`, `toc`, `section` |
| `create` | Create new doc | `title` | `description`, `content`, `tags`, `folder` |
| `update` | Update doc | `path` | `title`, `description`, `content`, `appendContent`, `tags`, `section` |
| `delete` | Delete doc (dry-run by default) | `path` | `dryRun` |
| `history` | Get version history | `path` | — |

### Unified Search & Retrieval

All search operations use the `search` tool with an `action` parameter.

| Action | Description | Parameters |
|--------|-------------|------------|
| `search` | Search tasks + docs + memories | `query`, `type?` (all/task/doc/memory), `mode?` (hybrid/semantic/keyword), `status?`, `priority?`, `label?`, `tag?`, `assignee?`, `limit?` |
| `retrieve` | Ranked context with citations | `query`, `mode?`, `limit?`, `sourceTypes?`, `expandReferences?`, `status?`, `priority?`, `assignee?`, `label?`, `tag?` |
| `resolve` | Resolve semantic reference | `ref` |

**Large Document Workflow:**

```json
// Step 1: Check size
{ "action": "get", "path": "readme", "info": true }  // → estimatedTokens: 12132

// Step 2: Get TOC (if >2000 tokens)
{ "action": "get", "path": "readme", "toc": true }

// Step 3: Read/Edit specific section
{ "action": "get", "path": "readme", "section": "2" }  // Read section
// docs({ action: "update", path: "readme", section: "2", content: "..." }) replaces only that section
```

### Templates

All template operations use the `templates` tool with an `action` parameter.

| Action | Description | Required Params | Optional Params |
|--------|-------------|-----------------|-----------------|
| `list` | List all templates | — | — |
| `get` | Get template config | `name` | — |
| `run` | Run template | `name` | `variables`, `dryRun` (default: true) |
| `create` | Create new template | `name` | `description`, `doc` |

**Template Workflow:**

```json
// Step 1: List available templates
mcp__knowns__templates({ "action": "list" })

// Step 2: Get template details
mcp__knowns__templates({ "action": "get", "name": "react-component" })

// Step 3: Preview (dry run)
mcp__knowns__templates({
  "action": "run",
  "name": "react-component",
  "variables": { "name": "UserProfile", "withTest": true },
  "dryRun": true
})

// Step 4: Generate files
mcp__knowns__templates({
  "action": "run",
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
| `entity`  | Validate a specific task or doc only |
| `strict`  | Treat warnings as errors (default: `false`) |

**Example:**

```json
// Validate all refs
mcp__knowns__validate({})

// Validate only tasks
mcp__knowns__validate({ "scope": "tasks" })
```

### Memory

All persistent memory operations use the consolidated `memory` tool with an `action` parameter:

| Action | Description | Required Params | Optional Params |
|--------|-------------|-----------------|-----------------|
| `add` | Create a memory entry (project or global layer) | `content` | `title`, `layer`, `category`, `tags` |
| `get` | Get memory entry by ID | `id` | — |
| `list` | List memories with filters | — | `layer`, `category`, `tag` |
| `update` | Update memory entry | `id` | `title`, `content`, `category`, `tags`, `clear` |
| `delete` | Delete memory entry (dry-run by default) | `id` | `dryRun` |
| `promote` | Promote up one layer (project→global) | `id` | — |
| `demote` | Demote down one layer (global→project) | `id` | — |

> **Note:** To search memory entries, use `search` with `type: "memory"`.

### Working Memory (Session-Scoped)

All session-scoped memory operations use the consolidated `working_memory` tool with an `action` parameter:

| Action | Description | Required Params | Optional Params |
|--------|-------------|-----------------|-----------------|
| `add` | Add ephemeral session memory | `content` | `title`, `category`, `tags` |
| `get` | Get working memory by ID | `id` | — |
| `list` | List all session memories | — | — |
| `delete` | Delete a working memory entry | `id` | — |
| `clear` | Clear all session memories | — | — |

### Code Intelligence

All code operations use the `code` tool with an `action` parameter:

| Action | Description | Required Params | Optional Params |
|--------|-------------|-----------------|-----------------|
| `search` | Search indexed code with neighbor expansion | `query` | `limit`, `neighbors`, `edgeTypes`, `mode` |
| `symbols` | List indexed code symbols | — | `path`, `kind`, `limit` |
| `deps` | List code dependency edges | — | `type`, `limit` |
| `graph` | Return full code graph (nodes and edges) | — | — |

> **Note:** Code intelligence tools require running `knowns code ingest` first to build the code index.

---

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
