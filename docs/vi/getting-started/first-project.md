# Dự án đầu tiên

Sau khi chạy `knowns init`, bạn nên làm bốn việc sau để có một dự án dùng được ngay:

1. tạo task
2. tạo một hoặc hai tài liệu
3. xác nhận tìm kiếm hoạt động
4. kết nối các AI runtime mà bạn thực sự dùng

## Ví dụ một phiên làm việc

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

## Vì sao các bước này quan trọng?

- task cho AI một mục tiêu cụ thể để làm
- tài liệu cho AI một lớp ngữ cảnh rõ ràng thay vì giải thích rời rạc trong chat
- tìm kiếm giúp xác nhận cơ chế truy xuất cục bộ đang hoạt động
- validation kiểm tra xem cấu trúc cơ bản của dự án đã ổn chưa

## Đọc tiếp

- [Hướng dẫn sử dụng](../guides/user-guide.md)
- [Tích hợp MCP](../guides/mcp-integration.md)
