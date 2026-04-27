# Quick start

Cách nhanh nhất để có một project Knowns chạy được.

## 1. Init project

```bash
knowns init
# hoặc không cài global:
npx knowns init
```

Init wizard cho phép cấu hình:

- tên project
- git tracking mode
- AI platforms cần tích hợp
- Chat UI qua OpenCode (tùy chọn)
- semantic search
- embedding model

Nếu terminal quá hẹp, Knowns tự chuyển sang non-interactive defaults.

## 2. Tạo task

```bash
knowns task create "Setup project" -d "Init project với Knowns"
```

## 3. Tạo doc

```bash
knowns doc create "Architecture" -d "Tổng quan hệ thống" -f architecture
```

## 4. Mở Web UI

```bash
knowns browser --open
```

## 5. Sync khi cần

```bash
knowns sync
knowns update
```

Chạy `knowns sync` sau khi clone repo, đổi platform, hoặc update CLI.

## 6. Mở lại Web UI

```bash
knowns browser --open
```

## Tiếp theo

- [Dự án đầu tiên](./first-project.md)
- [Hướng dẫn sử dụng](../guides/user-guide.md)
- [Workflow](../guides/workflow.md)
