# MCP

Knowns expose MCP server để AI assistants truy cập trực tiếp task, doc, memory, search, validation, và code tools.

## Server command

```bash
knowns mcp --stdio
```

## Platform support

| Platform | Config file | Scope | Auto setup |
|---|---|---|---|
| Claude Code | `.mcp.json` | per-project | yes |
| Kiro | `.kiro/settings/mcp.json` | per-project | yes |
| OpenCode | `opencode.json` | per-project | yes |
| Codex | `.codex/config.toml` | per-project | yes |
| Cursor | `.cursor/mcp.json` | per-project | yes |
| Antigravity | `~/.gemini/antigravity/mcp_config.json` | global | yes |
| Claude Desktop | app config | global | manual |

## Config ví dụ

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

## Lưu ý với global MCP clients

Với config MCP global, server có thể không biết project nào cần dùng lúc bắt đầu session.

Set active project trước:

```json
{ "action": "detect" }
{ "action": "set", "projectRoot": "/path/to/project" }
{ "action": "current" }
```

## Tại sao MCP hữu ích

- AI truy cập project state có cấu trúc
- Ít phải parse shell output
- Validation và retrieval workflows dễ hơn
