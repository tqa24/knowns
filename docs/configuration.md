# Configuration

Knowns stores project settings in `.knowns/config.json`.

---

## Project Config Shape

Current Go CLI config is centered around a root project object:

```json
{
  "name": "my-project",
  "id": "proj_123",
  "createdAt": "2026-03-19T10:00:00Z",
  "settings": {
    "defaultAssignee": "@me",
    "defaultPriority": "medium",
    "defaultLabels": ["backend"],
    "timeFormat": "24h",
    "gitTrackingMode": "git-tracked",
    "statuses": ["todo", "in-progress", "blocked", "done", "in-review"],
    "statusColors": {
      "todo": "gray",
      "in-progress": "blue",
      "blocked": "red",
      "done": "green"
    },
    "visibleColumns": ["todo", "in-progress", "blocked", "done", "in-review"],
    "semanticSearch": {
      "enabled": true,
      "model": "gte-small",
      "huggingFaceId": "Xenova/gte-small",
      "dimensions": 384,
      "maxTokens": 512
    },
    "serverPort": 3001,
    "platforms": ["claude-code", "opencode", "gemini", "copilot", "agents"]
  }
}
```

---

## Common Settings

| Key | Type | Description |
| --- | ---- | ----------- |
| `name` | string | Project name |
| `id` | string | Project identifier |
| `createdAt` | string | ISO timestamp |
| `settings.defaultAssignee` | string | Default assignee for new tasks |
| `settings.defaultPriority` | string | Default task priority |
| `settings.defaultLabels` | string[] | Default labels for new tasks |
| `settings.timeFormat` | string | `12h` or `24h` |
| `settings.gitTrackingMode` | string | `git-tracked`, `git-ignored`, or `none` |
| `settings.statuses` | string[] | Allowed task statuses |
| `settings.statusColors` | object | Board/status color mapping |
| `settings.visibleColumns` | string[] | Columns shown in the board |
| `settings.semanticSearch` | object | Semantic search settings |
| `settings.serverPort` | number | Browser server port override |
| `settings.platforms` | string[] | Enabled platform targets |
| `settings.autoSyncOnUpdate` | boolean | Auto-sync generated files after upgrade |
| `settings.enableChatUI` | boolean | Show/hide Chat UI in browser |
| `settings.opencodeServer` | object | OpenCode server connection settings |
| `settings.opencodeModels` | object | Project-level OpenCode model preferences |

---

## Semantic Search Settings

`settings.semanticSearch` supports:

| Key | Type | Description |
| --- | ---- | ----------- |
| `enabled` | boolean | Enable semantic search |
| `model` | string | Model ID such as `gte-small` |
| `huggingFaceId` | string | HuggingFace identifier for the model |
| `dimensions` | number | Embedding vector size |
| `maxTokens` | number | Max input tokens |

Useful commands:

```bash
knowns model list
knowns model download gte-small
knowns model set gte-small
knowns search --reindex
knowns search --status-check
```

---

## Platform Settings

`settings.platforms` can restrict instruction-file generation to a subset of platforms:

```json
{
  "settings": {
    "platforms": ["claude-code", "copilot", "agents"]
  }
}
```

Supported values in config:

- `claude-code`
- `opencode`
- `gemini`
- `copilot`
- `agents`

---

## Browser Server Port

If `settings.serverPort` is set, `knowns browser` uses it as the default port. Otherwise the CLI falls back to `3001`.

```bash
knowns browser
knowns browser --open
knowns browser --port 3002
```

---

## CLI Commands

Current config commands:

```bash
knowns config get <key>
knowns config set <key> <value>
knowns config list
knowns config reset
```

Examples:

```bash
knowns config get settings.semanticSearch
knowns config set settings.semanticSearch.enabled true
knowns config set settings.serverPort 3002
knowns config list
```

---

## Related

- [Semantic Search](./semantic-search.md) - Search-specific setup
- [Multi-Platform](./multi-platform.md) - Platform targets and sync
- [Command Reference](./commands.md) - Current CLI syntax
