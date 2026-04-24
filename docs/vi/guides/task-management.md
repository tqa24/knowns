# Quản lý task

Task là đơn vị công việc chính trong Knowns.

## Một task thường có gì?

Một task điển hình có thể gồm:

- tiêu đề
- mô tả
- trạng thái
- mức độ ưu tiên
- nhãn
- người phụ trách
- acceptance criteria
- kế hoạch thực hiện
- ghi chú thực hiện

## Vì sao task quan trọng?

Task quan trọng vì nó cho cả con người lẫn AI một mục tiêu công việc cụ thể.

Thay vì nói chung chung kiểu “làm phần auth đi”, bạn có thể định nghĩa rõ:

- cần xây gì
- kiểm tra thành công bằng cách nào
- ngữ cảnh nào liên quan
- hiện tại đã làm đến đâu

## Luồng điển hình

```bash
knowns task create "Add authentication" \
  -d "JWT-based auth with login and register endpoints" \
  --ac "User can register" \
  --ac "User can login" \
  --priority high

knowns task edit <id> -s in-progress
knowns task edit <id> --plan $'1. Review auth pattern\n2. Implement endpoints\n3. Add tests'
knowns task edit <id> --check-ac 1
knowns task edit <id> --append-notes "Completed middleware"
knowns task edit <id> -s done
```

## Acceptance criteria

Acceptance criteria đặc biệt quan trọng khi làm việc cùng AI vì chúng biến khái niệm “xong” thành thứ có thể kiểm tra được.

Acceptance criteria tốt nên:

- cụ thể
- quan sát được
- đủ nhỏ để kiểm từng cái một

## Tham chiếu trong task

Task có thể tham chiếu tới doc hoặc thực thể khác, ví dụ:

- `@doc/architecture/auth`
- `@task-abc123`

## Liên quan

- [Tra cứu lệnh](../reference/commands.md)
- [Hệ thống tham chiếu](../reference/reference-system.md)
- [Cách làm việc đề xuất](./workflow.md)
