---
title: Vibe Kanban
createdAt: '2026-03-05T03:44:03.428Z'
updatedAt: '2026-03-05T04:14:24.299Z'
description: >-
  Specification for Vibe Kanban - seamless task-to-AI assignment from kanban
  board with real-time progress and auto task updates
tags:
  - spec
  - approved
---
## Overview

Vibe Kanban biến kanban board thành trung tâm điều phối AI agent. User chỉ cần 1 click trên task card để giao cho AI (Claude/Gemini), theo dõi progress realtime ngay trên board, và nhận kết quả tự động cập nhật vào task khi agent hoàn thành.

Tận dụng 100% hạ tầng workspace đã có (@doc/specs/agent-workspace) — chỉ cần xây cầu nối giữa Kanban UI ↔ Workspace System.

## Goals

1. **1-click AI assignment** — Nút "Assign to AI" trên task card, auto tạo workspace + generate prompt + start agent
2. **Realtime progress trên kanban** — Task card hiển thị agent status (thinking/coding/done), phase progress, activity indicator
3. **Inline terminal** — Click vào task đang chạy → mở terminal panel ngay trong kanban view
4. **Auto task update** — Agent xong → auto check AC, append notes, move task sang "in-review"
5. **Bidirectional link** — Task ↔ Workspace luôn đồng bộ, xem workspace từ task hoặc ngược lại

## Requirements

### Functional Requirements

- FR-1: Task card trong kanban có nút/action "Assign to AI" khi task ở trạng thái todo/blocked
- FR-2: Click "Assign to AI" → dialog chọn agent **per phase** (với default từ config), confirm → auto tạo workspace từ task
- FR-3: System auto-generate prompt từ task title + description + acceptance criteria + linked docs
- FR-4: Task card hiển thị agent status overlay khi có workspace đang chạy (thinking indicator, phase badge, mini progress)
- FR-5: Click vào task card đang có agent chạy → mở inline terminal panel (slide-up hoặc drawer)
- FR-6: Khi agent hoàn thành (all phases done hoặc implement phase done):
  - Auto append phase outputs vào task implementation notes
  - Move task status sang "in-review"
  - Broadcast SSE event để kanban update realtime
- FR-7: Khi agent fail → task giữ nguyên status, hiển thị error indicator trên card, user có thể retry/skip phase
- FR-8: Task detail sheet hiển thị linked workspace info: status, phases, link sang workspace page
- FR-9: Workspace creation từ kanban tự động set `taskId` trên workspace
- FR-10: Nhiều task có thể chạy AI song song (mỗi task 1 workspace)
- FR-11: User có thể cấu hình default agent cho từng phase trong project config (ví dụ: research=claude, plan=gemini, implement=codex, review=claude)
- FR-12: AssignToAIDialog cho phép override agent cho từng phase riêng lẻ trước khi start
### Non-Functional Requirements

- NFR-1: Agent status update trên kanban card phải < 500ms latency (qua SSE)
- NFR-2: Không tăng bundle size quá 15KB (reuse existing components)
- NFR-3: Mobile responsive — agent indicator hiển thị tốt trên card nhỏ
- NFR-4: Terminal panel inline không block kanban interaction (có thể minimize/close)

## Acceptance Criteria

- [x] AC-1: Task card trong kanban hiển thị nút "Assign to AI" cho task chưa có workspace
- [x] AC-2: Dialog "Assign to AI" cho phép chọn agent type **per phase**, có default từ project config
- [x] AC-3: Prompt được auto-generate từ task metadata (title, description, AC, spec refs, parent/subtasks)
- [x] AC-4: Task card hiển thị realtime agent indicator khi workspace đang chạy (status + phase)
- [x] AC-5: Click task card đang chạy agent → mở inline terminal panel (xterm.js) với live output
- [x] AC-6: Agent hoàn thành → task implementation notes được auto-update với phase output
- [x] AC-7: Agent hoàn thành → task status auto chuyển sang "in-review"
- [x] AC-8: Agent fail → error indicator trên card, user có thể retry/skip từ kanban
- [x] AC-9: Task detail sheet hiển thị linked workspace info và actions
- [x] AC-10: Nhiều task có thể chạy AI đồng thời không conflict
- [x] AC-11: Mobile responsive — indicators và controls hoạt động trên mobile
- [x] AC-12: Project config `phaseAgentDefaults` cho phép set default agent cho từng phase
- [x] AC-13: Mỗi phase trong workspace có thể chạy agent khác nhau (ví dụ: plan=gemini, implement=claude)
## Architecture

### Process Proxy Layer

