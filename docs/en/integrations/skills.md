# Skills

Skills are reusable workflow instructions embedded in the Knowns binary and synced to platform-specific directories.

## Current skill paths

- `.claude/skills` -> Claude Code
- `.agents/skills` -> OpenCode, Codex, Antigravity
- `.kiro/skills` -> Kiro
- `.agent/skills` -> legacy/generic compatibility

## Notes

- `.agents/skills` is the primary path for agent-compatible platforms
- older projects with `.agent/skills` are still supported
- `knowns sync --skills` is the main entrypoint for regenerating skills
