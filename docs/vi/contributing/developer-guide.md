# Developer guide

Bắt đầu từ đây nếu muốn đóng góp cho Knowns.

## Đọc trước

1. `KNOWNS.md`
2. `README.md`
3. `docs/en/README.md`

## Thư mục quan trọng

- `internal/cli/`
- `internal/mcp/handlers/`
- `internal/search/`
- `internal/runtimeinstall/`
- `internal/codegen/`
- `internal/storage/`
- `tests/`
- `ui/`

## Lệnh hay dùng

```bash
go build -o ./bin/knowns ./cmd/knowns
go test ./...
go test ./internal/cli -count=1
go test ./tests -count=1
```

## Nguyên tắc

Khi behavior thay đổi, giữ code, tests, và docs đồng bộ trong cùng một pass.
