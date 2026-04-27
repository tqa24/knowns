# Workflow

Cách làm việc đề xuất khi kết hợp người và AI với Knowns.

## Vòng lặp chính

1. `knowns init` một lần cho mỗi project
2. Tạo task và doc hỗ trợ
3. AI đọc `KNOWNS.md`, task, doc, memory
4. Implement
5. Validate
6. Sync khi cần

## Lệnh hay đi cùng nhau

```bash
knowns task create "..."
knowns doc create "..."
knowns search "..." --plain
knowns retrieve "..." --json
knowns validate --plain
knowns sync
```

## Tại sao cách này hiệu quả?

- Task define mục tiêu
- Doc giải thích cấu trúc và intent
- Memory giữ lại decision và convention
- Retrieval nối tất cả lại cho cả người và AI

## Khi nào dùng gì

- **CLI**: thao tác nhanh, dễ script
- **MCP**: structured AI integration
- **Web UI**: board, docs, graph, chat
