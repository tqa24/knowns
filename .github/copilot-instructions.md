# knowns - GitHub Copilot Instructions

Compatibility entrypoint for runtimes that auto-detect `.github/copilot-instructions.md`.

<!-- KNOWNS GUIDELINES START -->

## Canonical Guidance

- Knowns is the repository memory layer for humans and the AI-friendly working layer for agents.
- The source of truth for repo-level agent guidance is `KNOWNS.md`.
- Read `KNOWNS.md` first whenever the runtime supports reading repository files.
- If this file and `KNOWNS.md` differ, follow `KNOWNS.md`.

## Minimum Rules

- Use Knowns as the canonical system for tasks, docs, templates, and workflow state.
- Never manually edit Knowns-managed task or doc markdown.
- Search first, then read only relevant docs and code.
- Plan before implementation unless the user explicitly overrides that workflow.
- Validate before considering work complete.

## Quick Reference

```bash
knowns doc list --plain               # List docs
knowns task list --plain              # List tasks
knowns task <id> --plain              # View task
knowns doc "<path>" --plain --smart  # View doc
knowns search "query" --plain        # Search docs/tasks
knowns guidelines --plain             # Full workflow reference
```

<!-- KNOWNS GUIDELINES END -->
