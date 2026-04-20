<p align="center">
  <img src="./images/cover.svg" alt="Knowns — The memory layer for AI-native development" width="100%">
</p>

# Knowns

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/go-%3E%3D1.24.2-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="https://www.npmjs.com/package/knowns"><img src="https://img.shields.io/npm/v/knowns.svg?style=flat-square" alt="npm"></a>
  <a href="https://github.com/knowns-dev/knowns/actions/workflows/ci.yml"><img src="https://github.com/knowns-dev/knowns/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="#installation"><img src="https://img.shields.io/badge/platform-win%20%7C%20mac%20%7C%20linux-lightgrey?style=flat-square" alt="Platform"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/knowns-dev/knowns?style=flat-square" alt="License"></a>
</p>

> Turn stateless AI into a project-aware engineering partner.

> [!WARNING]
> Knowns is under active development. APIs, database schemas, and configuration formats may change between releases. Review the known limitations and security considerations before deploying to production.

> [!IMPORTANT]
> **v0.13+: Rewritten in Go.** To support AI Agent Workspaces (process management, live terminal, git worktree isolation), Knowns has been rewritten in Go as a native binary. CLI commands and `.knowns/` data format are fully backward-compatible. Install via `npm i -g knowns` still works (auto-downloads platform binary).

**Knowns is the memory layer for AI-native software development — enabling AI to understand your project instantly.**

Instead of starting from zero every session, AI works with structured, persistent project context.

No repeated explanations.  
No pasted docs.  
No lost architectural knowledge.

Just AI that already understands your system.

⭐ If you believe AI should truly understand software projects, consider giving **Knowns** a star.

<p align="center">
  <img src="./images/task-workflow.gif" alt="Knowns task workflow demo" width="100%">
</p>

## Table of Contents

