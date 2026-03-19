# Knowns Developer Guide

Technical notes for contributors working on the current Go codebase.

---

## Tech Stack

| Layer | Technology |
| ----- | ---------- |
| Runtime | Go |
| CLI | Cobra |
| Storage | File-based `.knowns/` data |
| Web server | Go HTTP server |
| Web UI | React app embedded into the binary |
| AI integration | MCP server + generated instruction files |
| Search | Keyword + optional semantic search |

---

## Codebase Map

Main repo structure:

```text
cmd/knowns/          CLI entrypoint
internal/cli/        Cobra commands
internal/models/     Domain models and config structs
internal/storage/    Persistence for tasks, docs, versions, config
internal/server/     HTTP server, routes, browser backend
internal/mcp/        MCP server implementation
internal/search/     Search and semantic-search support
ui/                  React UI source and embedded assets
tests/               End-to-end and integration coverage
```

---

## Important Runtime Behavior

### Browser UI

- `knowns browser` starts the local web server
- default port is `3001` unless overridden by `settings.serverPort`
- the browser only auto-opens when `--open` is passed

### Sync

- `knowns sync` handles instruction-file and skill syncing
- `knowns agents --sync` remains as a compatibility path for instruction files
- platform filtering is done with `--platform` or `settings.platforms`

### Search

- `knowns search <query>` performs search
- `knowns search --reindex` rebuilds the index
- `knowns search --status-check` shows semantic-search status

---

## Config Model

Project config is defined in `internal/models/config.go`.

Important fields include:

- `name`, `id`, `createdAt`
- `settings.defaultAssignee`, `settings.defaultPriority`, `settings.defaultLabels`
- `settings.statuses`, `settings.statusColors`, `settings.visibleColumns`
- `settings.semanticSearch`
- `settings.serverPort`
- `settings.platforms`
- `settings.autoSyncOnUpdate`
- `settings.enableChatUI`
- `settings.opencodeServer`, `settings.opencodeModels`

---

## CLI Conventions

- Root commands are registered under `internal/cli/`
- command help is the source of truth for public syntax
- shorthand behavior exists for some groups, for example `knowns task <id>` and `knowns doc <path>`
- prefer updating docs from actual `knowns ... --help` output to avoid drift

---

## Development Commands

```bash
make build
make dev
make test
make test-e2e
make lint
make ui
```

Use `go run ./cmd/knowns --help` and subcommand help while updating docs or CLI behavior.

---

## Documentation Rule

This repo has accumulated older Node/TypeScript-era docs. If code and docs disagree, trust the Go implementation and CLI help first, then update docs to match.

---

## Related

- [Architecture](../ARCHITECTURE.md) - Higher-level system overview
- [Configuration](./configuration.md) - Runtime config fields
- [Command Reference](./commands.md) - Current CLI surface
