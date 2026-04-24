# Developer Guide

Start here if you are contributing to Knowns.

## Read first

1. `KNOWNS.md`
2. `README.md`
3. `docs/en/README.md`

## Important directories

- `internal/cli/`
- `internal/mcp/handlers/`
- `internal/search/`
- `internal/runtimeinstall/`
- `internal/codegen/`
- `internal/storage/`
- `tests/`
- `ui/`

## Useful commands

```bash
go build -o ./bin/knowns ./cmd/knowns
go test ./...
go test ./internal/cli -count=1
go test ./tests -count=1
```

## Working rule of thumb

When behavior changes, keep code, tests, and docs aligned in the same pass whenever possible.
