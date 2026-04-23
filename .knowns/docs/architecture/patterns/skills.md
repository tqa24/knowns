---
title: Claude Code Skills
description: Pattern for creating and managing Claude Code skills in Knowns CLI
createdAt: '2026-01-17T06:06:37.006Z'
updatedAt: '2026-04-02T09:38:30.160Z'
tags:
  - pattern
  - claude-code
  - skills
---

## Overview

Knowns CLI integrates with Claude Code skills - workflow templates that can be invoked via `/kn-<skill>` commands.

## Skill Structure

```
internal/instructions/skills/
├── kn-init/
│   └── SKILL.md               # Skill content with frontmatter
├── kn-plan/
│   └── SKILL.md
├── kn-implement/
│   └── SKILL.md
└── ...
```

## Naming Convention

- **Folder name**: `kn-<skill>`
- **Examples**: `kn-init`, `kn-plan`, `kn-implement`
- **Colon notation** creates clear namespace in Claude Code UI

## SKILL.md Format

```yaml
---
name: kn-init
description: Initialize session with project context
---

# Instructions

...skill content...
```

## Available Skills (13 total)

| Skill | Description |
|-------|-------------|
| `kn-init` | Initialize session (read docs, list tasks, load critical learnings, load project memories) |
| `kn-spec` | Create spec with Socratic exploring phase (SDD workflow). Searches memories for past decisions. |
| `kn-plan` | Plan task implementation (with pre-execution validation). Searches memories for patterns. |
| `kn-implement` | Implement task (includes reopen logic). Saves quick insights to memory. |
| `kn-research` | Research codebase before implementation. Searches docs and memories via unified search. |
| `kn-review` | Multi-perspective code review with P1/P2/P3 severity. Searches memories for conventions. |
| `kn-commit` | Generate commit message and commit changes |
| `kn-extract` | Extract patterns, decisions, failures + save to memory + consolidation mode |
| `kn-doc` | Create and update documentation |
| `kn-template` | Generate code from templates |
| `kn-verify` | SDD verification and coverage reporting |
| `kn-go` | Full pipeline execution from approved spec (no review gates). Searches memories per task. |
| `kn-debug` | Structured debugging: triage → reproduce → fix → learn. Searches and saves to memory. |

## Memory Integration

All skills participate in the memory loop:
- **Read skills** (init, research, plan, spec, go, review, debug): search `type: "memory"` via unified search
- **Write skills** (implement, extract, debug): save patterns/decisions/failures with `memory({ action: "add", layer: "project" })`
- The global rule in KNOWNS.md `## Memory Usage` encourages all skills to save knowledge as it emerges

## Workflow

### Manual flow (step by step)
```
/kn-init → /kn-research → /kn-spec → /kn-plan --from → /kn-plan <id> → /kn-implement <id> → /kn-review → /kn-commit → /kn-extract
```

### Go mode (full pipeline from spec)
```
/kn-spec → approve → /kn-go specs/<name>
```

### Debugging flow
```
/kn-debug → /kn-review → /kn-commit
```

## Sync Commands

```bash
knowns sync                   # Sync all (skills + agents + model + index + MCP)
knowns sync --skills          # Sync skills only
knowns sync --instructions    # Sync instruction files only
```

Sync always overwrites generated files. The `--force` flag is deprecated.

## Implementation Details

### Source Location

Skills source lives in `internal/instructions/skills/kn-*/SKILL.md`. These are embedded into the Go binary and synced to `.claude/skills/` (and other platform dirs) during `knowns sync`.

### Sync Function

Skills are synced to `.claude/skills/<folder-name>/SKILL.md` in the project.
