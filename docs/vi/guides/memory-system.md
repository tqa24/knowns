# Memory

Memory là nơi Knowns lưu context cần nhớ lại sau này.

## Ba layer

- **working memory** — context ngắn hạn, chỉ sống trong một session
- **project memory** — decision, pattern, convention riêng của một repo
- **global memory** — preference hoặc rule dùng được across projects

## Memory hay doc?

Dùng memory khi:

- ngắn gọn, cần recall nhanh
- thiên về decision hơn là giải thích dài
- hữu ích cho nhiều lần làm việc sau

Dùng doc khi cần giải thích dài hoặc chia thành nhiều section.

## Ví dụ

```
"We use repository pattern for data access"
"Always validate before marking a task done"
"Prefer semantic search over manual grep for exploration"
```

Nội dung memory thường viết bằng tiếng Anh vì AI đọc trực tiếp.

## Lệnh

```bash
knowns memory add "We use repository pattern" --category decision
knowns memory list --plain
knowns memory <id> --plain
```

## Xem thêm

- [Quản lý task](./task-management.md)
- [Reference system](../reference/reference-system.md)
