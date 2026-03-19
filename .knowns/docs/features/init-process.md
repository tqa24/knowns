---
title: Init Process
createdAt: '2026-01-23T04:13:12.738Z'
updatedAt: '2026-03-12T17:59:54.231Z'
description: Detailed init wizard flow and configuration steps
tags:
  - feature
  - init
  - wizard
---
## Overview

`knowns init` is an interactive wizard to set up Knowns in a project. Written in Go using the Cobra CLI framework, the init command guides users through project configuration and then executes post-wizard steps with progressive animation.

**Key technologies:**
- **Cobra** (`github.com/spf13/cobra`) — CLI command framework
- **huh** (`github.com/charmbracelet/huh`) — Interactive wizard forms (Catppuccin theme)
- **bubbletea** (`charm.land/bubbletea/v2`) — Download progress bars
- **Goroutine spinner** — Task step animation (avoids terminal escape sequence issues)

**Source files:**
- `internal/cli/init.go` — Main init command, wizard form, config types
- `internal/cli/init_steps.go` — Progressive step runner, goroutine spinner
- `internal/cli/download_setup.go` — Download steps with bubbletea progress bars

---

## Quick Start

```bash
# Interactive (full wizard)
knowns init

# Quick init with defaults (skip wizard)
knowns init my-project --no-wizard

# Force reinitialize
knowns init --force
```

---

## Wizard Flow

The wizard uses the `huh` library to present three form groups sequentially. Groups can be conditionally hidden based on previous answers.

```
┌─────────────────────────────────────────────────────────┐
│                  knowns init                             │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Group 1: Project Setup                                  │
│ • Project name (text input)                             │
│ • Git tracking mode (select, if in git repo)            │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Group 2: Semantic Search                                │
│ • Enable semantic search? (confirm)                     │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Group 3: Model Selection (hidden if semantic disabled)  │
│ • Select embedding model (select)                       │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Post-Wizard: Progressive Step Animation                 │
│ • Create project structure       ✓                      │
│ • Apply settings                 ✓                      │
│ • Configure git integration      ✓                      │
│ • Download ONNX Runtime          ⠋ (progress bar)       │
│ • Download model files           ○ (pending)            │
│ • Sync skills                    ○                      │
│ • Create MCP config              ○                      │
│ • Create instruction files       ○                      │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Done! Show get-started hints                            │
└─────────────────────────────────────────────────────────┘
```

---

## Group 1: Project Setup

```
? Project name: (my-project)
  # Default: current directory name

? Git tracking mode:
  > Git Tracked (recommended for teams)
    Git Ignored (personal use)
    None (ignore all)
```

The git tracking select field only appears if the current directory is a git repository AND neither `--git-tracked` nor `--git-ignored` flags were provided.

**Config struct:**
```go
type initConfig struct {
    Name             string
    GitTrackingMode  string   // "git-tracked" | "git-ignored" | "none"
    EnableSemantic   bool
    SemanticModel    string
}
```

---

## Group 2: Semantic Search

```
? Enable semantic search?
  Requires embedding model download
  > Yes / No
```

Defaults to `true`. If disabled, Group 3 is hidden and no download steps are added to the post-wizard pipeline.

---

## Group 3: Model Selection

Only shown when semantic search is enabled (uses `WithHideFunc`).

```
? Select embedding model:
  > gte-small (recommended) — 384 dims, 67MB — best balance
    all-MiniLM-L6-v2 — 384 dims, 45MB — fastest
    gte-base — 768 dims, 220MB — highest quality
    bge-small-en-v1.5 — 384 dims, 67MB — strong retrieval
    bge-base-en-v1.5 — 768 dims, 220MB — top retrieval quality
    nomic-embed-text-v1.5 — 768 dims, 274MB — long context (8192 tokens)
    multilingual-e5-small — 384 dims, 471MB — multilingual support
```

The selected model determines which files are downloaded in the post-wizard phase and is saved to `config.json` under `settings.semanticSearch`.

---

## Progressive Step Animation

After the wizard completes, all setup tasks run sequentially with animated feedback. Steps appear one by one — each step is only shown once the previous step finishes.

### Step Types

There are two types of steps, each with a different animation strategy:

**Task steps** use a goroutine-based spinner (not bubbletea). This avoids terminal escape sequence issues that can occur when mixing bubbletea programs with raw terminal output.

