# Configuration

Knowns stores project configuration in `.knowns/config.json`.

This file describes what the project wants Knowns to manage locally, including platform integrations, semantic search settings, and generated artifact behavior.

## Example

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

## Important settings

### `name`

The project name shown in Knowns surfaces.

### `settings.gitTrackingMode`

Controls how Knowns manages Git-related generated content.

Supported values:

- `git-tracked`
- `git-ignored`
- `none`

Behavior:

- `git-tracked`: keep `.knowns/` content tracked in Git
- `git-ignored`: keep config/docs/templates tracked while leaving some local data out of Git depending on generated ignore rules
- `none`: do not let Knowns manage `.gitignore`

### `settings.semanticSearch`

Controls local semantic search.

Relevant fields:

- `enabled`
- `model`
- `huggingFaceId`
- `dimensions`
- `maxTokens`

Common behavior:

- `knowns init` can set these values
- `knowns sync` can re-apply the semantic setup
- `knowns search --reindex` rebuilds the local index

### `settings.platforms`

Declares which platform integrations Knowns should manage.

Supported values:

- `claude-code`
- `opencode`
- `codex`
- `kiro`
- `antigravity`
- `cursor`
- `gemini`
- `copilot`
- `agents`

This setting affects what `knowns init`, `knowns sync`, and `knowns update` create or refresh.

Examples of managed artifacts:

- instruction files
- skills
- MCP config
- runtime hooks
- platform-specific config files

### `settings.enableChatUI`

Controls whether the browser UI exposes the chat-oriented experience.

### `settings.autoSyncOnUpdate`

Controls whether generated artifacts should be refreshed after upgrading the CLI.

## Practical rules

### When to edit config manually

You can edit `.knowns/config.json` directly if you know what you are doing, but the normal path is:

- `knowns init` for first-time setup
- `knowns sync` to re-apply config to the current machine

### When to use `knowns sync`

Use `knowns sync` after:

- cloning a repo with existing `.knowns/`
- changing selected platforms
- updating the CLI
- wanting to restore generated artifacts to match config

### Platform-related compatibility

Current skills mapping:

- `.claude/skills` -> Claude Code
- `.agents/skills` -> OpenCode, Codex, Antigravity
- `.kiro/skills` -> Kiro
- `.agent/skills` -> legacy/generic compatibility only

If an older project already has `.agent/skills`, Knowns keeps compatibility during sync.

## Related commands

```bash
knowns init
knowns sync
knowns model list
knowns model download multilingual-e5-small
knowns search --status-check
knowns search --reindex
```
