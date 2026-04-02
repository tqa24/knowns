---
title: Sync Command
description: knowns sync — apply config.json to set up skills, instructions, model, git, and search index
createdAt: '2026-03-27T07:10:42.070Z'
updatedAt: '2026-04-02T09:38:29.325Z'
tags:
  - feature
  - cli
  - sync
  - setup
---

## Overview

`knowns sync` reads `.knowns/config.json` and applies all settings locally. This is the recommended command after cloning a repo that uses Knowns.

## Use Case

```bash
git clone <repo>
knowns sync        # sets up everything from config.json
```

## What It Does

1. **Skills** — copies built-in skills to platform directories (`.claude/skills/`, `.agent/skills/`, `.kiro/skills/`)
2. **Instructions** — generates agent instruction files (KNOWNS.md, CLAUDE.md, AGENTS.md, etc.) for configured platforms. Always overwrites to stay in sync with templates.
3. **Git integration** — applies `.gitignore` rules based on `gitTrackingMode` setting
4. **Model download** — downloads the configured embedding model if not installed locally
5. **Import sync** — syncs git-based imports
6. **Search index** — rebuilds the semantic search index
7. **MCP configs** — syncs MCP config files to use the local binary

## Flags

| Flag | Description |
|------|-------------|
| `--force` | Deprecated — sync always overwrites generated files |
| `--skills` | Sync skills only |
| `--instructions` | Sync instruction files only |
| `--model` | Download embedding model only |
| `--platform <name>` | Sync specific platform (claude, gemini, copilot, agents) |

## Auto-Setup Warning

When running any `knowns` command in a project where semantic search is configured but the model is not installed, a warning is shown:

```
⚠ This project uses semantic search but the embedding model is not installed locally.
  Model: GTE Small (gte-small, ~67MB)

  Run: knowns sync
```

## Config Fields Used

| Config field | Sync action |
|---|---|
| `settings.platforms` | Which platform dirs to sync skills/instructions to |
| `settings.gitTrackingMode` | Which `.gitignore` rules to apply |
| `settings.semanticSearch.model` | Which embedding model to download |
| `settings.semanticSearch.enabled` | Whether to download model and reindex |
| `settings.autoSyncOnUpdate` | Whether to auto-sync skills on CLI version change |

## Git Tracking Modes

| Mode | Behavior |
|------|----------|
| `git-tracked` | All `.knowns/` content is committed. No gitignore rules added. |
| `git-ignored` | Only `config.json`, `docs/`, and `templates/` are committed. Tasks stay local. |
| `none` | No gitignore changes. User manages manually. |

## Related

- `knowns init` — interactive wizard for first-time setup
- `knowns init --force` — re-run wizard with existing config as defaults
- `knowns search --reindex` — rebuild search index only
- `knowns model download <id>` — download a specific model
