# Workflow cho AI

Hãy chọn đúng luồng làm việc cho loại việc bạn sắp làm. Tất cả các luồng dưới đây đều giả định rằng AI có thể dùng nhóm skill `/kn-*` khi runtime hỗ trợ.

Hãy dùng tài liệu này khi bạn cần quyết định xem một việc nên đi theo đầy đủ quy trình spec-driven development, đi theo luồng task thông thường, hay chỉ cần một quick fix ngắn.

## Chọn đúng luồng

| Luồng | Nên dùng khi nào | Trình tự điển hình |
|---|---|---|
| Full SDD | tính năng lớn, hệ thống mới, thay đổi chạm nhiều nơi | init -> research -> spec -> plan -> implement -> verify -> extract |
| Normal | tính năng nhỏ, cải tiến, task đã khá rõ | init -> plan -> implement |
| Quick Fix | bug fix, hotfix, sửa lỗi nhỏ | init -> implement |

## Vì sao nên research trước khi viết spec?

Trước khi viết spec, AI thường cần đủ ngữ cảnh để tránh viết sai yêu cầu.

Hãy research trước khi:

- khu vực codebase đó còn lạ
- thay đổi liên quan tới pattern hoặc ràng buộc sẵn có
- bạn cần xem cách chức năng tương tự đang được làm ra sao

Vì vậy, luồng Full SDD bên dưới bắt đầu bằng `research` trước rồi mới tới `spec`.

## Full SDD flow

Dùng luồng này cho các tính năng lớn hoặc thay đổi xứng đáng có spec riêng.

### 1. Khởi tạo phiên làm việc

```text
/kn-init
```

AI sẽ đọc project context, guidance, docs, và trạng thái hiện tại.

### 2. Research trước

```text
/kn-research
```

Dùng bước này để gom đủ ngữ cảnh từ codebase trước khi viết spec.

### 3. Tạo spec

```text
/kn-spec user-auth
```

AI sẽ tạo một spec document thường bao gồm:

- tổng quan và yêu cầu
- acceptance criteria
- tình huống và edge case

### 4. Lập kế hoạch từ spec

```text
/kn-plan --from @doc/specs/user-auth
```

AI tách spec thành các task và ánh xạ ngược về acceptance criteria của spec.

### 5. Thực hiện

```text
/kn-implement 42
```

AI đọc task, follow doc/spec liên quan, rồi thực hiện phần triển khai.

### 6. Kiểm tra lại

```text
/kn-verify
```

Những thứ thường được kiểm tra gồm:

- độ phủ acceptance criteria
- tính hợp lệ của reference
- độ nhất quán giữa spec và task

### 7. Trích xuất tri thức dùng lại được

```text
/kn-extract
```

Dùng bước này khi phần triển khai sinh ra pattern, quyết định, hoặc bài học có thể tái sử dụng về sau.

## Normal flow

Dùng cho các tính năng nhỏ hơn, khi task đã có sẵn và vấn đề đã khá rõ.

1. `/kn-init`
2. `/kn-plan 42`
3. `/kn-implement 42`

## Quick Fix flow

Dùng cho bug fix, hotfix, hoặc sửa lỗi nhỏ.

1. `/kn-init`
2. `/kn-implement 42`

## Khi nào nên dùng `kn-go`

`kn-go` hữu ích khi bạn đã có một spec được duyệt và muốn chạy toàn bộ pipeline liên tục, không cần chặn lại để review thủ công giữa từng giai đoạn.

Hãy dùng `kn-go` khi:

- spec đã được duyệt
- bạn muốn việc tạo task, lập kế hoạch, triển khai, kiểm tra lại, và chuẩn bị commit đi liền thành một luồng liên tục
- bạn không cần xem xét riêng từng task trước khi bắt đầu viết code

Hãy ưu tiên `/kn-plan` + `/kn-implement` khi:

- bạn muốn xem kỹ hoặc điều chỉnh từng task trước khi coding
- spec vẫn còn đang thay đổi
- bạn muốn có thêm các điểm review rõ ràng giữa các pha

