# Tương thích

Tài liệu này giải thích các hành vi tương thích chính mà Knowns giữ lại khi layout tích hợp hoặc generated artifact thay đổi theo thời gian.

## Vì sao cần tài liệu này?

Knowns quản lý nhiều loại file được tạo ra tự động, ví dụ:

- thư mục skills
- tệp cấu hình MCP
- instruction files
- runtime hooks

Khi cách tích hợp thay đổi, các dự án cũ có thể vẫn còn layout trước đây. Knowns cố giữ khả năng tương thích an toàn thay vì làm gãy các dự án đó ngay lập tức.

## Tương thích cho thư mục skills

Mapping chính hiện tại:

- `.claude/skills` -> Claude Code
- `.agents/skills` -> OpenCode, Codex, Antigravity
- `.kiro/skills` -> Kiro
- `.agent/skills` -> chỉ dùng cho tương thích cũ / generic

### Hành vi với thiết lập cũ

Nếu một dự án cũ đã có `.agent/skills`, Knowns vẫn tiếp tục đồng bộ thư mục đó để giữ khả năng tương thích.

Điều này có nghĩa là:

- dự án mới nên dùng `.agents/skills` cho các nền tảng agent-compatible
- dự án cũ không bị gãy ngay lập tức
- `knowns sync` có thể in warning khi phát hiện đường dẫn cũ

## Tương thích cho MCP theo từng nền tảng

Hiện nay Knowns quản lý cấu hình MCP cục bộ cho một số nền tảng như:

- Claude Code -> `.mcp.json`
- Kiro -> `.kiro/settings/mcp.json`
- Cursor -> `.cursor/mcp.json`
- Codex -> `.codex/config.toml`
- OpenCode -> `opencode.json`

Với Antigravity, cấu hình MCP là cấu hình toàn cục:

- `~/.gemini/antigravity/mcp_config.json`

## Vai trò của init, sync, và update

### `knowns init`

Tạo các artifact của nền tảng đã chọn ngay từ đầu cho một project mới.

### `knowns sync`

Áp lại `.knowns/config.json` lên máy hiện tại.

Hãy dùng sau khi:

- clone repository
- thay đổi các nền tảng đã chọn
- muốn các file được tạo khớp lại với config hiện tại

### `knowns update`

Cập nhật CLI, sau đó làm mới các generated artifact phụ thuộc vào binary hoặc chính sách cấu hình hiện tại.

## Khuyến nghị

- Với dự án mới, hãy đi theo layout chính hiện tại.
- Với dự án cũ, hãy để `knowns sync` và `knowns update` giữ tương thích trước, rồi mới migrate có chủ đích nếu cần.
