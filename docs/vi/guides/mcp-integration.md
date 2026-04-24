# Tích hợp MCP

Knowns cung cấp MCP server để AI assistants có thể truy cập trực tiếp task, tài liệu, memory, search, validation, và code tools.

## Lệnh server

```bash
knowns mcp --stdio
```

## Các nền tảng hiện đang hỗ trợ

| Nền tảng | Tệp cấu hình | Phạm vi | Tự động thiết lập |
|---|---|---|---|
| Claude Code | `.mcp.json` | per-project | yes |
| Kiro | `.kiro/settings/mcp.json` | per-project | yes |
| OpenCode | `opencode.json` | per-project | yes |
| Codex | `.codex/config.toml` | per-project | yes |
| Cursor | `.cursor/mcp.json` | per-project | yes |
| Antigravity | `~/.gemini/antigravity/mcp_config.json` | global | yes |
| Claude Desktop | app config | global | manual |

## Ví dụ cấu hình điển hình

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

## Lưu ý quan trọng với các client MCP dùng cấu hình toàn cục

Với các cấu hình MCP toàn cục, server có thể không biết dự án nào cần dùng ngay từ đầu phiên làm việc.

Hãy chọn dự án đang làm trước:

```json
{ "action": "detect" }
{ "action": "set", "projectRoot": "/path/to/project" }
{ "action": "current" }
```

## Vì sao MCP hữu ích?

- AI truy cập trạng thái của dự án theo dạng có cấu trúc
- ít phải phân tích shell output hoặc copy-paste prompt
- dễ làm các luồng validation và retrieval hơn
