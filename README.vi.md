<p align="center">
  <img src="./images/logo.png" alt="Knowns" width="120">
</p>

<h1 align="center">Knowns</h1>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/go-%3E%3D1.24.2-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="https://www.npmjs.com/package/knowns"><img src="https://img.shields.io/npm/v/knowns.svg?style=flat-square" alt="npm"></a>
  <a href="https://github.com/knowns-dev/knowns/actions/workflows/ci.yml"><img src="https://github.com/knowns-dev/knowns/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="#cài-đặt"><img src="https://img.shields.io/badge/platform-win%20%7C%20mac%20%7C%20linux-lightgrey?style=flat-square" alt="Platform"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/knowns-dev/knowns?style=flat-square" alt="License"></a>
</p>

<p align="center">
  <a href="https://knowns.sh">Trang chủ</a> |
  <a href="./README.md">English</a> |
  <a href="./docs/vi/README.md">Tài liệu</a>
</p>

<p align="center">
  <strong>Cho AI coding assistant của bạn quyền truy cập có cấu trúc vào task, tài liệu, đặc tả và quyết định - để nó ngừng đoán mò và bắt đầu xây dựng đúng cách.</strong>
</p>

---

Mỗi lần bắt đầu một phiên làm việc mới với AI, bạn lại phải giải thích lại kiến trúc, dán tài liệu, nhắc lại convention, rồi làm rõ những quyết định đã có từ trước. AI của bạn có thể rất mạnh - nhưng nó không nhớ được gì giữa các phiên.

**Knowns giải quyết đúng chỗ đó.** Nó cung cấp cho các AI assistants như Claude, Cursor, Copilot và nhiều công cụ khác quyền truy cập bền vững, có cấu trúc vào task, tài liệu, đặc tả, acceptance criteria và các quyết định kiến trúc của dự án. Thay vì phải bắt đầu lại từ đầu trong prompt, AI sẽ đọc đúng phần nó cần và tiếp tục công việc từ nơi bạn đã dừng.

Nếu bạn tin rằng AI nên thực sự hiểu dự án phần mềm, hãy cân nhắc tặng **Knowns** một star.

<p align="center">
  <img src="./images/task-workflow.gif" alt="Knowns task workflow demo" width="100%">
</p>

## Mục lục

