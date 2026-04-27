# Hướng dẫn sử dụng

Dành cho người dùng Knowns trong project thực tế, không chỉ thử CLI một lần.

## Mô hình chính

Knowns là một context layer cho project, gồm 5 phần gắn với nhau:

- task
- doc
- memory
- template
- search / retrieval

CLI, MCP server, và Web UI đều thao tác trên cùng một project state.

## `knowns init` làm gì?

- Chạy interactive wizard
- Cài OpenCode nếu cần
- Sau wizard:
  - tạo cấu trúc project
  - apply config
  - sync skills
  - tạo MCP/config files
  - cài runtime hooks
  - build semantic index

## Terminal

- Terminal quá hẹp → Knowns chuyển sang non-interactive defaults
- Wizard dùng alternate screen để giảm lỗi hiển thị khi resize
- Output từ installer bên thứ ba có thể khá ồn

## Thao tác hằng ngày

### Task

```bash
knowns task create "Add authentication" -d "JWT-based auth"
knowns task edit <id> -s in-progress
knowns task edit <id> --check-ac 1
knowns task edit <id> --append-notes "Completed auth middleware"
```

### Doc

```bash
knowns doc create "Auth Architecture" -d "Design overview" -f architecture
knowns doc "architecture/auth-architecture" --plain
knowns doc "architecture/auth-architecture" --toc --plain
```

### Search

```bash
knowns search "authentication" --plain
knowns retrieve "how auth works" --json
```

### Validate

```bash
knowns validate --plain
```

### Sync

```bash
knowns sync
```

## Tiếp theo

- [Workflow](./workflow.md)
- [MCP](./mcp-integration.md)
- [Lệnh](../reference/commands.md)
