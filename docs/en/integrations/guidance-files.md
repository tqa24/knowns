# Guidance Files

Knowns uses one canonical file plus several compatibility files for AI runtimes.

## Canonical file

- `KNOWNS.md`

## Compatibility files

- `CLAUDE.md`
- `OPENCODE.md`
- `GEMINI.md`
- `AGENTS.md`
- `.github/copilot-instructions.md`

## Refresh generated content

```bash
knowns sync
knowns sync --instructions
knowns agents --sync
```

There is no standalone `knowns guidelines` CLI command anymore.
