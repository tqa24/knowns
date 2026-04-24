# Memory System

Memory is where Knowns stores durable context that should be recalled later.

## The three layers

- **working memory**: short-lived, session-scoped context
- **project memory**: decisions, patterns, and conventions specific to one repository
- **global memory**: user-level preferences or reusable rules across projects

## When to use memory instead of docs

Use memory when the information is:

- short enough to recall quickly
- decision-oriented rather than explanatory
- useful across many future interactions

Use docs when the information needs longer narrative explanation or structured sections.

## Typical examples

- “We use repository pattern for data access”
- “Always validate before marking a task done”
- “This team prefers semantic search before manual grep for exploratory work”

## Commands

```bash
knowns memory add "We use repository pattern" --category decision
knowns memory list --plain
knowns memory <id> --plain
```

## Related

- [Task Management](./task-management.md)
- [Reference System](../reference/reference-system.md)
