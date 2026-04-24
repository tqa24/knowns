# Cách làm việc đề xuất

Đây là cách làm việc được khuyến nghị khi kết hợp con người và AI với Knowns.

## Vòng lặp đề xuất

1. chạy `knowns init` một lần cho mỗi dự án
2. tạo task và tài liệu hỗ trợ
3. để AI đọc `KNOWNS.md`, task, tài liệu, và memory
4. thực hiện thay đổi
5. validate
6. sync hoặc cập nhật các tệp được tạo khi cần

## Các lệnh thường đi cùng nhau

```bash
knowns task create "..."
knowns doc create "..."
knowns search "..." --plain
knowns retrieve "..." --json
knowns validate --plain
knowns sync
```

## Vì sao cách này hiệu quả?

- task định nghĩa mục tiêu cần thực hiện
- tài liệu giải thích cấu trúc và mục đích
- memory giữ lại quyết định và convention
- retrieval nối tất cả lại cho cả con người lẫn AI

## Khi nào nên dùng từng cách tương tác

- CLI: thao tác nhanh, dễ script hóa
- MCP: tích hợp AI có cấu trúc
- giao diện web: phù hợp cho các luồng bảng công việc, tài liệu, graph, và chat
