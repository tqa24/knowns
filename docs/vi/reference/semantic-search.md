# Semantic search

Semantic search giúp Knowns tìm theo ý nghĩa thay vì chỉ khớp từ khóa chính xác.

## Các lệnh chính

```bash
knowns model list
knowns model download multilingual-e5-small
knowns model set multilingual-e5-small
knowns search --status-check
knowns search --reindex
knowns search "how authentication works" --plain
```

## Các chế độ tìm kiếm

- `keyword`
- `semantic`
- `hybrid`

## Ghi chú vận hành

Nếu semantic components không sẵn sàng, các đường tìm kiếm liên quan có thể tự chuyển sang chế độ an toàn thay vì bị lỗi.
