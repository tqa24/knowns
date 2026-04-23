---
title: Skills System
description: Skill definition, sharing, and sync across platforms
createdAt: '2026-01-23T04:07:56.363Z'
updatedAt: '2026-04-23T11:23:34.829Z'
tags:
  - feature
  - ai
  - skills
---

## Overview

Skills are AI workflow instructions bundled with Knowns. They are embedded in the binary and synced to platform-specific directories (`.claude/skills/`, `.agent/skills/`, etc.) during `knowns init` or `knowns sync`.

**Related:** @doc/ai/platforms

---

## How Skills Work

1. Skills are embedded in the Knowns binary as SKILL.md files
2. During `knowns init`, skills are copied to the platform directories you selected
3. `knowns sync --skills` re-syncs skills from the binary to platform dirs
4. AI assistants (Claude Code, Kiro, etc.) discover skills from their platform directory

---

## Synced Locations

```
# Platform directories (auto-synced by knowns)
.claude/skills/              # Claude Code
.agent/skills/               # Kiro / Antigravity
```

Skills are synced as SKILL.md files. The format is the same across Claude Code and Kiro/Antigravity.

---

## CLI Commands

```bash
# Sync skills + instructions from binary to platform dirs
knowns sync

# Sync skills only
knowns sync --skills

# Sync instruction files only
knowns sync --instructions

# Sync a specific platform's instruction file
knowns sync --instructions --platform claude
```

---

## Built-in Skills

| Skill | Description |
|-------|-------------|
| `kn-init` | Initialize session, read docs, load memory, understand project |
| `kn-spec` | Create specification document for features (SDD) |
| `kn-plan` | Take task, gather context, create implementation plan |
| `kn-research` | Search codebase, find patterns, explore before coding |
| `kn-implement` | Execute plan, track progress, check acceptance criteria |
| `kn-review` | Multi-perspective code review (P1/P2/P3 severity) |
| `kn-commit` | Create conventional commit with verification |
| `kn-extract` | Extract reusable patterns into docs, templates, and memory |
| `kn-doc` | Create and update documentation |
| `kn-template` | List, run, or create code templates |
| `kn-verify` | Run SDD verification and coverage report |
| `kn-go` | Full pipeline from approved spec (no review gates) |
| `kn-debug` | Structured debugging: triage → fix → learn |

---

## SKILL.md Format

```markdown
---
name: kn-init
description: Initialize session - read docs, load memory, understand project
---

# Session Init

Instructions for the AI workflow...
```

### Frontmatter

| Field | Description |
|-------|-------------|
| `name` | Skill identifier |
| `description` | Short description shown in skill list |

---

## Memory Integration

All skills participate in the memory loop:
- **Read skills** (init, research, plan, spec, go, review, debug): search `type: "memory"` via unified search
- **Write skills** (implement, extract, debug): save patterns/decisions/failures with `memory({ action: "add", layer: "project" })`
- The global rule in KNOWNS.md `## Memory Usage` encourages all skills to save knowledge as it emerges

---

## MCP vs CLI in Skills

Skills use MCP tools when available, with CLI as fallback:

- **Reading tasks/docs**: MCP `tasks({ action: "get" })` or CLI `knowns task <id> --plain`
- **Updating tasks**: MCP `tasks({ action: "update" })` or CLI `knowns task edit <id>`
- **Searching**: MCP `search({ action: "search" })` or CLI `knowns search "query" --plain`
- **Time tracking**: MCP `time({ action: "start" })` or CLI `knowns time start <id>`

MCP is preferred when available because it's faster (direct call vs spawning a process) and returns structured JSON.
