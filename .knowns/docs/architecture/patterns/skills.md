---
title: Claude Code Skills
description: Pattern for creating and managing Claude Code skills in Knowns CLI
createdAt: '2026-01-17T06:06:37.006Z'
updatedAt: '2026-03-27T07:09:54.296Z'
tags:
  - pattern
  - claude-code
  - skills
---

## Overview

Knowns CLI integrates with Claude Code skills - workflow templates that can be invoked via `/kn-<skill>` commands.

## Skill Structure

```
src/instructions/skills/
├── index.ts                    # Export all skills
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
| `kn-init` | Initialize session (read docs, list tasks, load critical learnings) |
| `kn-spec` | Create spec with Socratic exploring phase (SDD workflow) |
| `kn-plan` | Plan task implementation (with pre-execution validation) |
| `kn-implement` | Implement task (includes reopen logic) |
| `kn-research` | Research codebase before implementation |
| `kn-review` | Multi-perspective code review with P1/P2/P3 severity |
| `kn-commit` | Generate commit message and commit changes |
| `kn-extract` | Extract patterns, decisions, failures + consolidation mode |
| `kn-doc` | Create and update documentation |
| `kn-template` | Generate code from templates |
| `kn-verify` | SDD verification and coverage reporting |
| `kn-go` | Full pipeline execution from approved spec (no review gates) |
| `kn-debug` | Structured debugging: triage → reproduce → fix → learn |
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
knowns sync                   # Sync all (skills + agents)
knowns sync skills            # Sync skills only
knowns sync skills --force    # Force overwrite
knowns sync agent             # Sync agent files only
knowns sync agent --type mcp  # Use MCP guidelines
knowns sync agent --all       # Include Gemini, Copilot
```

## Implementation Details

### index.ts Pattern

```typescript
import knInitMd from "./kn-init/SKILL.md";

function createSkill(content: string, folderName: string): Skill {
  const { name, description } = parseSkillFrontmatter(content);
  return { name, folderName, description, content };
}

export const SKILL_INIT = createSkill(knInitMd, "kn-init");
```

### Sync Function

Skills are synced to `.claude/skills/<folder-name>/SKILL.md` in the project.
