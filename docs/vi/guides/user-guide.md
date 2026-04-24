# Hướng dẫn sử dụng

Tài liệu này dành cho người dùng Knowns trong một dự án thực tế, không chỉ để thử CLI một lần.

## Mô hình cốt lõi

Knowns dễ hiểu nhất khi bạn xem nó như một lớp ngữ cảnh cho dự án với năm phần gắn với nhau:

- task
- tài liệu
- memory
- template
- tìm kiếm / truy xuất

CLI, MCP server, và giao diện web đều thao tác trên cùng một trạng thái của dự án.

## Bạn sẽ thấy gì trong `knowns init`

- wizard tương tác
- output từ installer nếu OpenCode cần được cài hoặc cập nhật
- các bước sau wizard như:
  - tạo cấu trúc dự án
  - áp dụng cấu hình
  - đồng bộ skills
  - tạo tệp MCP/cấu hình
  - cài hook tích hợp runtime
  - xây dựng semantic index

## Hành vi terminal

- nếu terminal quá hẹp, Knowns có thể chuyển sang mặc định không tương tác
- wizard dùng alternate screen để giảm lỗi hiển thị khi thay đổi kích thước terminal
- output từ installer bên thứ ba vẫn có thể khá ồn

## Các thao tác thường dùng hằng ngày

### Tạo và cập nhật task

```bash
knowns task create "Add authentication" -d "JWT-based auth"
knowns task edit <id> -s in-progress
knowns task edit <id> --check-ac 1
knowns task edit <id> --append-notes "Completed auth middleware"
```

### Tạo và đọc tài liệu

```bash
knowns doc create "Auth Architecture" -d "Design overview" -f architecture
knowns doc "architecture/auth-architecture" --plain
knowns doc "architecture/auth-architecture" --toc --plain
```

### Tìm ngữ cảnh

```bash
knowns search "authentication" --plain
knowns retrieve "how auth works" --json
```

### Validate trước khi chốt

```bash
knowns validate --plain
```

### Giữ các tệp được tạo luôn đồng bộ

```bash
knowns sync
```

## Nên đọc tiếp

- [Cách làm việc đề xuất](./workflow.md)
- [Tích hợp MCP](./mcp-integration.md)
- [Tra cứu lệnh](../reference/commands.md)
