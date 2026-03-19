---
title: MCP Configuration
createdAt: '2026-01-23T04:07:55.764Z'
updatedAt: '2026-03-12T17:59:05.274Z'
description: 'MCP server setup — Claude Code and OpenCode supported, others for reference'
tags:
  - feature
  - ai
  - mcp
---
## Overview

MCP (Model Context Protocol) allows AI to call Knowns functions directly.

**Related:** @doc/ai/platforms

---

## Session Initialization (CRITICAL)

**CRITICAL: At the START of every MCP session, run these tools to set the project context:**

```json
// 1. Detect available projects
mcp__knowns__detect_projects({})

// 2. Set the active project
mcp__knowns__set_project({ "projectRoot": "/path/to/project" })

// 3. Verify project is set
mcp__knowns__get_current_project({})
```

**Why?** MCP servers (especially global configs like Antigravity) don't know which project to work with. Without setting the project, operations will fail or affect the wrong directory.

---

## Support Matrix

| Platform | Config File | Auto-setup | Project Scope | Status |
|----------|-------------|------------|---------------|--------|
| **Claude Code** | `.mcp.json` (project) | ✅ `knowns init` | Per-project | **Supported** |
| **OpenCode** | `opencode.json` (project/global) | ✅ `knowns init` | Needs `set_project` | **Supported** |
| Gemini CLI | `~/.gemini/settings.json` (global) | — | Needs `set_project` | Reference |
| Antigravity | `~/.gemini/antigravity/mcp_config.json` (global) | — | Needs `set_project` | Reference |
| Cursor | `.cursor/mcp.json` (project) | — | Per-project | Reference |
| Cline | `.cline/mcp.json` (project) | — | Per-project | Reference |
| Continue | `.continue/config.json` (project) | — | Per-project | Reference |

See @doc/ai/platforms for full config examples.
## Knowns MCP Server

Knowns is a compiled Go binary. The MCP server is started with the `mcp` subcommand.

**If installed via npm, go install, or direct download:**
```json
{
  "command": "knowns",
  "args": ["mcp"]
}
```

**If using npx (no global install):**
```json
{
  "command": "npx",
  "args": ["-y", "knowns", "mcp"]
}
```

---
## Platform Configs

> **Note**: All examples below use `knowns mcp` directly (assumes `knowns` is installed globally via `npm install -g knowns`, `go install`, or direct binary download). Replace with `npx -y knowns mcp` if using npx without global install.

### Claude Code: `.mcp.json` (Project-level)
```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```

### Antigravity: `~/.gemini/antigravity/mcp_config.json` (Global)
```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```

> **Note**: Antigravity uses global config. Use `mcp__knowns__detect_projects` and `mcp__knowns__set_project` at session start to set the correct project.

### Gemini CLI: `~/.gemini/settings.json` (Global)
```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```

### Cursor: `.cursor/mcp.json`
```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```

### Continue: `.continue/config.json`
```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "knowns",
        "transport": {
          "type": "stdio",
          "command": "knowns",
          "args": ["mcp"]
        }
      }
    ]
  }
}
```

---
## CLI Commands

```bash
# Auto-generate MCP config
knowns mcp setup
```

---

## Available MCP Tools

### Project Tools (Session Init)
| Tool | Description |
|------|-------------|
| `mcp__knowns__detect_projects` | Scan for Knowns projects |
| `mcp__knowns__set_project` | Set active project |
| `mcp__knowns__get_current_project` | Get current project status |

### Tasks
| Tool | Description |
|------|-------------|
| `mcp__knowns__list_tasks` | List tasks |
| `mcp__knowns__get_task` | Get task |
| `mcp__knowns__create_task` | Create task |
| `mcp__knowns__update_task` | Update task (status, AC, plan, notes) |

### Docs
| Tool | Description |
|------|-------------|
| `mcp__knowns__list_docs` | List docs |
| `mcp__knowns__get_doc` | Get doc (with smart mode) |
| `mcp__knowns__create_doc` | Create doc |
| `mcp__knowns__update_doc` | Update doc |

### Search
| Tool | Description |
|------|-------------|
| `mcp__knowns__search` | Unified search (tasks + docs) with semantic support |