```go
// Goroutine spinner — writes directly to stderr
go func() {
    for !stopped.Load() {
        frame := StyleDim.Render(spinnerFrames[i%len(spinnerFrames)])
        fmt.Fprintf(os.Stderr, "\r  %s %s", frame, step.label)
        time.Sleep(80 * time.Millisecond)
        i++
    }
}()
```

**Download steps** are batched and run via a bubbletea `setupModel` with progress bars, spinners, speed display, and size tracking.

### Visual States

Each step displays in one of four visual states:

| State | Icon | Description |
|-------|------|-------------|
| Done | `✓` (green) | Step completed successfully |
| Active | `⠋` (spinner) | Currently executing |
| Pending | `○` (dim) | Waiting for previous steps |
| Error | `✗` (yellow) | Step failed with error message |

### Example Output

```
  ✓ Creating project structure
  ✓ Applying settings
  ✓ Configuring git integration
  ⠋ ONNX Runtime (darwin/arm64)
    ▓▓▓▓▓▓▓░░░░░░░░░░░░░ 35%  12MB/35MB  2.1MB/s
  ○ gte-small (recommended) — model.onnx
  ○ gte-small (recommended) — tokenizer.json
  ○ gte-small (recommended) — config.json
```

After all downloads complete:

```
  ✓ Creating project structure
  ✓ Applying settings
  ✓ Configuring git integration
  ✓ ONNX Runtime (darwin/arm64) (35MB)
  ✓ gte-small (recommended) — model.onnx (67MB)
  ✓ gte-small (recommended) — tokenizer.json (712KB)
  ✓ gte-small (recommended) — config.json (1KB)
  ✓ Syncing skills
  ✓ Creating MCP config
  ✓ Creating instruction files
```

### Step Execution Logic

The `runInitSteps` function in `init_steps.go` processes steps sequentially:

1. **Task steps** (steps with a `run` function): Executed with `runTaskStepAnimated()` — a goroutine spinner writes to stderr while the task runs on the main goroutine.
2. **Download steps** (steps with a `url` field): Consecutive download steps are batched and run together through a bubbletea `setupModel` program with progress bars.

```go
func runInitSteps(steps []initStep) error {
    for i < len(steps) {
        if step.run != nil {
            // Task step — goroutine spinner
            runTaskStepAnimated(step)
        } else {
            // Batch consecutive download steps
            // Run via bubbletea setupModel with progress bars
        }
    }
}
```

---

## Post-Wizard Steps (In Order)

### 1. Creating project structure

Creates the `.knowns/` directory tree via `storage.NewStore(root).Init(name)`:

```
.knowns/
├── config.json              # Project config
├── tasks/                   # Task files
├── docs/                    # Documentation
└── templates/               # Code templates
```

### 2. Applying settings

Loads the config created in step 1, then applies wizard answers:
- Sets `gitTrackingMode` from wizard selection
- If semantic search is enabled, saves `semanticSearch` settings with model ID, HuggingFace ID, dimensions, and max tokens

### 3. Configuring git integration

Based on the selected git tracking mode:

| Mode | .gitignore entries |
|------|-------------------|
| `git-tracked` | (nothing added) |
| `git-ignored` | `.knowns/` |
| `none` | `.knowns/` |

If semantic search is enabled, `.knowns/search-index/` is always added to `.gitignore`.

### 4. Semantic search downloads (conditional)

Only runs if semantic search was enabled. Three possible outcomes:

- **Already installed**: Shows a single "Semantic search (already installed)" task step
- **Needs download**: Adds download steps for ONNX Runtime and model files
- **Setup failed**: Prints warning and suggests `knowns model download <model>`

Download steps include:
- **ONNX Runtime** (`{os}/{arch}`): Downloaded as `.tgz`, extracted via `postHook` to `~/.knowns/lib/`
- **Model files**: Downloaded from HuggingFace to `~/.knowns/models/{huggingface-id}/`

### 5. Syncing skills

Runs `codegen.SyncSkills(cwd)` to generate skill files from embedded templates.

### 6. Creating MCP config

Creates `.mcp.json` in the project root for Claude Code MCP auto-discovery:

```json
{
  "mcpServers": {
    "knowns": {
      "command": "/path/to/knowns",
      "args": ["mcp", "--stdio"]
    }
  }
}
```

