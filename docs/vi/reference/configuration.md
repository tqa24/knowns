# Cấu hình

Knowns lưu project config trong `.knowns/config.json`.

File này khai báo những gì Knowns cần quản lý locally: platform integrations, semantic search, generated artifacts.

## Ví dụ

```json
{
  "name": "my-project",
  "settings": {
    "gitTrackingMode": "git-tracked",
    "semanticSearch": {
      "enabled": true,
      "model": "multilingual-e5-small",
      "huggingFaceId": "Xenova/multilingual-e5-small",
      "dimensions": 384,
      "maxTokens": 512
    },
    "platforms": [
      "claude-code",
      "opencode",
      "codex",
      "kiro",
      "antigravity",
      "cursor",
      "gemini",
      "copilot",
      "agents"
    ],
    "enableChatUI": true,
    "autoSyncOnUpdate": true
  }
}
```

## Các setting quan trọng

### `name`

Tên project hiển thị trong Knowns.

### `settings.gitTrackingMode`

- `git-tracked` — `.knowns/` content tracked trong Git
- `git-ignored` — config/docs/templates tracked, local data thì không
- `none` — Knowns không quản lý `.gitignore`

### `settings.semanticSearch`

Config cho local semantic search: `enabled`, `model`, `huggingFaceId`, `dimensions`, `maxTokens`.

- `knowns init` set các giá trị này
- `knowns sync` re-apply semantic setup
- `knowns search --reindex` rebuild local index

### `settings.platforms`

Khai báo platform integrations cần quản lý.

Supported: `claude-code`, `opencode`, `codex`, `kiro`, `antigravity`, `cursor`, `gemini`, `copilot`, `agents`.

Ảnh hưởng tới những gì `init`, `sync`, `update` tạo hoặc refresh: instruction files, skills, MCP config, runtime hooks, platform-specific config.

### `settings.enableChatUI`

Bật/tắt chat experience trong Web UI.

### `settings.autoSyncOnUpdate`

Tự động refresh generated artifacts sau khi upgrade CLI.

## Khi nào edit config trực tiếp?

Có thể edit `.knowns/config.json` trực tiếp, nhưng flow thường là:

- `knowns init` cho lần đầu
- `knowns sync` để re-apply config

## Skills mapping

- `.claude/skills` → Claude Code
- `.agents/skills` → OpenCode, Codex, Antigravity
- `.kiro/skills` → Kiro
- `.agent/skills` → legacy/generic compatibility

Project cũ có `.agent/skills` vẫn được giữ tương thích khi sync.

## Lệnh liên quan

```bash
knowns init
knowns sync
knowns model list
knowns model download multilingual-e5-small
knowns search --status-check
knowns search --reindex
```
