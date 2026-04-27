# Làm việc với AI

Cách dùng Knowns hiệu quả khi làm việc cùng AI assistants.

## Ý chính

AI làm việc tốt hơn nhiều khi không phải tự đoán context.

Knowns cho AI truy cập có cấu trúc vào:

- task
- doc
- memory
- reference
- validation
- search và retrieval

## Cách dùng

### 1. Load guidance trước

AI nên bắt đầu từ `KNOWNS.md` — file hướng dẫn canonical của repo.

### 2. Dùng task làm mục tiêu

Thay vì prompt mơ hồ, trỏ AI vào task có acceptance criteria rõ ràng.

### 3. Dùng doc cho context lâu dài

Architecture, pattern, hướng dẫn vận hành nên nằm trong doc, không chỉ tồn tại trong chat.

### 4. Dùng memory cho decision cần nhớ

Lưu decision, convention, failure vào memory để gọi lại sau.

### 5. Validate trước khi coi là xong

Validation nên là phần bình thường của workflow.

## MCP hay CLI?

### Ưu tiên MCP khi:

- AI runtime hỗ trợ MCP
- muốn structured tool calls
- muốn giảm parse shell output

### Ưu tiên CLI khi:

- MCP không có
- đang script ngoài MCP-aware runtime
- muốn xem output trực tiếp trong terminal

## Ví dụ workflow

1. AI đọc `KNOWNS.md`
2. AI đọc target task
3. AI follow `@doc/...` hoặc `@task-...` references
4. AI search/retrieve thêm context nếu cần
5. AI implement
6. AI chạy validation hoặc test

## Xem thêm

- [Quản lý task](./task-management.md)
- [MCP](./mcp-integration.md)
- [Workflow](./workflow.md)
