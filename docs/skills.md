# Skills

Skills are reusable AI workflows bundled with Knowns.

---

## Built-in Skills

Knowns ships 13 built-in skills:

| Skill | Trigger | Purpose |
| ----- | ------- | ------- |
| `kn-init` | `/kn-init` | Read project context, load critical learnings, load project memories |
| `kn-spec` | `/kn-spec` | Create spec with Socratic exploring phase (SDD) |
| `kn-plan` | `/kn-plan` | Create implementation plan with pre-execution validation |
| `kn-research` | `/kn-research` | Research docs and code before changes |
| `kn-implement` | `/kn-implement` | Execute an approved task |
| `kn-review` | `/kn-review` | Multi-perspective code review (P1/P2/P3 severity) |
| `kn-verify` | `/kn-verify` | SDD verification and coverage reporting |
| `kn-commit` | `/kn-commit` | Commit changes with conventional format |
| `kn-extract` | `/kn-extract` | Extract patterns, decisions, failures + consolidation |
| `kn-doc` | `/kn-doc` | Create or update documentation |
| `kn-template` | `/kn-template` | Work with code generation templates |
| `kn-go` | `/kn-go` | Full pipeline from approved spec (no review gates) |
| `kn-debug` | `/kn-debug` | Structured debugging: triage → fix → learn |

---

## CLI Commands

Skills are synced via the top-level `knowns sync` command:

```bash
knowns sync              # Sync all (skills + instructions + model + index)
knowns sync --skills     # Sync skills only
```

---

## Where Skills Live

Built-in skill files are typically synced into Claude-compatible folders such as:

```text
.claude/skills/
  kn-init/
    SKILL.md
  kn-plan/
    SKILL.md
  ...
```

The exact synced output depends on the active platforms and sync implementation in the current CLI build.

---

## Typical Workflow

### Manual flow (step by step)

```text
/kn-init → /kn-research → /kn-spec → /kn-plan --from → /kn-plan <id> → /kn-implement <id> → /kn-review → /kn-commit → /kn-extract
```

### Go mode (full pipeline from approved spec)

After creating and approving a spec with `/kn-spec`, run the entire pipeline in one shot:

```text
/kn-spec <name> → approve → /kn-go specs/<name>
```

Go mode generates tasks, plans, implements, verifies, and commits — only stopping once at the end for commit confirmation.

### Debugging flow

```text
/kn-debug → /kn-review → /kn-commit
```

### New skills detail

**`kn-review`** — Runs after implementation, before commit. Reviews code from 4 perspectives (quality, architecture, security, completeness). Findings are triaged as P1 (blocks commit), P2 (should fix), P3 (nice to have). P1 findings are a hard gate.

**`kn-go`** — Full pipeline execution from an approved spec. Generates tasks → plans each → implements each → verifies SDD coverage → commits. Supports `--dry-run` to preview, re-run to continue from where it left off, and context budget checkpointing.

**`kn-debug`** — Structured debugging: classify → check known patterns → reproduce → root cause → fix → verify → capture learning. Integrates with unified search (docs + memories) to find previously documented solutions. Saves quick debug patterns to project memory.

**`kn-extract` (enhanced)** — Now captures three categories: patterns, decisions (good/bad/surprise/tradeoff), and failures. Saves concise entries to project memory for fast agent recall alongside full docs. Promotes critical learnings to `learnings/critical-patterns` which is read by `kn-init`. Use `--consolidate` to review and merge all existing learnings.

**`kn-spec` (enhanced)** — Now includes Phase 0 Socratic exploring: assesses scope, classifies domain, identifies gray areas, asks one question at a time to lock decisions before writing the spec. Use `--skip-explore` for trivial features.

**`kn-plan` (enhanced)** — Now includes pre-execution plan check before approval: AC coverage, scope sizing, dependency check, and risk assessment.

---

## Imported Skills

Imported packages may provide additional skills.

```bash
knowns import add <source>
knowns import sync
```

---

## Custom Skills

You can add custom skill folders alongside synced skills by creating a `SKILL.md` in the appropriate platform directory (e.g., `.claude/skills/my-skill/SKILL.md`).

Keep custom skills separate from generated content so future syncs don't overwrite them.

---

## Related

- [Guidelines](./guidelines.md) - How full guidance is exposed
- [Multi-Platform](./multi-platform.md) - Platform targets and sync behavior
- [Command Reference](./commands.md) - Current CLI syntax