Node không spawn agent CLI trực tiếp. Thay vào đó dùng Go binary `agent-proxy` làm lớp trung gian:

```
Node.js (ProcessManager)
  │ spawn("dist/agent-proxy", ["claude", prompt, "--cwd", worktreePath])
  │
  └─▶ Go Binary (agent-proxy) — trong dist/
        │ spawn("claude", ["-p", prompt, "--output-format", "stream-json", "--verbose"])
        │
        └─▶ stdout: JSONL normalized events
              {"type":"init","text":"pid:1234","agent":"claude","ts":1772683506341}
              {"type":"thinking","text":"Analyzing...","agent":"claude","ts":...}
              {"type":"text","text":"JWT is...","agent":"claude","ts":...}
              {"type":"tool_use","text":"Read:...","agent":"claude","ts":...}
              {"type":"result","text":"Done","agent":"claude","ts":...}
              {"type":"exit","text":"code:0","agent":"claude","ts":...}
```

**Tại sao Go binary?**
- Node `child_process` có vấn đề stdout buffering khi spawn CLI agents
- Go `bufio.Scanner` đọc line-by-line ổn định, không mất event
- Go binary build sẵn trong `dist/`, không cần Go runtime khi deploy
- Clean env vars (xoá session markers để tránh nested session detection)
- Signal forwarding (SIGTERM/SIGINT → child process)
- Normalizer tích hợp sẵn cho Claude, Gemini (extensible cho Codex, OpenCode)

**Binary interface:**
```
agent-proxy <agent> <prompt> [--cwd <dir>] [--include-raw]
  agent:        claude | gemini | codex | ...
  prompt:       prompt string
  --cwd:        working directory for agent
  --include-raw: include original event in output (debug)

stdout: JSONL — mỗi dòng là 1 OutputEvent
stderr: error messages từ binary
exit code: forward từ agent process
```

**OutputEvent schema:**
```typescript
interface ProxyEvent {
  type: "init" | "thinking" | "text" | "tool_use" | "tool_result" | "result" | "error" | "stderr" | "exit";
  text?: string;
  agent: string;       // "claude" | "gemini"
  ts: number;           // Unix ms
  raw?: any;            // Original event (khi --include-raw)
}
```

### Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│  KANBAN BOARD                                                │
│  ┌─────────┐  ┌──────────────┐  ┌──────────┐  ┌─────────┐ │
│  │  TODO    │  │  IN PROGRESS │  │ IN REVIEW│  │  DONE   │ │
│  │         │  │              │  │          │  │         │ │
│  │ [Task A]│  │ [Task B 🤖]  │  │ [Task C] │  │ [Task D]│ │
│  │  [▸ AI] │  │  ◉ thinking  │  │          │  │         │ │
│  │         │  │  ████░░ 2/4  │  │          │  │         │ │
│  └─────────┘  └──────┬───────┘  └──────────┘  └─────────┘ │
│                       │ click                                │
│              ┌────────▼────────────────────────────────────┐ │
│              │  INLINE TERMINAL PANEL                      │ │
│              │  [Task B] Claude • Phase: Implement (3/4)   │ │
│              │  ┌──────────────────────────────────────┐   │ │
│              │  │ $ Reading src/auth/login.ts...       │   │ │
│              │  │ $ Writing JWT middleware...          │   │ │
│              │  └──────────────────────────────────────┘   │ │
│              │  [Stop] [Skip Phase] [View Full Terminal]   │ │
│              └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Full Pipeline

```
Browser (Kanban)
  │ POST /api/workspaces/from-task { taskId, phases }
  ▼
Express Server
  │ PhaseOrchestrator.start()
  │ ProcessManager.start()
  ▼
Node spawn("dist/agent-proxy", ["claude", prompt, "--cwd", worktreePath])
  ▼
Go Binary (agent-proxy)
  │ spawn("claude", ["-p", prompt, "--output-format", "stream-json", "--verbose"])
  │ parse JSONL → normalize → emit
  ▼
Node reads stdout line-by-line
  │ parse ProxyEvent JSON
  │ → outputBuffer (scrollback)
  │ → structuredOutput (phase result)
  │ → broadcastToClients(WebSocket)
  ▼
Browser WebSocket
  │ onOutput → xterm.js terminal
  │ onAgentEvent → AgentStatusOverlay
  │ onStatus → card indicator
  ▼
Kanban Card updates realtime
```
## Task: ${task.title}

## Description
${task.description || "No description provided."}

