# Command Reference

Complete reference for all Knowns CLI commands.

## Task Commands

### `knowns task <id>` (Shorthand)

View a single task (shorthand for `knowns task view`).

```bash
knowns task <id> [options]
```

| Option    | Description                |
| --------- | -------------------------- |
| `--plain` | Plain text output (for AI) |

**Examples:**

```bash
knowns task 42 --plain
```

### `knowns task create`

Create a new task.

```bash
knowns task create "Title" [options]
```

| Option              | Description                       |
| ------------------- | --------------------------------- |
| `-d, --description` | Task description                  |
| `--ac`              | Acceptance criterion (repeatable) |
| `-l, --label`       | Task label (repeatable)           |
| `--priority`        | `low`, `medium`, `high`           |
| `-a, --assignee`    | Assignee (e.g., `@me`, `@john`)   |
| `--parent`          | Parent task ID for subtasks       |

**Examples:**

```bash
# Basic task
knowns task create "Fix login bug"

# Task with details
knowns task create "Add authentication" \
  -d "Implement JWT auth following @doc/patterns/auth" \
  --ac "User can login" \
  --ac "Session persists" \
  --priority high \
  -l feature \
  -l auth

# Subtask
knowns task create "Write unit tests" --parent 42
```

### `knowns task list`

List all tasks.

```bash
knowns task list [options]
```

| Option       | Description                |
| ------------ | -------------------------- |
| `--status`   | Filter by status           |
| `--priority` | Filter by priority         |
| `--assignee` | Filter by assignee         |
| `--label`    | Filter by label            |
| `--tree`     | Show as tree hierarchy     |
| `--plain`    | Plain text output (for AI) |

**Examples:**

```bash
knowns task list --plain
knowns task list --status in-progress --assignee @me
knowns task list --tree --plain
```

### `knowns task view`

View a single task (full command form).

```bash
knowns task view <id> [options]
```

| Option    | Description                |
| --------- | -------------------------- |
| `--plain` | Plain text output (for AI) |

### `knowns task edit`

Edit an existing task.

```bash
knowns task edit <id> [options]
```

| Option              | Description                                           |
| ------------------- | ----------------------------------------------------- |
| `-t, --title`       | New title                                             |
| `-d, --description` | New description                                       |
| `-s, --status`      | `todo`, `in-progress`, `in-review`, `blocked`, `done` |
| `--priority`        | `low`, `medium`, `high`                               |
| `-a, --assignee`    | Assignee                                              |
| `--labels`          | Labels (comma-separated, replaces existing)           |
| `--ac`              | Add acceptance criterion                              |
| `--check-ac`        | Check criterion (1-indexed)                           |
| `--uncheck-ac`      | Uncheck criterion                                     |
| `--remove-ac`       | Remove criterion                                      |
| `--plan`            | Set implementation plan                               |
| `--notes`           | Set implementation notes                              |
| `--append-notes`    | Append to notes                                       |

**Examples:**

```bash
# Change status and assignee
knowns task edit 42 -s in-progress -a @me

# Check acceptance criteria
knowns task edit 42 --check-ac 1 --check-ac 2

# Add implementation plan
knowns task edit 42 --plan $'1. Research\n2. Implement\n3. Test'

# Add notes progressively
knowns task edit 42 --append-notes "Completed auth middleware"
```

---

## Documentation Commands

### `knowns doc <path>` (Shorthand)

View a document (shorthand for `knowns doc view`).

```bash
knowns doc <name-or-path> [options]
```

| Option              | Description                                      |
| ------------------- | ------------------------------------------------ |
| `--plain`           | Plain text output (for AI)                       |
| `--info`            | Show document stats (size, tokens, headings)     |
| `--toc`             | Show table of contents only                      |
| `--section <title>` | Show specific section by heading title or number |

**Examples:**

```bash
knowns doc "README" --plain
knowns doc "patterns/auth" --plain

# For large documents - check size first
knowns doc "README" --info --plain
# Output: Size: 42,461 chars (~12,132 tokens) | Headings: 83

# Get table of contents
knowns doc "README" --toc --plain

# Read specific section
knowns doc "README" --section "2. Installation" --plain
knowns doc "README" --section "2" --plain  # By number
```

