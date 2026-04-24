# Sync

`knowns sync` re-applies `.knowns/config.json` to the current machine.

## When to use it

Use `knowns sync` after:

- cloning a repository with existing `.knowns/`
- changing selected platforms
- upgrading the CLI
- wanting generated files to match config again

## Common forms

```bash
knowns sync
knowns sync --skills
knowns sync --instructions
knowns sync --model
knowns sync --instructions --platform claude
knowns sync --instructions --platform cursor
```

## What it can refresh

- skills
- instruction files
- MCP config
- platform-specific config
- git integration
- semantic-search setup
- search indexes where relevant flows apply

## Related

- [Configuration](./configuration.md)
- [Compatibility](../integrations/compatibility.md)
- [Auto Sync](../integrations/auto-sync.md)
