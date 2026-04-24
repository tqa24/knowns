# Hệ memory

Memory là nơi Knowns lưu phần ngữ cảnh cần được nhớ lại về sau.

## Ba lớp memory

- **working memory**: ngữ cảnh ngắn hạn, chỉ sống trong một phiên làm việc
- **project memory**: quyết định, pattern, và convention riêng của một repository
- **global memory**: sở thích hoặc quy tắc có thể áp dụng lại giữa nhiều dự án

## Khi nào nên dùng memory thay vì doc?

Hãy dùng memory khi thông tin đó:

- ngắn gọn và cần gọi lại nhanh
- thiên về quyết định hơn là giải thích dài
- hữu ích cho nhiều lần làm việc sau này

Hãy dùng doc khi bạn cần giải thích dài hơn hoặc cần nội dung chia thành nhiều phần rõ ràng.

## Ví dụ điển hình

- “We use repository pattern for data access”
- “Always validate before marking a task done”
- “This team prefers semantic search before manual grep for exploratory work”

## Các lệnh liên quan

```bash
knowns memory add "We use repository pattern" --category decision
knowns memory list --plain
knowns memory <id> --plain
```

## Liên quan

- [Quản lý task](./task-management.md)
- [Hệ thống tham chiếu](../reference/reference-system.md)
