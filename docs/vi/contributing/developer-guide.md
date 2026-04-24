# Hướng dẫn cho người đóng góp

Hãy bắt đầu từ đây nếu bạn đang đóng góp cho Knowns.

## Đọc trước

1. `KNOWNS.md`
2. `README.md`
3. `docs/en/README.md`

## Các thư mục quan trọng

- `internal/cli/`
- `internal/mcp/handlers/`
- `internal/search/`
- `internal/runtimeinstall/`
- `internal/codegen/`
- `internal/storage/`
- `tests/`
- `ui/`

## Các lệnh hữu ích

```bash
go build -o ./bin/knowns ./cmd/knowns
go test ./...
go test ./internal/cli -count=1
go test ./tests -count=1
```

## Nguyên tắc thực tế

Khi behavior thay đổi, hãy cố giữ code, tests, và docs đồng bộ trong cùng một pass.
