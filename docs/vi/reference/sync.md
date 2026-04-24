# Đồng bộ

`knowns sync` áp lại `.knowns/config.json` lên máy hiện tại.

## Khi nào nên dùng?

Hãy dùng `knowns sync` sau khi:

- clone một repository đã có `.knowns/`
- thay đổi các nền tảng đã chọn
- nâng cấp CLI
- muốn các file được tạo khớp lại với config hiện tại

## Các cách dùng phổ biến

```bash
knowns sync
knowns sync --skills
knowns sync --instructions
knowns sync --model
knowns sync --instructions --platform claude
knowns sync --instructions --platform cursor
```

## Nó có thể làm mới những gì?

- skills
- instruction files
- MCP config
- cấu hình riêng theo từng nền tảng
- git integration
- semantic-search setup
- search index trong các flow liên quan

## Liên quan

- [Cấu hình](./configuration.md)
- [Tương thích](../integrations/compatibility.md)
- [Tự động đồng bộ](../integrations/auto-sync.md)
