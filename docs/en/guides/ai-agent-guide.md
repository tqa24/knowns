# AI Agent Guide

This guide explains how to use Knowns effectively with AI assistants.

## Core idea

AI works better when it does not have to guess project context.

Knowns gives AI a structured way to access:

- tasks
- docs
- memory
- references
- validation
- search and retrieval

## Recommended usage pattern

### 1. Load guidance first

The AI should start with the canonical repository guidance in `KNOWNS.md`.

### 2. Use tasks as execution targets

Instead of giving a vague prompt, point the AI at a task with acceptance criteria.

### 3. Use docs for durable explanation

Architecture, patterns, and operational guidance should live in docs rather than only in chat.

### 4. Use memory for durable decisions

Store reusable decisions, conventions, and failures in memory so they can be recalled later.

### 5. Validate before calling work complete

Validation should be part of the normal workflow.

## MCP vs CLI

### Prefer MCP when:

- the AI runtime supports it
- you want structured tool calls
- you want less shell parsing and less prompt copy-paste

### Prefer CLI when:

- MCP is unavailable
- you are scripting outside an MCP-aware runtime
- you want to inspect output manually in a terminal

## Example workflow

1. AI reads `KNOWNS.md`
2. AI reads the target task
3. AI follows any `@doc/...` or `@task-...` references
4. AI searches or retrieves additional context if needed
5. AI implements changes
6. AI runs validation or tests

## Related

- [Task Management](./task-management.md)
- [MCP Integration](./mcp-integration.md)
- [Workflow](./workflow.md)