## Acceptance Criteria
${task.acceptanceCriteria.map((ac, i) => 
  `${i + 1}. ${ac.completed ? "[DONE]" : "[TODO]"} ${ac.text}`
).join("
")}
`;

  // Parent context (khi task là subtask)
  if (options.parentTask) {
    prompt += `
## Parent Task: ${options.parentTask.title}
${options.parentTask.description || ""}

### Parent Acceptance Criteria
${options.parentTask.acceptanceCriteria.map((ac, i) =>
  `${i + 1}. ${ac.completed ? "[DONE]" : "[TODO]"} ${ac.text}`
).join("
")}
`;
  }

  // Subtask context (khi task có subtasks)
  if (options.subtasks?.length) {
    prompt += `
## Subtasks
${options.subtasks.map(st => 
  `- [${st.status === "done" ? "DONE" : st.status.toUpperCase()}] ${st.title}`
).join("
")}
`;
  }

  if (task.spec) {
    prompt += `
## Spec Reference
See: ${task.spec}
`;
  }

  prompt += `
## Instructions
Complete all TODO acceptance criteria above. Follow project conventions.
Work in the workspace worktree directory.
`;

  return prompt.trim();
}
```
## Task: ${task.title}

## Description
${task.description || "No description provided."}

## Acceptance Criteria
${task.acceptanceCriteria.map((ac, i) => 
  `${i + 1}. ${ac.completed ? "[DONE]" : "[TODO]"} ${ac.text}`
).join("
")}
`;

  // Parent context (khi task là subtask)
  if (options.parentTask) {
    prompt += `
## Parent Task: ${options.parentTask.title}
${options.parentTask.description || ""}

### Parent Acceptance Criteria
${options.parentTask.acceptanceCriteria.map((ac, i) =>
  `${i + 1}. ${ac.completed ? "[DONE]" : "[TODO]"} ${ac.text}`
).join("
")}
`;
  }

  // Subtask context (khi task có subtasks)
  if (options.subtasks?.length) {
    prompt += `
## Subtasks
${options.subtasks.map(st => 
  `- [${st.status === "done" ? "DONE" : st.status.toUpperCase()}] ${st.title}`
).join("
")}
`;
  }

  if (task.spec) {
    prompt += `
## Spec Reference
See: ${task.spec}
`;
  }

  prompt += `
## Instructions
Complete all TODO acceptance criteria above. Follow project conventions.
Work in the workspace worktree directory.
`;

  return prompt.trim();
}
```
## Task: ${task.title}

## Description
${task.description || "No description provided."}

## Acceptance Criteria
${task.acceptanceCriteria.map((ac, i) => 
  `${i + 1}. ${ac.completed ? "[DONE]" : "[TODO]"} ${ac.text}`
).join("
")}

${task.spec ? `## Spec Reference
See: ${task.spec}` : ""}

## Instructions
Complete all TODO acceptance criteria above. Follow project conventions.
Work in the workspace worktree directory.
  `.trim();
}
```

## UI Components

### 1. AgentStatusOverlay

Overlay nhỏ trên task card khi có workspace đang chạy:

```
┌──────────────────────────────┐
│ #42 Add JWT auth        HIGH │
│                              │
│ Implement auth middleware... │
│                              │
│ ◉ Thinking    ████░░░ 2/4   │  ← AgentStatusOverlay
│ 🤖 claude                    │
└──────────────────────────────┘
```

States:
- `idle` — Workspace tạo nhưng chưa start → "Ready" badge
- `running` — Agent đang chạy → animated indicator + phase number
- `stopped/error` — Hiển thị error icon + retry button
- `completed` — Check mark, "AI Done" badge

### 2. AssignToAIButton

Nút nhỏ trên task card (chỉ hiện khi hover hoặc trên mobile luôn hiện):

```
┌──────────────────────────────┐
│ #42 Add JWT auth        HIGH │
│                              │
│ Implement auth middleware    │
│                              │
│ 📋 2/5     [▸ Assign to AI] │  ← Button
└──────────────────────────────┘
```

### 3. AssignToAIDialog

Dialog xác nhận trước khi giao — mỗi phase chọn agent riêng (prompt chỉ preview, không edit):

```
┌──────────────────────────────────────────┐
│  Assign Task to AI                       │
│                                          │
│  Task: #42 Add JWT auth                  │
│                                          │
│  Phases & Agents:                        │
│  ☑ Research    [Claude ▼]                │
│  ☑ Plan        [Gemini ▼]               │
│  ☑ Implement   [Claude ▼]               │
│  ☑ Review      [Claude ▼]               │
│                                          │
│  Prompt Preview:                         │
│  ┌──────────────────────────────────┐   │
│  │ ## Task: Add JWT auth            │   │
│  │ ## Acceptance Criteria           │   │
│  │ 1. [TODO] User can login...      │   │
│  └──────────────────────────────────┘   │
│                                          │
│  [Cancel]               [Start AI ▸]    │
└──────────────────────────────────────────┘
```

- Mỗi phase có dropdown chọn agent (chỉ hiện agents available trên máy)
- Default lấy từ project config `phaseAgentDefaults`
- Uncheck phase để skip
### 4. InlineTerminalPanel

Panel slide-up từ dưới kanban board:

```
┌─────────────────────────────────────────────────────┐
│  KANBAN BOARD (scrollable, height reduced)           │
│  ...cards...                                         │
├─────────────────────────────────────────────────────┤
│  ▾ #42 Add JWT auth │ Claude │ Implement (3/4)      │  ← Header (draggable resize)
│  ┌─────────────────────────────────────────────────┐│
│  │ [Terminal output via xterm.js]                   ││
│  │ Reading src/auth/login.ts                        ││
│  │ Writing middleware...                            ││
│  └─────────────────────────────────────────────────┘│
│  [Stop] [Skip Phase] [Retry] [Open Full ↗]  [✕]    │
└─────────────────────────────────────────────────────┘
```

- Dùng lại `useTerminalWebSocket` hook hiện có
- Resize bằng drag handle
- "Open Full" navigate sang `/workspaces/:id`
- Close không stop agent, chỉ ẩn panel

## Scenarios

### Scenario 1: Happy Path — Assign and Complete

**Given** task #42 "Add JWT auth" ở trạng thái "todo" trên kanban board
**When** user click "Assign to AI" → chọn Claude, all 4 phases → click "Start AI"
**Then**
- Workspace auto-created với taskId=#42
- Task card hiển thị animated "🤖 Running" indicator
- Phases chạy tuần tự: research → plan → implement → review
- Khi tất cả phases done → task notes được update, status chuyển "in-review"
- Task card trên kanban tự move sang cột "In Review"

### Scenario 2: Watch Progress

**Given** task #42 đang có agent chạy (phase: implement)
**When** user click vào task card
**Then**
- Inline terminal panel slide up từ dưới
- Terminal hiển thị live output từ agent
- Phase progress bar cho thấy "3/4 — Implement"
- User có thể đóng panel mà agent vẫn chạy

### Scenario 3: Agent Fails

**Given** task #42 đang chạy phase "implement"
**When** agent exit với error code
**Then**
- Task card hiển thị error indicator (đỏ)
- Phase "implement" marked as "failed"
- Workspace status → "stopped"
- User có thể click card → thấy error output → click "Retry" hoặc "Skip"
- Task status KHÔNG thay đổi (vẫn ở cột cũ)

### Scenario 4: Multiple Tasks Running

**Given** task #42 và #43 đều đang chạy agent
**When** user nhìn kanban board
**Then**
- Cả 2 card đều hiển thị agent indicator riêng
- Click card nào → terminal panel switch sang workspace tương ứng
- Không có conflict giữa 2 agent (isolated worktrees)

### Scenario 5: Mobile View

**Given** user trên mobile device
**When** task đang chạy agent
**Then**
- Agent indicator hiển thị compact (icon + phase number only)
- Terminal panel full-width bottom sheet
- Touch-friendly controls

### Scenario 6: Task Already Has Workspace

**Given** task #42 đã có workspace (idle hoặc stopped)
**When** user click "Assign to AI"
**Then**
- Dialog hiện workspace đã tồn tại
- Options: "Restart" (reuse workspace), "New" (tạo workspace mới), "Cancel"

## Implementation Phases

### Phase 1: Backend — Task-to-Workspace Bridge

- Endpoint `POST /api/workspaces/from-task`
- Prompt generation logic từ task metadata
- Auto-update task khi workspace completes (server-side hook trong PhaseOrchestrator)
- Helper: `getWorkspaceByTaskId()` trong WorkspaceStore

### Phase 2: State Management — Task↔Workspace Map

- Hook `useTaskWorkspaceMap` — derive Map<taskId, Workspace> từ WorkspaceContext
- Extend WorkspaceContext: thêm `getWorkspaceByTaskId()`
- SSE subscription đảm bảo realtime sync

### Phase 3: Kanban Card — Agent Overlay

- Component `AgentStatusOverlay` — render indicator trên TaskKanbanCard
- Component `AssignToAIButton` — trigger assign dialog
- Integrate vào `TaskKanbanCard` trong Board.tsx

### Phase 4: Assign Dialog

- Component `AssignToAIDialog` — agent/phase picker + prompt preview
- Prompt generation (client-side preview, server-side actual)
- Call `POST /api/workspaces/from-task` on confirm

### Phase 5: Inline Terminal Panel

- Component `InlineTerminalPanel` — reuse `useTerminalWebSocket`
- Resizable panel layout trong kanban page
- Phase progress bar + action buttons (stop/retry/skip)

### Phase 6: Task Detail Integration

- Thêm workspace section trong TaskDetailSheet
- Hiển thị: workspace status, phases, link to full workspace view
- Actions: start/stop/retry inline

## File Structure

### New Files

```
# Go binary (repo riêng, build ra dist/)
agent-proxy/
├── main.go                           # Entry point, CLI args, spawn + stream
├── normalizers.go                    # Claude, Gemini event normalizers
└── go.mod

# Node.js — workspace system
src/server/workspace/
├── process-manager.ts                # Spawn agent-proxy binary, manage lifecycle
├── worktree-manager.ts               # Git worktree create/remove
├── workspace-store.ts                # JSON persistence
├── phase-orchestrator.ts             # Multi-phase state machine
├── prompt-generator.ts              # Generate prompt từ task metadata
└── task-updater.ts                   # Auto-update task on completion

src/server/workspace/executors/
├── base.ts                           # AgentExecutor interface
├── proxy-agent-process.ts            # Spawn agent-proxy binary, parse ProxyEvent
├── registry.ts                       # Executor registry
└── utils.ts                          # resolveCommand, cleanEnv

# UI components
src/ui/components/organisms/kanban/
├── AgentStatusOverlay.tsx            # Agent indicator trên card
├── AssignToAIButton.tsx              # Button trigger dialog
├── AssignToAIDialog.tsx              # Agent/phase selection dialog
└── InlineTerminalPanel.tsx           # Slide-up terminal (xterm.js)

src/ui/hooks/
└── use-task-workspace-map.ts         # Task↔Workspace mapping hook
```

### Modified Files

```
src/models/workspace.ts                            # AgentType → dynamic string
src/ui/components/organisms/Board.tsx              # Integrate overlays + terminal panel
src/ui/components/organisms/task-detail/           # Add workspace section
src/ui/contexts/WorkspaceContext.tsx                # Add getWorkspaceByTaskId
src/server/routes/workspaces.ts                    # Add /from-task endpoint
src/server/index.ts                                # WebSocket setup for terminal
```
## Dependencies

### Runtime
- **agent-proxy** Go binary trong `dist/` — Node spawn binary này thay vì spawn CLI trực tiếp
- xterm.js — đã có trong project
- Reuse: `WorkspaceContext`, `SSEContext`, `KanbanProvider`, WebSocket

### Build
- Go 1.21+ để build agent-proxy (cross-compile cho darwin/linux)
- Không thêm Node package mới

### Model Changes

`AgentType` trong `src/models/workspace.ts` chuyển sang dynamic:

```typescript
// Hiện tại (cứng):
export type AgentType = "claude" | "gemini";

// Sau (dynamic, dựa trên agent-proxy + executor registry):
export type AgentType = string;
```

`DEFAULT_PHASE_AGENTS` đọc từ project config:

```typescript
export function getPhaseAgentDefaults(config?: ProjectConfig): Record<PhaseType, string> {
  return {
    research: config?.phaseAgentDefaults?.research ?? "claude",
    plan: config?.phaseAgentDefaults?.plan ?? "claude",
    implement: config?.phaseAgentDefaults?.implement ?? "claude",
    review: config?.phaseAgentDefaults?.review ?? "claude",
  };
}
```

### Node ↔ Go Binary Interface

Node spawn Go binary và đọc stdout line-by-line:

```typescript
import { spawn } from "child_process";
import { createInterface } from "readline";

const proc = spawn("dist/agent-proxy", [agent, prompt, "--cwd", cwd]);
const rl = createInterface({ input: proc.stdout });

rl.on("line", (line) => {
  const event: ProxyEvent = JSON.parse(line);
  // event.type: init | thinking | text | tool_use | tool_result | result | error | exit
  handleEvent(event);
});
```
## Open Questions

- [x] ~~Có nên cho phép edit prompt trước khi start, hay chỉ preview?~~ → **Chỉ preview**, không cho edit để giữ UX đơn giản
- [x] ~~Khi task có subtasks, agent có nên nhận context của parent task không?~~ → **Có**, prompt generator sẽ include parent task context
- [x] ~~Inline terminal nên dùng xterm.js đầy đủ hay simplified text view để nhẹ hơn?~~ → **xterm.js đầy đủ**, reuse existing TerminalPanel component
