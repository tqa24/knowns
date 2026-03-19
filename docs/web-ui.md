# Web UI Guide

Knowns includes a local browser UI for tasks and docs.

---

## Start the UI

```bash
# Start server only
knowns browser

# Start and open browser
knowns browser --open
```

Default port is `3001` unless overridden by `settings.serverPort` or `--port`.

---

## Browser Flags

```bash
knowns browser --port 3002
knowns browser --no-open
knowns browser --restart
knowns browser --dev
```

---

## What the UI Covers

- task board and task details
- document browsing and markdown rendering
- search and navigation shortcuts
- real-time updates from local project data

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
