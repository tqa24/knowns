# Web UI Guide

Knowns includes a local browser UI for tasks, docs, workspaces, code graph exploration, and chat.

---

## Start the UI

```bash
# Start server only
knowns browser

# Start and open browser
knowns browser --open

# Open a specific project directly
knowns browser --project ~/Workspaces/my-app --open

# Scan project folders when starting outside a repo
knowns browser --scan ~/Workspaces,~/Projects --open

# Enable background code auto-indexing while the UI is running
knowns browser --watch
```

Default port is `3001` unless overridden by `settings.serverPort` or `--port`.

---

## Browser Flags

```bash
knowns browser --port 3002
knowns browser --no-open
knowns browser --restart
knowns browser --dev
knowns browser --project ~/Workspaces/my-app
knowns browser --scan ~/Workspaces,~/Projects
knowns browser --watch
```

---

## What the UI Covers

- task board and task details
- document browsing and markdown rendering
- workspace picker and project switching
- knowledge graph visualization (tasks, docs, memories, and optional code relationships)
- memory management (3-layer: working, project, global)
- search and navigation shortcuts
- chat UI with timeline/history navigation and runtime status
- real-time updates from local project data

### Workspace Switching

You can launch `knowns browser` from outside any repo. In that mode, the UI can:

- list known workspaces from the local registry
- scan common folders or user-provided paths for `.knowns/` projects
- switch between projects without restarting the server

This is useful when you work across multiple repos and want one Knowns UI session to move between them.

### Knowledge Graph

The `/graph` page visualizes relationships between tasks, docs, memories, and indexed code using Cytoscape.js:

- **Node types**: tasks (circle), docs (rounded rectangle), memories (hexagon), code (when indexed)
- **Edge types**: parent (solid), spec (dashed), mention (dotted), code dependency edges (when code graph is enabled)
- **Features**: search highlighting, impact analysis (2-hop BFS), cluster detection
- **Interactions**: hover to highlight neighbors, drag nodes, click for details

If you have run `knowns code ingest`, the graph can include indexed code symbols and relationships in addition to the usual knowledge graph entities.

### Chat and Runtime Status

The chat page includes runtime-aware improvements shipped in `v0.18.0`:

- clearer rendering for tool-heavy assistant messages
- a timeline/history dialog for jumping through long sessions
- runtime status surfaces for OpenCode readiness and degraded states
- better multi-session and sidebar behavior for longer conversations

The exact page layout can evolve, but the browser UI is powered by the local Knowns server and reads from your current project.

---

## Real-Time Behavior

The browser connects to the local Go server and updates when project data changes.

- CLI edits can appear in the UI without restarting the server
- reconnect behavior handles normal local restarts and sleep/wake flows
- all data remains local to your machine unless you choose to sync it elsewhere

---

## Troubleshooting

### Port already in use

```bash
knowns browser --port 3002
```

### Browser does not open automatically

```bash
knowns browser --open
```

### Existing server instance is stale

```bash
knowns browser --restart
```

---

## Related

- [Configuration](./configuration.md) - `settings.serverPort`
- [User Guide](./user-guide.md) - Day-to-day usage
- [Command Reference](./commands.md) - Current browser flags