- [Vì sao cần Knowns?](#vì-sao-cần-knowns)
- [Trước và sau khi dùng](#trước-và-sau-khi-dùng)
- [Knowns là gì?](#knowns-là-gì)
- [Dành cho ai?](#dành-cho-ai)
- [Cách hoạt động](#cách-hoạt-động)
- [Bắt đầu nhanh](#bắt-đầu-nhanh)
- [Các khả năng chính](#các-khả-năng-chính)
- [Bạn có thể xây gì với Knowns?](#bạn-có-thể-xây-gì-với-knowns)
- [Workflow skills cho Claude Code](#workflow-skills-cho-claude-code)
- [Cài đặt](#cài-đặt)
- [Tài liệu](#tài-liệu)
- [Lộ trình phát triển](#lộ-trình-phát-triển)
- [Phát triển](#phát-triển)
- [Liên kết](#liên-kết)

---

## Vì sao cần Knowns?

AI coding assistants là công cụ không có trạng thái. Mỗi phiên làm việc gần như bắt đầu lại từ đầu.

Điều đó dẫn tới việc bạn phải làm đi làm lại những việc giống nhau:

- **Giải thích lại** kiến trúc và các design pattern của dự án
- **Dán lại** tài liệu vào khung chat
- **Nhắc lại** convention code và quy tắc làm việc
- **Làm rõ lại** những quyết định đã được chốt từ tuần trước
- **Dựng lại ngữ cảnh** vốn đã mất 20 phút để chuẩn bị

AI không thiếu thông minh. **Nó thiếu quyền truy cập vào những gì dự án của bạn đã biết.**

Knowns cung cấp đúng quyền truy cập đó.

---

## Trước và sau khi dùng

| Không có Knowns | Có Knowns |
|---|---|
| "Bọn tôi dùng repository pattern như này..." _(dán 50 dòng)_ | AI tự đọc `@doc/patterns/repository` |
| "Đây là task, acceptance criteria là..." _(gõ lại từ đầu)_ | AI đọc task, AC, spec liên quan và docs liên quan |
| "Nhớ là tuần trước bọn tôi quyết định..." _(hy vọng nó nhớ)_ | Quyết định được lưu trong project memory - AI nhớ lại ở mỗi phiên |
| "Luồng auth đang hoạt động như này..." _(giải thích lần thứ 4)_ | AI đi theo `@doc/architecture/auth` và xây tiếp trên đó |
| "Không biết xong chưa, để tôi kiểm tra lại yêu cầu..." | AI tự kiểm acceptance criteria và validate trạng thái hoàn thành |
| Phiên mới bắt đầu lạnh - mất 10 phút để dựng context | Phiên mới bắt đầu ấm - AI đã biết dự án |

---

## Knowns là gì?

Knowns là một **lớp ngữ cảnh local-first, tự host được**, dành cho phát triển phần mềm theo hướng AI-native.

Nó lưu tri thức của dự án dưới dạng file có cấu trúc, dễ đọc với AI - và mở tri thức đó cho AI assistants qua CLI và [MCP (Model Context Protocol)](https://modelcontextprotocol.io/).

<p align="center">
  <img src="./images/knowledge-graph.svg" alt="Knowns Knowledge Graph" width="100%">
</p>

Cụ thể, Knowns quản lý:

- **Tasks** với acceptance criteria, implementation plan, và theo dõi trạng thái
- **Documentation** trong các thư mục markdown lồng nhau, có cross-reference
- **Specs** để định nghĩa rõ thế nào là "xong"
- **Memory** - tri thức ở cấp project, session, và global để AI có thể nhớ lại
- **Templates** để sinh mã bằng Handlebars
- **References** như `@task-42` và `@doc/patterns/auth` để AI có thể follow và resolve
- **Code intelligence** - symbol được index bằng AST, dependency graph, và semantic code search

Mọi thứ nằm trong thư mục `.knowns/` của repo. Chỉ là file thường. Commit vào Git được. Không cần cloud.

---

## Dành cho ai?

- **Lập trình viên solo** pair với AI hằng ngày và muốn AI nhớ context của dự án qua nhiều phiên
- **Nhóm phát triển** đang dùng AI assistants và chán cảnh ai cũng phải giải thích lại kiến trúc từ đầu
- **Maintainer mã nguồn mở** muốn người đóng góp (người hoặc AI) onboard nhanh hơn
- **Bất kỳ ai** dùng Claude, Cursor, Copilot, Windsurf hoặc các AI coding tools khác và muốn chúng thực sự hiểu dự án

---

## Cách hoạt động

Knowns nằm cạnh các công cụ bạn đang dùng. Stack hiện tại của bạn không cần đổi.

<p align="center">
  <img src="./images/architecture.svg" alt="Knowns Architecture" width="100%">
</p>

1. **Bạn cấu trúc tri thức của dự án** - task, docs, specs, decisions - bằng CLI hoặc Web UI của Knowns
2. **AI đọc phần đó** - qua MCP integration hoặc CLI commands, AI assistant truy cập đúng phần context nó cần
3. **AI hành động dựa trên đó** - follow reference, kiểm acceptance criteria, cập nhật task status, và xây dựng với đầy đủ nhận thức về dự án
4. **Tri thức được tích lũy** - decisions, patterns, và conventions không biến mất sau mỗi phiên nữa

Đặc tả -> được hiểu. Task -> được kết nối. Doc -> dùng được. Quyết định -> được nhớ lại.

---

## Bắt đầu nhanh

```bash
# Cài đặt
brew install knowns-dev/tap/knowns
# hoặc: npm install -g knowns
# hoặc: curl -fsSL https://knowns.sh/script/install | sh
# hoặc trên PowerShell (Windows):
# irm https://knowns.sh/script/install.ps1 | iex

# kiểm tra bản cài
knowns --version

# Khởi tạo trong dự án của bạn
cd your-project
knowns init

# hoặc chạy ngay mà không cần cài global
npx knowns init

# Tạo task đầu tiên
knowns task create "Add user authentication" \
  -d "JWT-based auth with login and register endpoints" \
  --ac "User can register with email/password" \
  --ac "User can login and receive JWT token" \
  --ac "Protected routes reject unauthenticated requests"

# Thêm tài liệu cho dự án
knowns doc create "Auth Architecture" \
  -f "architecture" \
  -d "Authentication design decisions and patterns"

# Mở Web UI
knowns browser --open

# Cập nhật Knowns sau này
knowns update

# Kết nối AI assistant qua MCP
# Xem: docs/vi/guides/mcp-integration.md
```

Khi AI đọc dự án lúc này, nó sẽ thấy task có acceptance criteria, tài liệu được liên kết với nhau, và định nghĩa rõ ràng về việc thế nào là hoàn thành - thay vì phải đoán.

---

## Các khả năng chính

### Quản lý task và workflow

Tạo task với acceptance criteria, implementation plan, và trạng thái. AI có thể đọc task, đi theo kế hoạch, check từng AC, và biết rõ khi nào công việc đã hoàn tất.

```bash
knowns task create "Title" --ac "Criterion 1" --ac "Criterion 2"
knowns task edit <id> -s in-progress
knowns task edit <id> --check-ac 1
```

### Doc có cấu trúc

Tổ chức tri thức của dự án trong các thư mục markdown lồng nhau. Liên kết bằng `@doc/path` và `@task-id`. AI sẽ follow các reference này để nạp đúng phần context cần dùng.

```bash
knowns doc create "API Design" -f "architecture"
knowns doc "architecture/api-design" --smart --plain
```

### Project memory

Hệ memory 3 lớp - **project**, **session**, và **global** - giúp AI nhớ lại quyết định, pattern, và convention mà không cần bạn nhắc lại.

```bash
knowns memory add "We use repository pattern for data access" --category decision
knowns memory list --plain
```

### Semantic search

Tìm theo ý nghĩa chứ không chỉ theo từ khóa. Chạy local bằng ONNX models - hoàn toàn offline, không cần API key.

```bash
knowns search "how does authentication work" --plain
```

### MCP integration

MCP server đầy đủ theo [Model Context Protocol](https://modelcontextprotocol.io/). Claude, Cursor và các AI assistants hỗ trợ MCP có thể truy cập trực tiếp task, tài liệu, memory, search, và validation - không cần copy-paste.

### Code intelligence

Index bằng AST cho Go, TypeScript, JavaScript, và Python. Tìm symbol, lần dependency, và khám phá cấu trúc codebase - tất cả đều mở cho AI.

```bash
knowns code ingest
knowns code search "oauth login" --neighbors 5
knowns code deps --type calls
```

### Templates và sinh mã

Template dựa trên Handlebars để scaffold. Định nghĩa pattern một lần, sinh nhất quán nhiều lần.

```bash
knowns template list
knowns template run <name> --name "UserService"
```

### AI Agent Workspaces

Điều phối agent theo nhiều pha với git worktree isolation, live terminal streaming, và tự động chuyển pha (research -> plan -> implement -> review).

### Web UI

Kanban board, document browser, knowledge graph visualization, và hỗ trợ mermaid - tất cả trong browser UI local.

```bash
knowns browser --open
```

---

## Bạn có thể xây gì với Knowns?

| Khả năng | Mô tả |
|---|---|
| **Task Management** | Task có acceptance criteria, plan, status, và time tracking |
| **Doc** | Thư mục markdown lồng nhau với cross-reference và hỗ trợ mermaid |
| **Semantic Search** | Tìm theo ý nghĩa với local AI models (offline hoàn toàn) |
| **Time Tracking** | Bộ đếm thời gian và báo cáo theo từng task |
| **Context Linking** | `@task-42` và `@doc/patterns/auth` để AI resolve được ngữ cảnh |
| **Validation** | Phát hiện broken reference và task chưa hoàn chỉnh bằng `knowns validate` |
| **Template System** | Sinh mã bằng Handlebars (`.hbs`) templates |
| **Import System** | Import docs và templates từ git, npm, hoặc local source |
| **Memory System** | Memory ở cấp project / session / global để AI nhớ lâu dài |
| **MCP Server** | Tích hợp AI assistant với đầy đủ tool access |
| **AI Workspaces** | Điều phối agent nhiều pha với worktree isolation |
| **Code Intelligence** | AST indexing, symbol search, và dependency graphs |
| **Web UI** | Kanban board, doc browser, knowledge graph, mermaid diagrams |

---

## Workflow skills cho Claude Code

Khi dùng Claude Code với Knowns, các skills (slash commands) tạo ra một workflow rõ ràng:

<p align="center">
  <img src="./images/workflow.svg" alt="Knowns AI Workflow" width="100%">
</p>

```
/kn-init      -> Đọc tài liệu dự án, hiểu ngữ cảnh
/kn-plan      -> Nhận task, gom context, lập kế hoạch (chờ duyệt)
/kn-implement -> Làm theo kế hoạch, kiểm AC, theo dõi tiến độ
/kn-commit    -> Commit theo conventional format
```

### Ví dụ một phiên làm việc

```
Bạn: /kn-init
Claude: [Đọc README, ARCHITECTURE, kiểm task backlog]
        "Project dùng NestJS với CQRS pattern. Có 3 task đang in-progress..."

Bạn: /kn-plan 42
Claude: [Nhận task, đọc refs, tìm docs]
        "Task #42: Add user authentication

         ## Kế hoạch thực hiện
         1. Review @doc/patterns/auth
         2. Tạo AuthModule với JWT strategy
         3. Thêm login/register endpoints
         4. Viết unit tests

         Bạn có duyệt kế hoạch này không?"

Bạn: Có

Bạn: /kn-implement 42
Claude: [Làm theo kế hoạch, sinh code, check AC]
        "✓ AC1: User can login - DONE
         ✓ AC2: JWT token returned - DONE
         ✓ AC3: Tests pass - DONE

         Tất cả acceptance criteria đã hoàn tất. Sẵn sàng để commit?"

Bạn: /kn-commit
Claude: [Tạo conventional commit]
        "feat(auth): add JWT authentication

         - Add AuthModule with passport JWT strategy
         - Add login/register endpoints
         - Add unit tests (94% coverage)"
```

### Danh sách skills

| Skill | Mô tả |
|---|---|
| `/kn-init` | Khởi tạo phiên làm việc - đọc docs, nạp memory, hiểu project |
| `/kn-plan <id>` | Nhận task, gom context, tạo implementation plan |
| `/kn-implement <id>` | Thực hiện theo kế hoạch, theo dõi tiến độ, kiểm acceptance criteria |
| `/kn-research` | Tìm trong codebase, lần pattern, khám phá trước khi viết code |
| `/kn-commit` | Tạo commit theo conventional format có kiểm tra |
| `/kn-spec` | Tạo specification document cho feature (SDD) |
| `/kn-go <spec>` | Chạy full pipeline từ spec đã duyệt (không có review gate) |
| `/kn-verify` | Chạy SDD verification và coverage report |
| `/kn-review` | Code review nhiều góc nhìn (P1/P2/P3) |
| `/kn-doc` | Tạo hoặc cập nhật tài liệu |
| `/kn-extract` | Trích xuất pattern tái sử dụng vào docs, templates, và memory |
| `/kn-template` | Liệt kê, chạy, hoặc tạo template mới |
| `/kn-debug` | Debug lỗi và failure theo kiểu triage có memory hỗ trợ |

---

## Cài đặt

### Homebrew (macOS/Linux)

```bash
brew install knowns-dev/tap/knowns
```

### Shell installer (macOS/Linux)

```bash
curl -fsSL https://knowns.sh/script/install | sh

# Hoặc dùng wget
wget -qO- https://knowns.sh/script/install | sh

# Cài một version cụ thể
curl -fsSL https://knowns.sh/script/install | KNOWNS_VERSION=0.18.0 sh
```

### PowerShell installer (Windows)

```powershell
irm https://knowns.sh/script/install.ps1 | iex

# Cài một version cụ thể
$env:KNOWNS_VERSION = "0.18.0"; irm https://knowns.sh/script/install.ps1 | iex
```

Shell installer trên macOS/Linux và PowerShell installer trên Windows đều tự chạy `knowns search --install-runtime` sau khi cài binary. Nếu bước đó lỗi, hãy chạy lại bằng tay.

### npm

```bash
# Cài global - tự tải binary theo từng nền tảng
npm install -g knowns

# Hoặc chạy mà không cần cài
npx knowns
```

### Build từ source (Go 1.24.2+)

```bash
go install github.com/howznguyen/knowns/cmd/knowns@latest

# Hoặc clone rồi build
git clone https://github.com/knowns-dev/knowns.git
cd knowns
make build        # Output: bin/knowns
make install      # Cài vào GOPATH/bin
```

### Gỡ cài đặt

```bash
# macOS/Linux
curl -fsSL https://knowns.sh/script/uninstall | sh

# Windows
irm https://knowns.sh/script/uninstall.ps1 | iex
```

Các script gỡ cài đặt chỉ xóa CLI binary và PATH entries do installer thêm vào. Chúng không đụng tới thư mục `.knowns/` trong các dự án của bạn.

---

## Tra cứu nhanh

```bash
# Tasks
knowns task create "Title" -d "Description" --ac "Criterion"
knowns task list --plain
knowns task <id> --plain
knowns task edit <id> -s in-progress -a @me
knowns task edit <id> --check-ac 1

# Documentation
knowns doc create "Title" -d "Description" -f "folder"
knowns doc "doc-name" --plain
knowns doc "doc-name" --smart --plain
knowns doc "doc-name" --section "2" --plain

# Templates
knowns template list
knowns template run <name> --name "X"
knowns template create <name>

# Imports
knowns import add <name> <source>
knowns import sync
knowns import list

# Time, Search & Validate
knowns time start <id> && knowns time stop
knowns search "query" --plain
knowns validate

# Code intelligence
knowns code ingest
knowns code search "oauth login" --neighbors 5
knowns code deps --type calls
knowns code symbols --kind function

# AI Guidelines
knowns agents --sync
knowns sync
```

---

## Tài liệu

| Hướng dẫn | Mô tả |
|---|---|
| [Hướng dẫn sử dụng](./docs/vi/guides/user-guide.md) | Bắt đầu và sử dụng hằng ngày |
| [Tra cứu lệnh](./docs/vi/reference/commands.md) | Các lệnh CLI chính và ví dụ |
| [Workflow Guide](./docs/vi/guides/workflow.md) | Cách làm việc đề xuất từ lúc tạo task đến khi hoàn tất |
| [MCP Integration](./docs/vi/guides/mcp-integration.md) | Thiết lập MCP cho Claude Desktop, Cursor, Codex, OpenCode, ... |
| [Reference System](./docs/vi/reference/reference-system.md) | Cách `@doc/` và `@task-` hoạt động |
| [Semantic Search](./docs/vi/reference/semantic-search.md) | Thiết lập và sử dụng semantic search |
| [Templates](./docs/vi/integrations/templates.md) | Sinh mã bằng Handlebars templates |
| [Web UI](./docs/vi/guides/web-ui.md) | Board, doc browser, graph, và chat UI |
| [Configuration](./docs/vi/reference/configuration.md) | Cấu trúc project config và các tùy chọn |
| [Skills](./docs/vi/integrations/skills.md) | Skills và đường dẫn sync theo nền tảng |
| [Developer Guide](./docs/vi/contributing/developer-guide.md) | Hướng dẫn cho người đóng góp |
| [Platforms](./docs/vi/integrations/platforms.md) | Mapping tích hợp đa nền tảng |

---

## Lộ trình phát triển

### AI Agent Workspaces ✅ (Đang hoạt động)

Điều phối agent theo nhiều pha - giao task cho AI agents với git worktree isolation, live terminal streaming, và tự động chuyển pha (research -> plan -> implement -> review).

### Self-Hosted Team Sync 🚧 (Dự kiến)

Một self-hosted sync server tùy chọn để có shared visibility mà không phải bỏ mô hình local-first.

- **Quan sát thời gian thực** - Biết ai đang làm gì
- **Tri thức chia sẻ** - Đồng bộ task và tài liệu trong nhóm
- **Toàn quyền dữ liệu** - Tự host, không phụ thuộc cloud

---

## Phát triển

Cần **Go 1.24.2+** và tùy chọn **Node.js + pnpm** nếu bạn phát triển UI.

```bash
make build              # Build binary -> bin/knowns
make dev                # Build với race detector
make test               # Chạy unit tests
make test-e2e           # Chạy CLI + MCP E2E tests
make test-e2e-semantic  # E2E tests bao gồm semantic search
make lint               # Chạy golangci-lint
make cross-compile      # Build cho 6 nền tảng
make ui                 # Build lại embedded Web UI (cần pnpm)
```

### Cấu trúc project

```
cmd/knowns/          # Điểm vào của CLI
internal/
  cli/               # Cobra commands
  models/            # Domain models
  storage/           # File-based storage (.knowns/)
  server/            # HTTP server, SSE, WebSocket
    routes/          # REST API handlers
    workspace/       # Agent orchestrator, process manager, worktree
  mcp/               # MCP server (stdio)
  search/            # Semantic search (ONNX)
ui/                  # Embedded React UI (built assets)
tests/               # E2E tests
```

---

## Liên kết

- [npm](https://www.npmjs.com/package/knowns)
- [GitHub](https://github.com/knowns-dev/knowns)
- [Discord](https://discord.knowns.dev)
- [Changelog](./CHANGELOG.md)

Nếu muốn xem thêm về nguyên tắc thiết kế và định hướng dài hạn, hãy đọc [Philosophy](./PHILOSOPHY.md).

Nếu muốn xem thêm tài liệu kỹ thuật, hãy đọc [Architecture](./ARCHITECTURE.md) và [Contributing](./CONTRIBUTING.md).

---

## Star History

<a href="https://www.star-history.com/?repos=knowns-dev%2Fknowns&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=knowns-dev/knowns&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=knowns-dev/knowns&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=knowns-dev/knowns&type=date&legend=top-left" />
 </picture>
</a>

---

<p align="center">
  <strong>What your AI should have knowns.</strong><br>
  Dành cho các nhóm phát triển phần mềm đang pair cùng AI.
</p>