### `knowns doc create`

Create a new document.

```bash
knowns doc create "Title" [options]
```

| Option              | Description                                        |
| ------------------- | -------------------------------------------------- |
| `-d, --description` | Document description                               |
| `-t, --tags`        | Comma-separated tags                               |
| `-f, --folder`      | Folder path (e.g., `patterns`, `architecture/api`) |

**Examples:**

```bash
# Simple doc
knowns doc create "API Guidelines" -d "REST API conventions"

# Doc in folder
knowns doc create "Auth Pattern" \
  -d "JWT authentication pattern" \
  -t "patterns,security" \
  -f patterns
```

### `knowns doc list`

List all documents.

```bash
knowns doc list [options]
```

| Option    | Description                                      |
| --------- | ------------------------------------------------ |
| `--tag`   | Filter by tag                                    |
| `--plain` | Plain text output (tree format, token-efficient) |

**Examples:**

```bash
# List all docs
knowns doc list

# Filter by tag
knowns doc list --tag architecture
```

### `knowns doc view`

View a document.

```bash
knowns doc view <name-or-path> [options]
```

| Option              | Description                                      |
| ------------------- | ------------------------------------------------ |
| `--plain`           | Plain text output                                |
| `--info`            | Show document stats (size, tokens, headings)     |
| `--toc`             | Show table of contents only                      |
| `--section <title>` | Show specific section by heading title or number |

**Examples:**

```bash
knowns doc view "auth-pattern" --plain
knowns doc view "patterns/auth-pattern" --plain
knowns doc view "README" --info --plain      # Check size first
knowns doc view "README" --toc --plain       # Get TOC
knowns doc view "README" --section "2" --plain  # Read section
```

### `knowns doc edit`

Edit a document.

```bash
knowns doc edit <name-or-path> [options]
```

| Option                  | Description                                              |
| ----------------------- | -------------------------------------------------------- |
| `-t, --title`           | New title                                                |
| `--tags`                | New tags                                                 |
| `-c, --content`         | Replace content (or section content if `--section` used) |
| `-a, --append`          | Append to content                                        |
| `--section <title>`     | Target section to replace (use with `-c`)                |

**Examples:**

```bash
# Edit content directly
knowns doc edit "README" -c "New content here"

# Append content
knowns doc edit "README" -a "## New Section"

# Edit specific section only (context-efficient!)
knowns doc edit "README" --section "2. Installation" -c "New section content"
knowns doc edit "README" --section "2" -c "New content"  # By number
```

---

## Time Tracking Commands

### `knowns time start`

Start tracking time on a task.

```bash
knowns time start <task-id>
```

### `knowns time stop`

Stop the current timer.

```bash
knowns time stop
```

### `knowns time pause` / `knowns time resume`

Pause or resume the current timer.

```bash
knowns time pause
knowns time resume
```

### `knowns time status`

Show current timer status.

```bash
knowns time status
```

### `knowns time add`

Add manual time entry.

```bash
knowns time add <task-id> <duration> [options]
```

| Option       | Description       |
| ------------ | ----------------- |
| `-n, --note` | Note for entry    |
| `-d, --date` | Date (YYYY-MM-DD) |

**Examples:**

```bash
knowns time add 42 2h -n "Code review"
knowns time add 42 30m -d "2025-01-15"
```

### `knowns time report`

Generate time report.

```bash
knowns time report [options]
```

| Option       | Description             |
| ------------ | ----------------------- |
| `--from`     | Start date (YYYY-MM-DD) |
| `--to`       | End date (YYYY-MM-DD)   |
| `--by-label` | Group by label          |
| `--csv`      | CSV output              |

---

## Memory Commands

### `knowns memory list`

List memory entries.

```bash
knowns memory list [options]
```

| Option     | Description                |
| ---------- | -------------------------- |
| `--layer`  | Filter by layer (`project`, `global`) |
| `--category` | Filter by category       |
| `--tag`    | Filter by tag              |
| `--plain`  | Plain text output          |

### `knowns memory view`

View a memory entry.

```bash
knowns memory view <id> [options]
```

| Option    | Description                |
| --------- | -------------------------- |
| `--plain` | Plain text output          |

### `knowns memory add`

Add a memory entry.

