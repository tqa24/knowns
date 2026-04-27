# Auto sync

Knowns dùng `knowns sync` và `knowns update` để giữ generated artifacts đồng bộ với binary và project config.

## Sync được gì

- instruction files
- skills
- MCP config
- platform-specific config
- git integration
- semantic setup và indexing

## Lệnh

```bash
knowns sync
knowns sync --skills
knowns sync --instructions
knowns update
```

## Legacy

Project cũ vẫn dùng `.agent/skills` → Knowns tiếp tục sync để giữ tương thích, có thể in warning.
