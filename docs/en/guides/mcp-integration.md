# MCP Integration

Knowns exposes an MCP server so AI assistants can access tasks, docs, memory, search, validation, and code tools directly.

## Server command

```bash
knowns mcp --stdio
```

## Current platform support

| Platform | Config file | Scope | Auto setup |
|---|---|---|---|
| Claude Code | `.mcp.json` | per-project | yes |
| Kiro | `.kiro/settings/mcp.json` | per-project | yes |
| OpenCode | `opencode.json` | per-project | yes |
| Codex | `.codex/config.toml` | per-project | yes |
| Cursor | `.cursor/mcp.json` | per-project | yes |
| Antigravity | `~/.gemini/antigravity/mcp_config.json` | global | yes |
| Claude Desktop | app config | global | manual |

## Typical config examples

### Claude Code

```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp", "--stdio"]
    }
  }
}
```

### Cursor

```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp", "--stdio"]
    }
  }
}
```

### Codex

```toml
[mcp_servers.knowns]
command = "knowns"
args = ["mcp", "--stdio"]
```

### OpenCode

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

## Important note for global MCP clients

For global MCP configs, the server may not know which project to use at session start.

Set the active project first:

```json
{ "action": "detect" }
{ "action": "set", "projectRoot": "/path/to/project" }
{ "action": "current" }
```

## Why MCP is useful

- structured AI access to project state
- less shell parsing and less prompt copy-paste
- easier validation and retrieval workflows for AI assistants