```bash
knowns memory add [options]
```

| Option       | Description                          |
| ------------ | ------------------------------------ |
| `-t, --title` | Memory title                        |
| `-c, --content` | Memory content                    |
| `--layer`    | `project` (default) or `global`      |
| `--category` | Category (pattern, decision, etc.)   |
| `--tags`     | Comma-separated tags                 |

### `knowns memory promote`

Promote a memory entry up one layer (working→project→global).

```bash
knowns memory promote <id>
```

### `knowns memory demote`

Demote a memory entry down one layer (global→project→working).

```bash
knowns memory demote <id>
```

---

## Validate Command

### `knowns validate`

Validate references and file integrity across tasks and docs.

```bash
knowns validate [options]
```

| Option    | Description                                    |
| --------- | ---------------------------------------------- |
| `--scope` | `all`, `tasks`, `docs`, `templates`, or `sdd`  |
| `--fix`   | Attempt to auto-fix issues                     |
| `--entity`| Validate a specific task or doc only            |
| `--strict`| Treat warnings as errors                        |
| `--plain` | Plain text output                              |

**What it checks:**

- Broken `@doc/...` references
- Broken `@task-...` references
- File format integrity
- Missing required fields

**Examples:**

```bash
# Validate everything
knowns validate

# Validate only tasks
knowns validate --scope tasks

# Validate with plain output (for AI)
knowns validate --plain
```

---

## Search Commands

### `knowns search`

Search tasks, documentation, and memories.

```bash
knowns search <query> [options]
```

| Option              | Description                                  |
| ------------------- | -------------------------------------------- |
| `--type`            | `all`, `task`, `doc`, or `memory`            |
| `--status`          | Filter tasks by status                       |
| `--priority`        | Filter tasks by priority                     |
| `--label`           | Filter tasks by label                        |
| `--tag`             | Filter docs or memories by tag               |
| `--assignee`        | Filter tasks by assignee                     |
| `--keyword`         | Force keyword-only search (skip semantic)    |
| `--limit`           | Limit search results (default: 20)           |
| `--reindex`         | Rebuild the search index                     |
| `--setup`           | Set up semantic search                       |
| `--status-check`    | Show semantic search status                  |
| `--install-runtime` | Download and install ONNX Runtime (also run by installers) |
| `--plain`           | Plain text output                            |
| `--json`            | JSON output                                  |

**Examples:**

```bash
knowns search "authentication" --type doc --plain
knowns search "login bug" --type task --status in-progress
knowns search "auth pattern" --keyword --limit 5
knowns search --status-check
knowns search --reindex
knowns search --install-runtime
```

### `knowns retrieve`

Retrieve ranked context across docs, tasks, and memories with citations and context-pack assembly.

```bash
knowns retrieve <query> [options]
```

| Option                | Description                                      |
| --------------------- | ------------------------------------------------ |
| `--status`            | Filter tasks by status                           |
| `--priority`          | Filter tasks by priority                         |
| `--label`             | Filter tasks by label                            |
| `--tag`               | Filter docs or memories by tag                   |
| `--assignee`          | Filter tasks by assignee                         |
| `--keyword`           | Force keyword-only retrieval                     |
| `--expand-references` | Expand @doc/@task/@memory references into result |
| `--source-types`      | Comma-separated source types: `doc,task,memory`  |
| `--limit`             | Limit ranked candidates (default: 20)            |
| `--plain`             | Plain text output                                |
| `--json`              | JSON output                                      |

**Examples:**

```bash
knowns retrieve "error handling patterns" --json
knowns retrieve "auth" --expand-references --source-types doc,memory
knowns retrieve "api design" --limit 10 --plain
```

---

## Model Commands

Manage embedding models for semantic search.

### `knowns model` (Shorthand)

Show current model status (shorthand for `knowns model status`).

```bash
knowns model
```

### `knowns model list`

List available embedding models.

```bash
knowns model list
```

**Output shows:**
- Model ID and name
- Quality tier (Fast, Balanced, Quality)
- HuggingFace ID
- Dimensions and max tokens
- Download status
- Recommended models marked with ★

**Examples:**

```bash
knowns model list
```

### `knowns model download`

Download an embedding model.