### Time
| Tool | Description |
|------|-------------|
| `mcp__knowns__start_time` | Start timer |
| `mcp__knowns__stop_time` | Stop timer |
| `mcp__knowns__add_time` | Add manual time entry |
| `mcp__knowns__get_time_report` | Get time report |

### Templates
| Tool | Description |
|------|-------------|
| `mcp__knowns__list_templates` | List templates |
| `mcp__knowns__get_template` | Get template config |
| `mcp__knowns__run_template` | Run template |
| `mcp__knowns__create_template` | Create template scaffold |

### Validation
| Tool | Description |
|------|-------------|
| `mcp__knowns__validate` | Validate tasks, docs, templates for broken refs and quality |

### Other
| Tool | Description |
|------|-------------|
| `mcp__knowns__search` | Unified search (tasks + docs) |
| `mcp__knowns__get_board` | Get kanban board |
## MCP vs CLI

| Aspect | MCP | CLI |
|--------|-----|-----|
| Speed | Faster | Slower |
| Output | JSON | Text |
| Complex ops | ✅ Full support | ✅ Full support |

**Recommendation:** Use MCP for all operations when available.

---

## Full Feature Parity

| Feature | CLI | MCP |
|---------|-----|-----|
| List tasks | ✅ | ✅ |
| Get task | ✅ | ✅ |
| Create task | ✅ | ✅ |
| Update status | ✅ | ✅ |
| **Add AC** | ✅ | ✅ |
| **Check AC** | ✅ | ✅ |
| **Set plan** | ✅ | ✅ |
| **Set notes** | ✅ | ✅ |
| **Append notes** | ✅ | ✅ |
| Time tracking | ✅ | ✅ |
| List docs | ✅ | ✅ |
| Get doc | ✅ | ✅ |
| Create doc | ✅ | ✅ |
| Update doc | ✅ | ✅ |
| Search | ✅ | ✅ |
| Templates | ✅ | ✅ |
| **Validate** | ✅ | ✅ |
| **Project detection** | N/A | ✅ |
## Example: Full Task Workflow via MCP

```json
// 0. Session init (required for global MCP configs)
mcp__knowns__detect_projects({})
mcp__knowns__set_project({ "projectRoot": "/path/to/project" })

// 1. Get task
mcp__knowns__get_task({ "taskId": "abc123" })

// 2. Take task
mcp__knowns__update_task({
  "taskId": "abc123",
  "status": "in-progress",
  "assignee": "@me"
})

// 3. Start timer
mcp__knowns__start_time({ "taskId": "abc123" })

// 4. Add acceptance criteria
mcp__knowns__update_task({
  "taskId": "abc123",
  "addAc": ["User can login"]
})

// 5. Set implementation plan
mcp__knowns__update_task({
  "taskId": "abc123",
  "plan": "1. Research
2. Implement
3. Test"
})

// 6. Check AC after completing
mcp__knowns__update_task({
  "taskId": "abc123",
  "checkAc": [1]
})

// 7. Append progress notes
mcp__knowns__update_task({
  "taskId": "abc123",
  "appendNotes": "✓ Completed: login feature"
})

// 8. Stop timer
mcp__knowns__stop_time({ "taskId": "abc123" })

// 9. Mark done
mcp__knowns__update_task({
  "taskId": "abc123",
  "status": "done"
})
```



---

## Example: Validate Before Planning

```json
// Check for broken refs before starting work
mcp__knowns__validate({})

// Returns:
{
  "success": true,
  "valid": true,  // or false if errors
  "stats": { "tasks": 48, "docs": 38, "templates": 2 },
  "summary": { "errors": 0, "warnings": 2, "info": 5 },
  "issues": [...]
}

// Validate specific type only
mcp__knowns__validate({ "type": "task" })

// Strict mode (warnings → errors)
mcp__knowns__validate({ "strict": true })

// Auto-fix broken refs
mcp__knowns__validate({ "fix": true })
```

**Use cases:**
- Before planning: check if refs in task description are valid
- After editing docs: verify no broken links introduced
- CI/CD integration: fail build if validation errors exist
