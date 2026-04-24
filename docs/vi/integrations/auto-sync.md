# Tự động đồng bộ

Knowns dùng `knowns sync` và `knowns update` để giữ các tệp được tạo luôn đồng bộ với binary và cấu hình dự án hiện tại.

## Những gì được đồng bộ

- instruction files
- skills
- MCP config
- cấu hình riêng theo từng nền tảng
- git integration
- semantic setup và indexing

## Các lệnh liên quan

```bash
knowns sync
knowns sync --skills
knowns sync --instructions
knowns update
```

## Ghi chú về tương thích cũ

Nếu một dự án cũ vẫn dùng `.agent/skills`, Knowns sẽ tiếp tục đồng bộ để giữ khả năng tương thích và có thể in warning nhẹ.
