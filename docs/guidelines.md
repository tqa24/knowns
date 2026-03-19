# Guidelines System

Knowns ships AI instruction files for supported platforms and exposes the full guidance through the CLI.

---

## Overview

There are two layers of guidance:

| Layer | Where it lives | Purpose |
| ----- | -------------- | ------- |
| Core instructions | `KNOWNS.md`, `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, `OPENCODE.md`, `.github/copilot-instructions.md` | Short, always-on agent rules |
| Full reference | `knowns guidelines --plain` | Full usage guidance rendered by the CLI |

The CLI currently exposes `knowns guidelines` as a single command. It does not support section subcommands or extra filtering flags.

---

## View Guidelines

Use the CLI to print the complete guidance:

```bash
knowns guidelines --plain
```

This is the canonical reference when you want the latest rendered guidance from the current binary.

---

## Sync Instruction Files

Use `knowns sync` to sync built-in skills and instruction files:

```bash
# Sync skills and instruction files
knowns sync

# Sync skills only
knowns sync --skills

# Sync instruction files only
knowns sync --instructions

# Sync a specific platform
knowns sync --instructions --platform claude

# Overwrite existing files
knowns sync --force
```

Supported instruction targets:

- `CLAUDE.md`
- `OPENCODE.md`
- `GEMINI.md`
- `AGENTS.md`
- `.github/copilot-instructions.md`

For backwards compatibility, `knowns agents --sync` also exists for instruction-file generation only.

---

## Markers

Generated guidance inside instruction files is wrapped in markers:

```markdown
<!-- KNOWNS GUIDELINES START -->
# Knowns Guidelines
...
<!-- KNOWNS GUIDELINES END -->
```

When markers already exist, Knowns replaces only the content inside them and preserves the rest of the file.

---

## Practical Notes

- Prefer `knowns guidelines --plain` when you want the exact text agents should follow.
- Prefer `knowns sync` for normal upkeep; use `knowns agents --sync` only when you specifically want instruction files.
- If docs and generated instruction files disagree, trust the current CLI output first.

---

## Related

- [Command Reference](./commands.md) - CLI syntax
- [MCP Integration](./mcp-integration.md) - MCP setup and usage
- [Multi-Platform](./multi-platform.md) - Platform-specific instruction files
