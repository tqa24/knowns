---
name: kn-spec
description: Use when creating a specification document for a feature (SDD workflow)
---

# Creating a Spec Document

Create a specification document for a feature using SDD (Spec-Driven Development).

**Announce:** "Using kn-spec to create spec for [name]."

**Core principle:** SPEC FIRST → REVIEW → APPROVE → THEN PLAN TASKS.

## Inputs

- Feature name
- User requirements, scenarios, constraints, and non-functional expectations
- Related docs/tasks, if any

## Spec Quality Rules

- Requirements must be testable
- ACs must be observable outcomes, not vague goals
- Scenarios should cover happy path plus at least important edge cases
- Open questions should stay explicit instead of being buried in prose
- If background knowledge is too broad for the spec body, move it into a supporting doc and reference it

## Step 1: Get Feature Name

If `$ARGUMENTS` provided, use it as spec name.

If no arguments, ask user:
> What feature are you speccing? (e.g., "user-auth", "payment-flow")

## Step 2: Gather Requirements

Ask user to describe the feature:
> Please describe the feature requirements. What should it do?

Listen for:
- Core functionality
- User stories / scenarios
- Edge cases
- Non-functional requirements

If requirements depend on large domain or architecture context:

- create/update a supporting doc first
- keep the spec focused on product/behavioral requirements
- reference the supporting doc with `@doc/<path>` instead of dumping background material inline

## Step 3: Create Spec Document

```json
mcp__knowns__create_doc({
  "title": "<Feature Name>",
  "description": "Specification for <feature>",
  "folder": "specs",
  "tags": ["spec", "draft"],
  "content": "<spec content>"
})
```

**Spec Template:**

```markdown
## Overview

Brief description of the feature and its purpose.

## Requirements

### Functional Requirements
- FR-1: [Requirement description]
- FR-2: [Requirement description]

### Non-Functional Requirements
- NFR-1: [Performance, security, etc.]

## Acceptance Criteria

- [ ] AC-1: [Testable criterion]
- [ ] AC-2: [Testable criterion]
- [ ] AC-3: [Testable criterion]

## Scenarios

### Scenario 1: [Happy Path]
**Given** [context]
**When** [action]
**Then** [expected result]

### Scenario 2: [Edge Case]
**Given** [context]
**When** [action]
**Then** [expected result]

## Technical Notes

Optional implementation hints or constraints.

## Open Questions

- [ ] Question 1?
- [ ] Question 2?
```

## Step 3.5: Validate Spec

**CRITICAL:** After creating spec, validate to catch issues:

```json
mcp__knowns__validate({ "entity": "specs/<name>" })
```

## Step 4: Ask for Review

Present the spec and ask:
> Please review this spec:
> - **Approve** if requirements are complete
> - **Edit** if you want to modify something
> - **Add more** if requirements are missing

## Step 5: Handle Response

**If approved:**
```json
mcp__knowns__update_doc({
  "path": "specs/<name>",
  "tags": ["spec", "approved"]
})
```

**If edit requested:**
Update the spec based on feedback and return to Step 4.

**If add more:**
Gather additional requirements and update spec.

## Final Response Contract

All built-in skills in scope must end with the same user-facing information order: `kn-init`, `kn-spec`, `kn-plan`, `kn-research`, `kn-implement`, `kn-verify`, `kn-doc`, `kn-template`, `kn-extract`, and `kn-commit`.

Required order for the final user-facing response:

1. Goal/result - state what spec was drafted, revised, approved, or blocked.
2. Key details - include the most important supporting context, refs, open questions, or validation.
3. Next action - recommend a concrete follow-up command only when a natural handoff exists.

Keep this concise for CLI use. Skill-specific content may extend the key-details section, but must not replace or reorder the shared structure.

Out of scope: explaining, syncing, or generating `.claude/skills/*`. Runtime auto-sync already handles platform copies, so this skill source only defines the built-in output contract.

For `kn-spec`, the key details should cover:

- the concrete spec draft or revision
- clear open questions, if any
- approval status
- any validation result or unresolved gaps

## Spillover Rule

If the spec uncovers cross-cutting or general knowledge work:

- create a separate task for that work
- reference it from the spec or generated task set
- keep the spec focused on the feature, not on every general improvement the discussion surfaced

---

## CRITICAL: Next Step Suggestion

**You MUST suggest the next action when a natural follow-up exists. User won't know what to do next.**

After spec is approved:

```
✓ Spec approved: @doc/specs/<name>

This spec will generate multiple tasks. Ready to create them?

Run: /kn-plan --from @doc/specs/<name>
```

**Show what will happen:**
```
This will:
1. Parse requirements from spec
2. Generate tasks with ACs
3. Link all tasks to this spec
4. You review and approve before creation
```

---

## Related Skills

- `/kn-plan --from @doc/specs/<name>` - Generate tasks from this spec
- `/kn-plan <id>` - Plan individual task
- `/kn-verify` - Verify implementation against spec

## Checklist

- [ ] Feature name determined
- [ ] Requirements gathered
- [ ] Spec created in specs/ folder
- [ ] Includes: Overview, Requirements, ACs, Scenarios
- [ ] User reviewed
- [ ] Status updated (draft → approved)
- [ ] **Next step suggested** (/kn-plan --from)

## Red Flags

- Creating spec without user input
- Skipping review step
- Approving without explicit user confirmation
- **Not suggesting task creation after approval**
- Writing implementation notes instead of requirements
- Leaving ambiguous AC text that cannot be verified later
