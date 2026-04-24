# Hướng dẫn cho AI agent

Tài liệu này giải thích cách dùng Knowns hiệu quả khi làm việc cùng AI assistants.

## Ý chính

AI làm việc tốt hơn rất nhiều khi nó không phải tự đoán ngữ cảnh của dự án.

Knowns cung cấp cho AI một cách có cấu trúc để truy cập:

- task
- doc
- memory
- reference
- validation
- search và retrieval

## Cách dùng được khuyến nghị

### 1. Để AI nạp guidance trước

AI nên bắt đầu từ file hướng dẫn canonical của repo là `KNOWNS.md`.

### 2. Dùng task làm mục tiêu thực thi

Thay vì đưa prompt mơ hồ, hãy cho AI làm việc dựa trên một task có acceptance criteria rõ ràng.

### 3. Dùng doc cho phần giải thích lâu dài

Kiến trúc, pattern, và hướng dẫn vận hành nên nằm trong doc thay vì chỉ tồn tại trong chat.

### 4. Dùng memory cho các quyết định cần nhớ lâu

Lưu decision, convention, và failure vào memory để có thể gọi lại về sau.

### 5. Validate trước khi coi là xong

Validation nên là một phần của workflow bình thường.

## MCP hay CLI?

### Ưu tiên MCP khi:

- AI runtime hỗ trợ MCP
- bạn muốn tool call có cấu trúc
- bạn muốn giảm việc parse shell output hoặc copy-paste prompt

### Ưu tiên CLI khi:

- MCP không có sẵn
- bạn đang script bên ngoài một runtime hỗ trợ MCP
- bạn muốn tự xem output trực tiếp trong terminal

## Ví dụ một workflow điển hình

1. AI đọc `KNOWNS.md`
2. AI đọc task mục tiêu
3. AI follow các reference như `@doc/...` hoặc `@task-...`
4. AI search hoặc retrieve thêm context nếu cần
5. AI thực hiện thay đổi
6. AI chạy validation hoặc test

## Liên quan

- [Quản lý task](./task-management.md)
- [Tích hợp MCP](./mcp-integration.md)
- [Cách làm việc đề xuất](./workflow.md)
