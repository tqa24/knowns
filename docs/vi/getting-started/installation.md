# Cài đặt

Bạn có thể cài Knowns qua package manager hoặc build trực tiếp từ source.

## Yêu cầu

- môi trường terminal được hỗ trợ trên macOS, Linux, hoặc Windows
- Git nếu bạn muốn `knowns init` hoạt động theo ngữ cảnh repository
- tùy chọn: tải model cục bộ nếu bạn định dùng semantic search

## Cài bằng Homebrew

```bash
brew install knowns-dev/tap/knowns
```

Đây là cách nên dùng trên macOS và Linux nếu bạn muốn một bản cài đặt dạng package.

## Cài bằng npm

```bash
npm install -g knowns
```

Phù hợp khi môi trường của bạn đã dùng sẵn Node tooling.

## Shell installer (macOS/Linux)

```bash
curl -fsSL https://knowns.sh/script/install | sh
```

## PowerShell installer (Windows)

```powershell
irm https://knowns.sh/script/install.ps1 | iex
```

## Build từ source

```bash
go build -o ./bin/knowns ./cmd/knowns
```

Đây là lựa chọn phù hợp nhất khi bạn đang phát triển chính Knowns.

## Kiểm tra cài đặt

```bash
knowns --version
```

## Chạy mà không cần cài global

Nếu bạn không muốn cài global, vẫn có thể chạy Knowns qua npm:

```bash
npx knowns init
```

## Đọc tiếp

- [Bắt đầu nhanh](./quick-start.md)