```bash
knowns model download <model-id>
```

| Argument     | Description             |
| ------------ | ----------------------- |
| `<model-id>` | Built-in model ID       |

**Built-in models:**

| Model ID            | Quality   | Dimensions | Size    |
| ------------------- | --------- | ---------- | ------- |
| `gte-small` ★       | Balanced  | 384        | ~50MB   |
| `all-MiniLM-L6-v2`  | Fast      | 384        | ~45MB   |
| `gte-base`          | Quality   | 768        | ~110MB  |
| `bge-small-en-v1.5` | Balanced  | 384        | ~50MB   |
| `bge-base-en-v1.5`  | Quality   | 768        | ~110MB  |
| `e5-small-v2`       | Balanced  | 384        | ~50MB   |

**Examples:**

```bash
# Download default recommended model
knowns model download gte-small

# Download higher quality model
knowns model download gte-base

```

### `knowns model set`

Set the embedding model for the current project.

```bash
knowns model set <model-id>
```

| Argument     | Description                |
| ------------ | -------------------------- |
| `<model-id>` | Model ID to use            |

**Note:** If the model is not downloaded, it will be downloaded automatically.

**Examples:**

```bash
# Set model for current project
knowns model set gte-small

# Set higher quality model
knowns model set gte-base
```

**After setting a new model:**

```bash
# Rebuild search index with new model
knowns search --reindex
```

### `knowns model status`

Show detailed status of models and current project configuration.

```bash
knowns model status
```

**Output shows:**
- Global models directory location
- Number of downloaded models
- Total disk usage
- List of downloaded models with sizes
- Current project's model configuration

### `knowns model remove`

Remove a downloaded embedding model.

```bash
knowns model remove <model-id>
```

| Argument     | Description             |
| ------------ | ----------------------- |
| `<model-id>` | Downloaded model ID     |

**Examples:**

```bash
# Remove a downloaded model
knowns model remove gte-small
```

---

## Code Intelligence Commands

### `knowns code`

Code intelligence commands for AST-based indexing and graph analysis.

```bash
knowns code <command>
```

**Available subcommands:**

- `knowns code ingest` - index supported code files into the local search/code graph
- `knowns code watch` - watch code files and auto-index on changes
- `knowns code search <query>` - search indexed code with optional neighbor expansion
- `knowns code deps` - inspect indexed dependency edges such as `calls` or `imports`
- `knowns code symbols` - inspect indexed symbols by file or kind

### `knowns code ingest`

Index code files using AST-based code intelligence.

```bash
knowns code ingest [options]
```

| Option            | Description                                          |
| ----------------- | ---------------------------------------------------- |
| `--dry-run`       | Preview what would be indexed without writing to disk |
| `--include-tests` | Include test files in the code index                 |

**Examples:**

```bash
knowns code ingest
knowns code ingest --dry-run
knowns code ingest --include-tests
```

### `knowns code watch`

Watch code files and auto-index on changes.

```bash
knowns code watch [options]
```

| Option         | Description                           |
| -------------- | ------------------------------------- |
| `--debounce`   | Debounce delay in milliseconds        |

**Examples:**

```bash
knowns code watch
knowns code watch --debounce 500
```

### `knowns code search`

Search indexed code and optionally expand related neighbors from the code graph.

```bash
knowns code search <query> [options]
```

| Option           | Description                              |
| ---------------- | ---------------------------------------- |
| `--limit`        | Limit direct code matches                |
| `--neighbors`    | Max neighbors per match (1-hop)          |
| `--edge-types`   | Comma-separated edge types to expand     |
| `--keyword`      | Force keyword-only code search           |
| `--show-snippet` | Show snippet/preview for each match      |

**Examples:**

```bash
knowns code search "oauth login"
knowns code search backfill --neighbors 5 --edge-types calls,imports
knowns code search token --keyword --show-snippet
```

### `knowns code deps`

Inspect indexed code dependency data.

```bash
knowns code deps [options]
```

| Option      | Description                  |
| ----------- | ---------------------------- |
| `--type`    | Filter dependency edge type  |
| `--limit`   | Limit dependency results     |

**Examples:**

```bash
knowns code deps
knowns code deps --type calls
```

### `knowns code symbols`