## Khi nào nên dùng `kn-debug`

Hãy dùng `kn-debug` khi công việc bị chặn bởi một lỗi thật sự, chứ không phải vì thiếu kế hoạch.

Các tình huống điển hình:

- lỗi build hoặc lỗi kiểu dữ liệu
- test bị fail
- runtime crash hoặc exception
- lỗi integration
- task bị chặn nhưng bạn vẫn chưa rõ nguyên nhân gốc là gì

Nói ngắn gọn: nếu bước tiếp theo hợp lý nhất là tái hiện lỗi, chẩn đoán, rồi sửa lỗi một cách có hệ thống, thì hãy chuyển sang `kn-debug` thay vì cứ tiếp tục `implement`.

## Khi nào nên dùng `kn-extract`

Hãy dùng `kn-extract` khi phần việc bạn vừa hoàn thành tạo ra thứ gì đó có thể tái sử dụng, và không nên để nó bị chôn trong một task hoặc một phiên chat duy nhất.

Các tình huống điển hình:

- bạn tìm ra một pattern triển khai có thể lặp lại ở nhiều nơi
- bạn chốt một quyết định ở cấp dự án mà các phần việc sau cần tuân theo
- bạn gặp một failure mode mà cách nhận biết và cách sửa nên được nhớ lại về sau
- bạn muốn chuyển tri thức mang tính ngẫu hứng trong lúc làm việc thành docs, memory, hoặc template dùng lại được

Nên dùng `kn-extract` ở gần cuối task, hoặc sau bước verify, khi bạn đã biết chắc phần kết quả đó đủ giá trị để lưu lại cho con người và AI dùng về sau.

## Tra cứu skill

| Skill | Mục đích |
|---|---|
| `/kn-init` | Nạp ngữ cảnh dự án |
| `/kn-research` | Khám phá codebase và pattern sẵn có |
| `/kn-spec` | Tạo spec document |
| `/kn-plan` | Lập kế hoạch triển khai |
| `/kn-implement` | Thực hiện công việc |
| `/kn-verify` | Kiểm AC, refs, và độ nhất quán |
| `/kn-extract` | Lưu tri thức có thể tái sử dụng |
| `/kn-doc` | Làm việc với docs |
| `/kn-template` | Chạy templates |
| `/kn-debug` | Gỡ lỗi khi công việc bị chặn hoặc bị fail |

## CLI fallback

Nếu runtime không có skills, hãy dùng CLI trực tiếp.

```bash
# Nạp context bằng tay
knowns doc list --plain
knowns doc "readme" --plain --smart

# Nhận task
knowns task edit 42 -s in-progress -a @me
knowns time start 42

# Thêm kế hoạch
knowns task edit 42 --plan $'1. Research\n2. Implement\n3. Test'

# Đánh dấu AC và thêm ghi chú
knowns task edit 42 --check-ac 1
knowns task edit 42 --append-notes "Completed feature X"

# Hoàn tất
knowns time stop
knowns task edit 42 -s done
```

## Tách phiên làm việc khi cần

Với công việc lớn, thường nên tách phiên làm việc riêng theo từng task hoặc từng pha.

Ví dụ:

- một phiên cho research
- một phiên cho spec và planning
- một phiên cho implementation

Làm như vậy sẽ giảm nguy cơ context bị compact và giúp mỗi phiên dễ kiểm soát hơn.

## Định nghĩa “xong”

Theo kinh nghiệm, một task có thể xem là hoàn tất khi:

- acceptance criteria đã được check
- notes hoặc chi tiết triển khai đã được ghi lại
- nếu task đang được theo dõi thời gian thì timer đã được dừng
- validation hoặc test liên quan đã pass
- trạng thái task đã được cập nhật đúng

## Liên quan

- [Quản lý task](./task-management.md)
- [Hướng dẫn cho AI agent](./ai-agent-guide.md)
- [Cách làm việc đề xuất](./workflow.md)
