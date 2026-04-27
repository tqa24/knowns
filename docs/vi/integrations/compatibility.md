# Tương thích

Giải thích các compatibility behaviors mà Knowns giữ lại khi platform integrations hoặc generated artifact layouts thay đổi.

## Tại sao cần

Knowns quản lý nhiều loại generated files:

- skills directories
- MCP config files
- instruction files
- runtime hooks

Khi integrations thay đổi, project cũ có thể còn layout trước đây. Knowns giữ safe compatibility thay vì break ngay.

## Skills directory

Mapping chính:

- `.claude/skills` → Claude Code
- `.agents/skills` → OpenCode, Codex, Antigravity
- `.kiro/skills` → Kiro
- `.agent/skills` → legacy/generic only

### Legacy

Project cũ có `.agent/skills` → Knowns vẫn sync.

- Project mới nên dùng `.agents/skills`
- Project cũ không bị break ngay
- `knowns sync` có thể in warning khi phát hiện legacy path

## MCP config

Knowns quản lý project-local MCP config:

- Claude Code → `.mcp.json`
- Kiro → `.kiro/settings/mcp.json`
- Cursor → `.cursor/mcp.json`
- Codex → `.codex/config.toml`
- OpenCode → `opencode.json`

Antigravity dùng global config:

- `~/.gemini/antigravity/mcp_config.json`

## Init, sync, update

### `knowns init`

Tạo platform artifacts cho project mới.

### `knowns sync`

Re-apply `.knowns/config.json` lên máy hiện tại.

Dùng sau khi:

- clone repo
- đổi platforms
- muốn generated files khớp lại với config

### `knowns update`

Update CLI, rồi refresh generated artifacts phụ thuộc vào binary hoặc config policy.

## Khuyến nghị

- Project mới → follow layout chính hiện tại
- Project cũ → để `knowns sync` và `knowns update` giữ tương thích trước, migrate có chủ đích sau