Inspect indexed code symbols.

```bash
knowns code symbols [options]
```

| Option      | Description                  |
| ----------- | ---------------------------- |
| `--path`    | Filter symbols by file path  |
| `--kind`    | Filter symbols by kind       |
| `--limit`   | Limit symbol results         |

**Examples:**

```bash
knowns code symbols
knowns code symbols --path internal/server/server.go
knowns code symbols --kind function
```

---

## Template Commands

### `knowns template list`

List available templates.

```bash
knowns template list [options]
```

| Option    | Description       |
| --------- | ----------------- |
| `--plain` | Plain text output |

**Examples:**

```bash
knowns template list
knowns template list --plain
```

### `knowns template run`

Run a template to generate files.

```bash
knowns template run <name> [options]
```

| Option            | Description                                 |
| ----------------- | ------------------------------------------- |
| `--dry-run`       | Preview without creating files              |
| `-v, --var`       | Template variable as `key=value` (repeatable) |

**Examples:**

```bash
# Interactive mode
knowns template run react-component

# With explicit variables
knowns template run react-component -v name=UserProfile

# Preview only
knowns template run react-component --dry-run
```

### `knowns template view`

View template details.

```bash
knowns template view <name>
```

**Examples:**

```bash
knowns template view react-component
knowns template view react-component --plain
```

### `knowns template create`

Create a new template.

```bash
knowns template create <name> [options]
```

| Option              | Description                |
| ------------------- | -------------------------- |
| `-d, --description` | Template description       |
| `--doc`             | Link to existing doc path  |

**Examples:**

```bash
# Basic template
knowns template create my-component

# With description and linked doc
knowns template create api-service \
  -d "REST API service class" \
  --doc patterns/api-service
```

---

## Other Commands

### `knowns update`

Update the Knowns CLI to the latest version and sync project configs.

```bash
knowns update [options]
```

| Option    | Description                              |
| --------- | ---------------------------------------- |
| `--check` | Only check for updates without installing |

This command:

1. Checks the npm registry for the latest version
2. Detects how Knowns was installed (Homebrew, npm, shell script, etc.)
3. Runs the appropriate upgrade command
4. Syncs MCP configs (`.mcp.json`, `.kiro/settings/mcp.json`, `opencode.json`) to use the local binary

For script-managed installs (`~/.knowns/bin/`), the binary is downloaded and replaced directly.

**Examples:**

```bash
# Check for updates
knowns update --check

# Update and sync
knowns update
```

### `knowns init`

Initialize Knowns in current directory with interactive wizard.

**Requirement:** Git must be initialized first (`git init`).

```bash
knowns init [project-name] [options]
```

| Option        | Description                               |
| ------------- | ----------------------------------------- |
| `--wizard`    | Force interactive wizard mode             |
| `--no-wizard` | Skip wizard, use defaults                 |
| `-f, --force` | Reinitialize (overwrites existing config) |
| `--open`      | Launch Chat UI immediately after init     |
| `--no-open`   | Skip the Chat UI launch after init        |
| `--git-tracked` | Track .knowns/ files in git             |
| `--git-ignored` | Add .knowns/ to .gitignore              |

**Examples:**

```bash
# Interactive wizard (default when no name provided)
knowns init

# Quick init with name
knowns init my-project

# Force reinitialize
knowns init --force
```

**Wizard prompts (each shown separately):**

1. Project name
2. Git tracking mode (`git-tracked`, `git-ignored`, or `none`)
3. AI platforms to integrate (multi-select: `claude-code`, `opencode`, `codex`, `kiro`, `gemini`, `copilot`, `agents`)
4. Enable Chat UI
5. Enable semantic search
6. Select embedding model (if semantic enabled)

When using `--force`, all fields are pre-populated from existing `config.json`.

**When MCP is selected:**

- Automatically creates `.mcp.json` for Claude Code auto-discovery

**Git Tracking Modes:**

| Mode          | Description                                                 |
| ------------- | ----------------------------------------------------------- |
| `git-tracked` | All `.knowns/` files tracked in git (recommended for teams) |
| `git-ignored` | `config.json`, docs, and templates tracked; tasks stay local |
| `none`        | No `.gitignore` changes; you manage manually |

