# Cấu hình

Knowns lưu cấu hình của dự án trong `.knowns/config.json`.

## Các thiết lập quan trọng

- `settings.gitTrackingMode`
- `settings.semanticSearch`
- `settings.platforms`
- `settings.enableChatUI`
- `settings.autoSyncOnUpdate`

## Ví dụ

```json
{
  "name": "my-project",
  "settings": {
    "gitTrackingMode": "git-tracked",
    "semanticSearch": {
      "enabled": true,
      "model": "multilingual-e5-small"
    },
    "platforms": ["claude-code", "opencode", "codex", "kiro", "antigravity", "cursor", "gemini", "copilot", "agents"],
    "enableChatUI": true,
    "autoSyncOnUpdate": true
  }
}
```

## Các platform ID được hỗ trợ

- `claude-code`
- `opencode`
- `codex`
- `kiro`
- `antigravity`
- `cursor`
- `gemini`
- `copilot`
- `agents`

## Ghi chú

- `knowns init` tạo cấu hình ban đầu
- `knowns sync` áp lại cấu hình vào các tệp được tạo và thiết lập cục bộ
