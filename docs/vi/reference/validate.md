# Kiểm tra hợp lệ

`knowns validate` dùng để kiểm tra tính nhất quán của lớp ngữ cảnh hiện tại trong dự án.

## Nó dùng để làm gì?

Hãy dùng validation để phát hiện các vấn đề như:

- tham chiếu bị hỏng
- quan hệ task/spec chưa đầy đủ
- chênh lệch giữa cấu trúc mong đợi và dữ liệu đang lưu

## Các lệnh thường dùng

```bash
knowns validate --plain
knowns validate --scope docs --plain
knowns validate --scope sdd --plain
knowns validate --strict --plain
```

## Khi nào nên chạy?

- trước khi chốt một task
- sau khi sắp xếp lại docs
- sau khi thay đổi references hoặc generated files
- trước khi để AI dựa nhiều vào project context đã lưu
