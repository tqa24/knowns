# Compatibility

This document explains the main compatibility behaviors Knowns preserves when platform integrations or generated artifact layouts change.

## Why this exists

Knowns manages generated files such as:

- skills directories
- MCP configuration files
- instruction files
- runtime hooks

As integrations evolve, older projects may still contain previously generated layouts. Knowns tries to preserve safe compatibility instead of breaking those projects immediately.

## Skills directory compatibility

Current primary mapping:

- `.claude/skills` -> Claude Code
- `.agents/skills` -> OpenCode, Codex, Antigravity
- `.kiro/skills` -> Kiro
- `.agent/skills` -> legacy/generic compatibility only

### Legacy behavior

If an older project already contains `.agent/skills`, Knowns continues syncing it for compatibility.

This means:

- new projects should prefer `.agents/skills` for agent-compatible platforms
- old projects are not forced to break immediately
- `knowns sync` may print a warning when it detects the legacy path

## Platform-specific MCP compatibility

Knowns now manages project-local MCP config for several platforms, for example:

- Claude Code -> `.mcp.json`
- Kiro -> `.kiro/settings/mcp.json`
- Cursor -> `.cursor/mcp.json`
- Codex -> `.codex/config.toml`
- OpenCode -> `opencode.json`

For Antigravity, the MCP config is global:

- `~/.gemini/antigravity/mcp_config.json`

## Init, sync, and update

### `knowns init`

Creates the selected platform artifacts for a project from scratch.

### `knowns sync`

Re-applies `.knowns/config.json` to the current machine.

Use it after:

- cloning a repository
- changing selected platforms
- wanting generated files to match the current config again

### `knowns update`

Updates the CLI, then refreshes generated artifacts that depend on the binary or config policy.

## Recommendation

- For new projects, follow the current primary layout.
- For older projects, let `knowns sync` and `knowns update` preserve compatibility first, then migrate deliberately when needed.
