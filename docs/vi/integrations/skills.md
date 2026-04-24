# Skills

Skills là các hướng dẫn workflow có thể tái sử dụng, được nhúng trong binary của Knowns và sync ra các thư mục riêng cho từng nền tảng.

## Các skill path hiện tại

- `.claude/skills` -> Claude Code
- `.agents/skills` -> OpenCode, Codex, Antigravity
- `.kiro/skills` -> Kiro
- `.agent/skills` -> legacy/generic compatibility

## Ghi chú

- `.agents/skills` là path chính cho các nền tảng agent-compatible
- project cũ với `.agent/skills` vẫn được hỗ trợ
- `knowns sync --skills` là entrypoint chính để tạo lại skills
