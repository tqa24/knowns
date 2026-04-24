# Quản lý model

Knowns có thể dùng các embedding model cục bộ để chạy semantic search.

## Các lệnh chính

```bash
knowns model list
knowns model download multilingual-e5-small
knowns model set multilingual-e5-small
knowns model status
```

## Quy trình điển hình

1. liệt kê các model có sẵn
2. tải về một model
3. đặt model đó vào cấu hình của dự án
4. reindex nếu cần

## Các lệnh liên quan

```bash
knowns search --status-check
knowns search --reindex
```

## Vì sao phần này quan trọng?

Nếu không có model cục bộ, semantic search sẽ không hoạt động và Knowns sẽ dùng hành vi tìm kiếm theo từ khóa ở những nơi phù hợp.
