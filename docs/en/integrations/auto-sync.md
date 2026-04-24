# Auto Sync

Knowns uses `knowns sync` and `knowns update` to keep generated artifacts aligned with the current binary and project config.

## What gets synced

- instruction files
- skills
- MCP config
- platform-specific config
- git integration
- semantic setup and indexing

## Related commands

```bash
knowns sync
knowns sync --skills
knowns sync --instructions
knowns update
```

## Legacy note

If an older project still uses `.agent/skills`, Knowns continues syncing it for compatibility and may print a warning.