- [Why Knowns?](#why-knowns)
- [What is Knowns?](#what-is-knowns---really)
- [Core Capabilities](#core-capabilities)
- [How It Works](#how-it-works)
- [Installation](#installation)
- [What You Can Build](#what-you-can-build-with-knowns)
- [Quick Reference](#quick-reference)
- [Claude Code Skills Workflow](#claude-code-skills-workflow)
- [Documentation](#documentation)
- [Roadmap](#roadmap)
- [Development](#development)
- [Links](#links)

---

## Why Knowns?

AI is powerful — but fundamentally **stateless**.

Every session forces developers to:

- Re-explain architecture
- Paste documentation
- Repeat conventions
- Clarify past decisions
- Rebuild context

This breaks flow and limits AI’s effectiveness.

### AI doesn't lack intelligence.

### It lacks the right context.

**Knowns fixes that.**

---

## What is Knowns - Really?

Knowns provides **persistent, structured project understanding** so AI can operate with full awareness of your software environment.

Think of it as your project's **external brain**.

<p align="center">
  <img src="./images/knowledge-graph.svg" alt="Knowns Knowledge Graph" width="100%">
</p>

Knowns connects:

- Specs
- Tasks
- Documentation
- Decisions
- Team knowledge

So AI doesn’t just generate code — it understands what it’s building.

---

## Core Capabilities

<p align="center">
  <img src="./images/capabilities.svg" alt="Knowns Core Capabilities" width="100%">
</p>

### 🧠 Persistent Project Memory

Give AI long-term understanding of your codebase and workflows.

### 🔗 Structured Knowledge

Connect specs, tasks, and docs into a unified context layer.

### ⚡ Smart Context Delivery

Automatically provide relevant context to AI — reducing noise and token usage.

### 🤝 AI-Native Workflow

Transform AI from a tool into a true engineering collaborator.

### 🔐 Self-Hostable

Keep your knowledge private and fully under your control.

## What's New In v0.18.0

- Workspace-aware browser mode: run `knowns browser` outside a repo, scan for projects, and switch workspaces from the UI without restarting the server.
- AST code intelligence: index Go, TypeScript, JavaScript, and Python symbols with `knowns code ingest`, keep them fresh with `knowns code watch`, and inspect relationships with `knowns code search`, `knowns code deps`, and `knowns code symbols`.
- Code graph support: the browser graph can now include indexed code nodes and dependency edges alongside tasks, docs, and memories.
- Chat runtime upgrades: the chat UI includes stronger tool-output rendering, timeline/history navigation, and clearer OpenCode runtime status.

---

## How It Works

Knowns sits **above your existing tools** and makes them readable by AI.

<p align="center">
  <img src="./images/architecture.svg" alt="Knowns Architecture" width="100%">
</p>

Your stack stays the same.

But now:

- Specs → understood
- Tasks → connected
- Docs → usable
- Decisions → remembered

AI stops guessing — and starts contributing.

---

## Installation

### Pre-built binaries

```bash
# Homebrew (macOS/Linux)
brew install knowns-dev/tap/knowns
```

```bash
# Shell installer (macOS/Linux)
curl -fsSL https://knowns.sh/script/install | sh

# Or with wget
wget -qO- https://knowns.sh/script/install | sh

# Install a specific version
curl -fsSL https://knowns.sh/script/install | KNOWNS_VERSION=0.18.0 sh
```

```powershell
# PowerShell installer (Windows)
irm https://knowns.sh/script/install.ps1 | iex

# Install a specific version
$env:KNOWNS_VERSION = "0.18.0"; irm https://knowns.sh/script/install.ps1 | iex
```

The shell installer on macOS/Linux and the PowerShell installer on Windows both auto-run `knowns search --install-runtime` after installing the binary. If that step fails, rerun it manually.

### Uninstall

```bash
# Shell uninstaller (macOS/Linux)
curl -fsSL https://knowns.sh/script/uninstall | sh
```

```powershell
# PowerShell uninstaller (Windows)
irm https://knowns.sh/script/uninstall.ps1 | iex
```

The uninstall scripts only remove installed CLI binaries and PATH entries added by the installer. They leave project `.knowns/` folders untouched.

```bash
# npm — installs platform-specific binary automatically
npm install -g knowns

# npx (no install)
npx knowns
```

### From source (Go 1.24.2+)

```bash
# Install to GOPATH/bin
go install github.com/howznguyen/knowns/cmd/knowns@latest

# Or clone and build
git clone https://github.com/knowns-dev/knowns.git
cd knowns
make build        # Output: bin/knowns
make install      # Install to GOPATH/bin
```

### Get started

```bash
knowns init
knowns browser --open   # Start Web UI and open browser
```

---

## What You Can Build With Knowns

| Feature             | Description                                        |
| ------------------- | -------------------------------------------------- |
| **Task Management** | Create, track tasks with acceptance criteria       |
| **Documentation**   | Nested folders with markdown + mermaid support     |
| **Semantic Search** | Search by meaning with local AI models (offline)   |
| **Time Tracking**   | Built-in timers and reports                        |
| **Context Linking** | `@task-42` and `@doc/patterns/auth` references     |
| **Validation**      | Check broken refs with `knowns validate`           |
| **Template System** | Code generation with Handlebars (`.hbs`) templates |
| **Import System**   | Import docs/templates from git, npm, or local      |
| **Memory System**   | 3-layer memory (project/working/global) for AI recall |
| **AI Integration**  | Full MCP Server with AC/plan/notes operations      |
| **AI Workspaces**   | Multi-phase agent orchestration with live terminal |
| **Code Intelligence** | AST indexing, code search, and dependency graph   |
| **Web UI**          | Kanban board, doc browser, mermaid diagrams        |
| **Knowledge Graph** | Visual graph of tasks, docs, memories, and optional code relationships |

---

## Quick Reference

```bash
# Tasks
knowns task create "Title" -d "Description" --ac "Criterion"
knowns task list --plain
knowns task <id> --plain                    # View task (shorthand)
knowns task edit <id> -s in-progress -a @me
knowns task edit <id> --check-ac 1          # Mark AC as done

# Documentation
knowns doc create "Title" -d "Description" -f "folder"
knowns doc "doc-name" --plain               # View doc (shorthand)
knowns doc "doc-name" --smart --plain       # Auto-handle small/large docs
knowns doc "doc-name" --section "2" --plain # Read specific section

# Templates
knowns template list                        # List available templates
knowns template run <name> --name "X"       # Generate from template
knowns template create <name>               # Create new template

# Imports
knowns import add <name> <source>           # Add import
knowns import sync                          # Sync all imports
knowns import list                          # List imports

# Time, Search & Validate
knowns time start <id> && knowns time stop
knowns search "query" --plain
knowns validate                             # Check broken refs

# Code intelligence
knowns code ingest
knowns code search "oauth login" --neighbors 5
knowns code deps --type calls
knowns code symbols --kind function

# AI Guidelines
knowns agents --sync                        # Sync/generate instruction files
knowns sync                                 # Sync skills + instruction files
```

---

## Claude Code Skills Workflow

When using Claude Code with Knowns, use skills (slash commands) for a guided workflow:

<p align="center">
  <img src="./images/workflow.svg" alt="Knowns AI Workflow" width="100%">
</p>

```
/kn-init     → Read project docs, understand context
/kn-plan     → Take task, gather context, create plan (wait for approval)
/kn-implement → Follow plan, check ACs, track progress
/kn-commit   → Commit with conventional format
```

### Example Session

```
You: /kn-init
Claude: [Reads README, ARCHITECTURE, checks task backlog]
        "Project uses NestJS with CQRS pattern. 3 tasks in-progress..."

You: /kn-plan 42
Claude: [Takes task, reads refs, searches docs]
        "Task #42: Add user authentication

         ## Implementation Plan
         1. Review @doc/patterns/auth
         2. Create AuthModule with JWT strategy
         3. Add login/register endpoints
         4. Write unit tests

         Approve this plan?"

You: Yes

You: /kn-implement 42
Claude: [Follows plan, generates code, checks ACs]
        "✓ AC1: User can login - DONE
         ✓ AC2: JWT token returned - DONE
         ✓ AC3: Tests pass - DONE

         All acceptance criteria complete. Ready to commit?"

You: /kn-commit
Claude: [Creates conventional commit]
        "feat(auth): add JWT authentication

         - Add AuthModule with passport JWT strategy
         - Add login/register endpoints
         - Add unit tests (94% coverage)"
```

### All Skills

| Skill                | Description                                             |
| -------------------- | ------------------------------------------------------- |
| `/kn-init`           | Initialize session - read docs, load memory, understand project |
| `/kn-plan <id>`      | Take task, gather context, create implementation plan           |
| `/kn-implement <id>` | Execute plan, track progress, check acceptance criteria         |
| `/kn-research`       | Search codebase, find patterns, explore before coding           |
| `/kn-commit`         | Create conventional commit with verification                    |
| `/kn-spec`           | Create specification document for features (SDD)                |
| `/kn-go <spec>`      | Full pipeline from approved spec (no review gates)              |
| `/kn-verify`         | Run SDD verification and coverage report                        |
| `/kn-review`         | Multi-perspective code review (P1/P2/P3 severity)               |
| `/kn-doc`            | Create or update documentation                                  |
| `/kn-extract`        | Extract reusable patterns into docs, templates, and memory      |
| `/kn-template`       | List, run, or create code templates                             |
| `/kn-debug`          | Debug errors and failures with memory-backed triage             |

---

## Documentation

| Guide                                          | Description                                |
| ---------------------------------------------- | ------------------------------------------ |
| [Command Reference](./docs/commands.md)        | All CLI commands with examples             |
| [Workflow Guide](./docs/workflow.md)           | Task lifecycle from creation to completion |
| [Reference System](./docs/reference-system.md) | How `@doc/` and `@task-` linking works     |
| [Semantic Search](./docs/semantic-search.md)   | Setup and usage of AI-powered search       |
| [Templates](./docs/templates.md)               | Code generation with Handlebars            |
| [Web UI](./docs/web-ui.md)                     | Kanban board, doc browser, and knowledge graph |
| [MCP Integration](./docs/mcp-integration.md)   | Claude Desktop setup with full MCP tools   |
| [Configuration](./docs/configuration.md)       | Project structure and options              |
| [Developer Guide](./docs/developer-guide.md)   | Technical docs for contributors            |
| [User Guide](./docs/user-guide.md)             | Getting started and daily usage            |
| [Multi-Platform](./docs/multi-platform.md)     | Cross-platform build and distribution      |

---

## Roadmap

### AI Agent Workspaces ✅ (Active)

Multi-phase agent orchestration — assign tasks to AI agents with git worktree isolation, live terminal streaming, and automatic phase progression (research → plan → implement → review).

### Self-Hosted Team Sync 🚧 (Planned)

Optional self-hosted sync server for shared visibility without giving up local-first workflows.

- **Real-time visibility** — See who is working on what
- **Shared knowledge** — Sync tasks and documentation across the team
- **Full data control** — Self-hosted, no cloud dependency

---

## Development

Requires **Go 1.24.2+** and optionally **Node.js + pnpm** for UI development.

```bash
make build              # Build binary → bin/knowns
make dev                # Build with race detector
make test               # Run unit tests
make test-e2e           # Run CLI + MCP E2E tests
make test-e2e-semantic  # E2E tests including semantic search
make lint               # Run golangci-lint
make cross-compile      # Build for all 6 platforms
make ui                 # Rebuild embedded Web UI (requires pnpm)
```

### Project structure

```
cmd/knowns/          # CLI entry point
internal/
  cli/               # Cobra commands
  models/            # Domain models
  storage/           # File-based storage (.knowns/)
  server/            # HTTP server, SSE, WebSocket
    routes/          # REST API handlers
    workspace/       # Agent orchestrator, process manager, worktree
  mcp/               # MCP server (stdio)
  search/            # Semantic search (ONNX)
ui/                  # Embedded React UI (built assets)
tests/               # E2E tests
```

## Links

- [npm](https://www.npmjs.com/package/knowns)
- [GitHub](https://github.com/knowns-dev/knowns)
- [Discord](https://discord.knowns.dev)
- [Changelog](./CHANGELOG.md)

For design principles and long-term direction, see [Philosophy](./PHILOSOPHY.md).

For technical details, see [Architecture](./ARCHITECTURE.md) and [Contributing](./CONTRIBUTING.md).

---

<p align="center">
  <strong>What your AI should have knowns.</strong><br>
  Built for dev teams who pair with AI.
</p>
