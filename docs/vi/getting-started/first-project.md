# Dự án đầu tiên

Sau `knowns init`, nên làm 4 việc để project dùng được ngay:

1. tạo task
2. tạo 1-2 doc
3. kiểm tra search hoạt động
4. kết nối AI runtime đang dùng

## Ví dụ

```bash
knowns task create "Add authentication" \
  -d "JWT-based auth with login and register endpoints" \
  --ac "User can register with email/password" \
  --ac "User can login and receive JWT token"

knowns doc create "Auth Architecture" \
  -d "Authentication design decisions" \
  -f architecture

knowns search "authentication" --plain
knowns validate --plain
knowns browser --open
```

## Tại sao?

- Task cho AI mục tiêu cụ thể
- Doc cho AI context có cấu trúc thay vì giải thích rời rạc trong chat
- Search xác nhận retrieval đang chạy
- Validate kiểm tra cấu trúc project cơ bản

## Tiếp theo

- [Hướng dẫn sử dụng](../guides/user-guide.md)
- [MCP](../guides/mcp-integration.md)