Both `git-tracked` and `git-ignored` allow `config.json` to be pushed, enabling `knowns sync` after cloning.

### `knowns config`

Manage project configuration.

```bash
knowns config <command> [key] [value]
```

**Commands:**

```bash
# Get a config value
knowns config get defaultAssignee --plain

# Set a config value
knowns config set defaultAssignee "@john"

# List all config
knowns config list
```

### `knowns browser`

Start the Knowns web UI server.

```bash
knowns browser [options]
```

| Option       | Description                              |
| ------------ | ---------------------------------------- |
| `--dev`      | Enable development mode                  |
| `--open`     | Open browser after starting              |
| `--no-open`  | Don't automatically open browser         |
| `--port`     | Custom port (default: `6420`; tries next ports if busy) |
| `--restart`  | Restart server if already running        |
| `--project`  | Open a specific project path directly    |
| `--scan`     | Comma-separated directories to scan for projects |
| `--watch`    | Enable code watcher for auto-indexing    |

**Examples:**

```bash
# Start server only
knowns browser

# Start and open browser
knowns browser --open

# Start outside a repo and scan common workspaces
knowns browser --scan ~/Workspaces,~/Projects --open

# Open a specific project directly
knowns browser --project ~/Workspaces/my-app --open

# Run browser with code auto-indexing enabled
knowns browser --watch
```

### `knowns mcp`

Start the MCP server.

```bash
knowns mcp [options]
```

| Option    | Description                  |
| --------- | ---------------------------- |
| `--stdio` | Use stdio transport          |

**Examples:**

```bash
# Start server over stdio (for MCP clients)
knowns mcp --stdio
```

### `knowns mcp setup`

Setup Knowns MCP server in Claude Code.

```bash
knowns mcp setup [options]
```

This command currently has no documented flags in the CLI help.

### `knowns sync`

Apply project configuration from `.knowns/config.json`. Recommended after cloning a repo with Knowns.

```bash
knowns sync [options]
```

| Option           | Description                                                 |
| ---------------- | ----------------------------------------------------------- |
| `--force`        | Force resync (deprecated — sync always overwrites) |
| `--skills`       | Sync skills only                                            |
| `--instructions` | Sync instruction files only                                 |
| `--model`        | Download embedding model only                               |
| `--platform`     | Sync a specific platform (`claude`, `gemini`, `copilot`, `agents`) |

**What it does (in order):**

1. Skills — copies built-in skills to platform directories
2. Instructions — generates agent instruction files for configured platforms
3. Git integration — applies `.gitignore` rules based on `gitTrackingMode`
4. Model download — downloads configured embedding model if not installed
5. Search index — rebuilds the semantic search index
6. MCP configs — syncs MCP config files to use the local binary

**After cloning a repo:**

```bash
git clone <repo>
knowns sync        # sets up everything from config.json
```

**Supported instruction files:**

| File                              | Description              |
| --------------------------------- | ------------------------ |
| `CLAUDE.md`                       | Claude Code instructions |
| `OPENCODE.md`                     | OpenCode instructions    |
| `GEMINI.md`                       | Gemini CLI instructions  |
| `AGENTS.md`                       | Generic AI instructions  |
| `.github/copilot-instructions.md` | GitHub Copilot           |

**Examples:**

```bash
# Full sync (skills + instructions + model + reindex)
knowns sync

# Sync only skills
knowns sync --skills

# Download model only
knowns sync --model

# Sync only instruction files for Claude
knowns sync --instructions --platform claude

# Force overwrite everything
knowns sync --force
```

---

## Output Formats

### `--plain`

Plain text output optimized for AI consumption. Always use this when working with AI assistants.

```bash
knowns task 42 --plain
knowns doc list --plain
knowns search "auth" --plain
```

---

## Multi-line Input

### Bash / Zsh

```bash
knowns task edit 42 --plan $'1. Step one\n2. Step two\n3. Step three'
```

### PowerShell

```powershell
knowns task edit 42 --notes "Line 1`nLine 2`nLine 3"
```

### Heredoc (long content)

```bash
knowns task edit 42 --plan "$(cat <<EOF
1. Research existing patterns
2. Design solution
3. Implement
4. Write tests
5. Update documentation
EOF
)"
```
