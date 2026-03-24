---
name: kn-extract
description: Use when extracting reusable patterns, solutions, or knowledge into documentation
---

# Extracting Knowledge

**Announce:** "Using kn-extract to extract knowledge."

**Core principle:** ONLY EXTRACT GENERALIZABLE KNOWLEDGE.

## Inputs

- Usually a completed task ID
- Sometimes a code change, repeated pattern, or recurring support issue

## Extraction Rules

- Extract patterns, not one-off hacks
- Prefer updating an existing doc over creating a duplicate
- Link the extracted knowledge back to the source task or source doc
- Only create a template if the pattern is genuinely reusable for generation

## Step 1: Identify Source

```json
mcp__knowns__get_task({ "taskId": "$ARGUMENTS" })
```

Look for: patterns, problems solved, decisions made, lessons learned.

## Step 2: Search for Existing Docs

```json
mcp__knowns__search({ "query": "<pattern/topic>", "type": "doc" })
```

**Don't duplicate.** Update existing docs when possible.

## Step 3: Create Documentation

```json
mcp__knowns__create_doc({
  "title": "Pattern: <Name>",
  "description": "Reusable pattern for <purpose>",
  "tags": ["pattern", "<domain>"],
  "folder": "patterns"
})

mcp__knowns__update_doc({
  "path": "patterns/<name>",
  "content": "# Pattern: <Name>\n\n## Problem\n...\n\n## Solution\n...\n\n## Example\n```typescript\n// Code\n```\n\n## Source\n@task-<id>"
})
```

## Step 4: Create Template (if code-generatable)

```json
mcp__knowns__create_template({
  "name": "<pattern-name>",
  "description": "Generate <what>",
  "doc": "patterns/<pattern-name>"
})
```

Link template in doc:
```json
mcp__knowns__update_doc({
  "path": "patterns/<name>",
  "appendContent": "\n\n## Generate\n\nUse @template/<pattern-name>"
})
```

## Step 5: Validate

**CRITICAL:** After creating doc/template, validate to catch broken refs:

```json
mcp__knowns__validate({ "entity": "patterns/<name>" })
```

If errors found, fix before continuing.

## Step 6: Link Back to Task

```json
mcp__knowns__update_task({
  "taskId": "$ARGUMENTS",
  "appendNotes": "📚 Extracted to @doc/patterns/<name>"
})
```

## Shared Output Contract

All built-in skills in scope must end with the same user-facing information order: `kn-init`, `kn-spec`, `kn-plan`, `kn-research`, `kn-implement`, `kn-verify`, `kn-doc`, `kn-template`, `kn-extract`, and `kn-commit`.

Required order for the final user-facing response:

1. Goal/result - state what knowledge was extracted, updated, or intentionally not extracted.
2. Key details - include the most important supporting context, refs, canonical location, warnings, or validation.
3. Next action - recommend a concrete follow-up command only when a natural handoff exists.

Keep this concise for CLI use. Extraction-specific content may extend the key-details section, but must not replace or reorder the shared structure.

Out of scope: explaining, syncing, or generating `.claude/skills/*`. Runtime auto-sync already handles platform copies, so this skill source only defines the built-in output contract.

For `kn-extract`, the key details should cover:

- what knowledge was extracted
- whether a doc was created or updated
- whether a template was created
- where the canonical knowledge now lives

When the extraction leads to a clear follow-up, include the best next command. If the correct outcome is a no-op or a completed doc update with no obvious continuation, stop after the result and key details.

## No-Op Case

If the work is too specific to generalize, say so explicitly and do not force a new doc.

## What to Extract

| Source | Extract As | Template? |
|--------|------------|-----------|
| Code pattern | Pattern doc | ✅ Yes |
| API pattern | Integration guide | ✅ Yes |
| Error solution | Troubleshooting | ❌ No |
| Security approach | Guidelines | ❌ No |

## Checklist

- [ ] Knowledge is generalizable
- [ ] Includes working example
- [ ] Links back to source
- [ ] Template created (if applicable)
- [ ] **Validated (no broken refs)**
