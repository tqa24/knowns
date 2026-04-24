# Workflow

This is the recommended human + AI workflow with Knowns.

## Suggested loop

1. run `knowns init` once per project
2. create tasks and supporting docs
3. let AI read `KNOWNS.md`, tasks, docs, and memory
4. implement changes
5. validate
6. sync or update generated artifacts when needed

## Typical command loop

```bash
knowns task create "..."
knowns doc create "..."
knowns search "..." --plain
knowns retrieve "..." --json
knowns validate --plain
knowns sync
```

## Why this works well

- tasks define execution targets
- docs explain structure and intent
- memory preserves decisions and conventions
- retrieval connects everything for humans and AI

## When to use which surface

- CLI: quick authoring and scripting
- MCP: structured AI integration
- browser UI: board/docs/graph/chat workflows
