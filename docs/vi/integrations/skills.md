# Skills

Skills là reusable workflow instructions, embedded trong Knowns binary và sync ra platform-specific directories.

## Skill paths

- `.claude/skills` → Claude Code
- `.agents/skills` → OpenCode, Codex, Antigravity
- `.kiro/skills` → Kiro
- `.agent/skills` → legacy/generic compatibility

## Ghi chú

- `.agents/skills` là primary path cho agent-compatible platforms
- Project cũ có `.agent/skills` vẫn được hỗ trợ
- `knowns sync --skills` là entrypoint chính để regenerate skills