The command path is resolved from `os.Executable()`, falling back to `"knowns"`.

### 7. Creating instruction files

Generates agent instruction files for all supported platforms:

| File | Platform |
|------|----------|
| `CLAUDE.md` | Claude Code |
| `GEMINI.md` | Gemini CLI |
| `AGENTS.md` | Generic AI |
| `.github/copilot-instructions.md` | GitHub Copilot |

Each file contains a condensed version of the Knowns guidelines with quick reference commands. Files are skipped if they already exist (unless `--force`).

---

## CLI Options

```bash
knowns init [name] [flags]

# Arguments:
#   name          Project name (default: directory name)

# Flags:
#   --git-tracked     Track .knowns/ files in git
#   --git-ignored     Add .knowns/ to .gitignore
#   --wizard          Run interactive setup wizard (default behavior)
#   --no-wizard       Skip interactive prompts, use defaults
#   -f, --force       Force reinitialize even if already initialized
```

### Non-interactive mode

When `--no-wizard` is passed or a name argument is provided, the wizard is skipped and defaults are used:
- Name: argument value or directory name
- Git tracking: `git-tracked` (unless overridden by flags)
- Semantic search: enabled with `gte-small` model

---

## Config Output

```json
// .knowns/config.json
{
  "name": "my-project",
  "settings": {
    "gitTrackingMode": "git-tracked",
    "semanticSearch": {
      "enabled": true,
      "model": "gte-small",
      "huggingFaceId": "Xenova/gte-small",
      "dimensions": 384,
      "maxTokens": 512
    }
  }
}
```

---

## Complete Example

```bash
$ knowns init

🚀 Knowns Project Setup
   Quick configuration

? Project name: my-awesome-app

? Git tracking mode: Git Tracked (recommended for teams)

? Enable semantic search? Yes

? Select embedding model: gte-small (recommended) — 384 dims, 67MB — best balance

  ✓ Creating project structure
  ✓ Applying settings
  ✓ Configuring git integration
  ⠋ ONNX Runtime (darwin/arm64)
    ▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░ 65%  23MB/35MB  3.2MB/s
  ○ gte-small (recommended) — model.onnx
  ○ gte-small (recommended) — tokenizer.json
  ...

  ✓ Creating project structure
  ✓ Applying settings
  ✓ Configuring git integration
  ✓ ONNX Runtime (darwin/arm64) (35MB)
  ✓ gte-small (recommended) — model.onnx (67MB)
  ✓ gte-small (recommended) — tokenizer.json (712KB)
  ✓ gte-small (recommended) — config.json (1KB)
  ✓ Syncing skills
  ✓ Creating MCP config
  ✓ Creating instruction files

Get started:
  knowns task create "My first task"
  Use /kn-init to start an AI session
```

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Already initialized | Shows warning, exits (use `--force` to override) |
| No git repo | Shows warning but continues |
| User cancels wizard (Ctrl+C) | Shows "Setup cancelled", exits cleanly |
| Download fails | Step shows `✗` with error, init aborts |
| Semantic setup fails | Warns, suggests `knowns model download` |
| File already exists | Skipped (unless `--force`) |

---

## Architecture Notes

### Why two animation strategies?

**Task steps** (creating directories, writing configs) use a simple goroutine spinner that writes to stderr. This avoids the terminal escape sequence leakage (`^[[?2026;2$y`) that can occur when multiple bubbletea programs run in sequence.

**Download steps** use bubbletea because they need richer UI: progress bars, percentage display, speed calculation, and multi-step state management. Consecutive download steps are batched into a single bubbletea program to avoid repeated terminal initialization.

### drainStdin

The `drainStdin()` function in `download_setup.go` reads and discards pending bytes on stdin (non-blocking) between bubbletea programs. This prevents stale terminal escape responses (DECRQM) from leaking into the shell after huh/bubbletea programs exit.

### AI platforms are auto-configured

Unlike the previous Node version which had an AI platform selection step, the Go version automatically generates instruction files for all supported platforms (Claude Code, OpenCode, Gemini CLI, GitHub Copilot, Generic AI) and writes project-level MCP config for Claude Code and OpenCode. This reduces wizard friction since most users want the common AI tooling configured during init.
