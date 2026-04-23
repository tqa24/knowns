---
title: AI Platforms
createdAt: '2026-01-23T04:07:55.100Z'
updatedAt: '2026-03-12T17:59:05.289Z'
description: 'Supported AI platforms: Claude Code, OpenCode. Reference configs for others.'
tags:
  - feature
  - ai
  - platforms
---
## Overview

Knowns officially supports **Claude Code** and **OpenCode**. Other platforms can use Knowns via MCP — configs below are for reference only.

**Related:** @doc/ai/mcp, @doc/ai/skills

---

## Supported Platforms

### Claude Code

| Item | Value |
|------|-------|
| **MCP Config** | `.mcp.json` (project-level, auto-created by `knowns init`) |
| **Instructions** | `CLAUDE.md` (auto-created by `knowns init`) |
| **Skills** | `.claude/skills/` (auto-synced) |

```
.claude/
├── CLAUDE.md              # Instructions
├── settings.json
└── skills/
    └── kn-*/SKILL.md      # Synced from .knowns/skills/
```

MCP config (`.mcp.json`):
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

> Auto-configured by `knowns init`. No manual setup needed.

---

### OpenCode

| Item | Value |
|------|-------|
| **MCP Config** | `opencode.json` (project-level) or `~/.config/opencode/opencode.json` (global) |
| **Instructions** | `OPENCODE.md` or project rules |
| **Skills** | N/A |

MCP config (`opencode.json` in project root):
```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "knowns": {
      "type": "local",
      "command": ["knowns", "mcp"],
      "enabled": true
    }
  }
}
```

> **Note**: OpenCode uses global MCP config. Run `mcp__knowns__project({ action: "detect" })` and `mcp__knowns__project({ action: "set" })` at session start.

#### OpenCode + Antigravity Auth (Community)

Use [opencode-antigravity-auth](https://github.com/NoeFabris/opencode-antigravity-auth) to access Gemini/Claude models via Google OAuth:

```bash
npm i -g opencode-antigravity-auth
opencode auth login  # Choose Google -> OAuth with Google (Antigravity)
```

> **Warning**: Google may block accounts using this plugin. Use at your own risk.

---
## Comparison

| | Claude Code | OpenCode |
|--|-------------|----------|
| **MCP** | `.mcp.json` (project) | `opencode.json` (project/global) |
| **Instructions** | `CLAUDE.md` | `OPENCODE.md` |
| **Skills** | `.claude/skills/` | N/A |
| **Auto-setup** | `knowns init` | `knowns init` |
| **Project scope** | Per-project | Needs `set_project` |

---

## Other Platforms (Reference Only)

These configs are **not officially supported** but may work with Knowns MCP.

### Gemini CLI

```json
// ~/.gemini/settings.json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```

### Antigravity (Google IDE)

```json
// ~/.gemini/antigravity/mcp_config.json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```

### Cursor

```json
// .cursor/mcp.json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```

### Continue

```json
// .continue/config.json
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

### Cline

```json
// .cline/mcp.json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"]
    }
  }
}
```
