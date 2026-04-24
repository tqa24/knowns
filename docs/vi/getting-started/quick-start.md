# Bắt đầu nhanh

Đây là cách đơn giản nhất để bắt đầu một dự án với Knowns.

## 1. Khởi tạo dự án

```bash
knowns init
# hoặc, nếu chưa cài global:
npx knowns init
```

Khi chạy `knowns init`, bạn có thể cấu hình:

- tên dự án
- chế độ theo dõi bằng Git
- các nền tảng AI cần tích hợp
- Chat UI dùng OpenCode (tùy chọn)
- semantic search
- mô hình embedding

Nếu cửa sổ terminal quá hẹp, Knowns có thể tự chuyển sang cấu hình mặc định không tương tác thay vì cố hiển thị toàn bộ wizard.

## 2. Tạo task đầu tiên

```bash
knowns task create "Setup project" -d "Khởi tạo dự án với Knowns"
```

## 3. Tạo tài liệu đầu tiên

```bash
knowns doc create "Architecture" -d "Tổng quan hệ thống" -f architecture
```

## 4. Mở giao diện web

```bash
knowns browser --open
```

## 5. Đồng bộ lại các tệp được tạo khi cần

```bash
knowns sync
knowns update
```

Dùng `knowns sync` sau khi clone repo, sau khi đổi nền tảng đã chọn, hoặc sau khi cập nhật CLI.

## 6. Mở lại giao diện web khi cần

```bash
knowns browser --open
```

## Đọc tiếp

- [Dự án đầu tiên](./first-project.md)
- [Hướng dẫn sử dụng](../guides/user-guide.md)
- [Cách làm việc đề xuất](../guides/workflow.md)
