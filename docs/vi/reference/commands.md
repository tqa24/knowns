# Tra cứu lệnh

Hãy dùng `knowns <command> --help` để xem cú pháp chính xác. Trang này là bản tổng quan thực dụng.

## Các lệnh chính

```bash
knowns init
knowns sync
knowns update
knowns browser --open
```

## Quản lý task

```bash
knowns task create "Title" -d "Description"
knowns task list --plain
knowns task <id> --plain
knowns task edit <id> -s in-progress
knowns task edit <id> --check-ac 1
knowns task edit <id> --append-notes "Done"
```

## Doc

```bash
knowns doc create "API Guidelines" -d "REST conventions" -f architecture
knowns doc list --plain
knowns doc "architecture/api-guidelines" --plain
knowns doc edit "architecture/api-guidelines" -a "\n\n## Notes\n..."
```

## Tìm kiếm và truy xuất

```bash
knowns search "authentication" --plain
knowns search "jwt" --keyword --plain
knowns retrieve "how auth works" --json
knowns resolve "@doc/specs/auth{implements}" --plain
```

## Validation và theo dõi thời gian

```bash
knowns validate --plain
knowns time start <task-id>
knowns time stop
knowns time report
```

## Model và các lệnh semantic search

```bash
knowns model list
knowns model download multilingual-e5-small
knowns model set multilingual-e5-small
knowns search --status-check
knowns search --reindex
```
