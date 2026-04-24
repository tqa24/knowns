# Reference System

Knowns hỗ trợ các tham chiếu có cấu trúc giữa task, tài liệu, memory, và template.

## Các dạng phổ biến

- `@task-abc123`
- `@doc/guides/setup`
- `@memory-xyz789`
- `@template/react-component`

## Các hậu tố hữu ích cho tài liệu

- `@doc/path:42`
- `@doc/path:10-20`
- `@doc/path#heading-slug`

## Vì sao tham chiếu quan trọng

Tham chiếu giúp cả con người lẫn AI di chuyển trong ngữ cảnh project mà không phải đoán filename hay ID bằng tay.

## Các lệnh liên quan

```bash
knowns resolve "@doc/specs/auth{implements}" --plain
knowns search "authentication" --plain
knowns retrieve "how auth works" --json
```
