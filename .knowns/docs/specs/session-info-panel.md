---
title: Session Info Panel
description: Specification for Session Info Panel in Chat UI
createdAt: '2026-03-12T18:53:43.440Z'
updatedAt: '2026-03-12T18:53:43.440Z'
tags:
  - spec
  - draft
---

## Overview

Session Info Panel là một collapsible panel trong Chat UI, hiển thị thông tin chi tiết về session đang active, bao gồm metadata và context (files/folders) mà session đang sử dụng.

## Requirements

### Functional Requirements

- FR-1: Panel hiển thị collapsible bên trong Chat UI
- FR-2: Hiển thị session metadata: model, agent type, status, created/updated timestamps
- FR-3: Hiển thị context breakdown - danh sách files/folders mà session đang sử dụng
- FR-4: Cho phép expand/collapse để xem chi tiết context
- FR-5: Tự động cập nhật khi session thay đổi

### Non-Functional Requirements

- NFR-1: UI nhất quán với TaskHistoryPanel pattern
- NFR-2: Load context data khi panel mở (lazy load)
- NFR-3: Hiển thị loading state khi đang fetch data

## Acceptance Criteria

- [ ] AC-1: Panel hiển thị trong ChatPage, bên cạnh hoặc dưới session list
- [ ] AC-2: Session metadata hiển thị đầy đủ: model name, agent type, status badge, created/updated
- [ ] AC-3: Context section hiển thị danh sách files/folders với icon và path
- [ ] AC-4: Panel có thể collapse/expand
- [ ] AC-5: Loading state hiển thị khi đang fetch context
- [ ] AC-6: Empty state hiển thị khi không có context

## Scenarios

### Scenario 1: View Session Info
**Given** User đang ở Chat UI với session đang active
**When** User mở Session Info Panel
**Then** Hiển thị session metadata và context breakdown

### Scenario 2: No Context
**Given** Session mới tạo chưa có context
**When** User mở Session Info Panel
**Then** Hiển thị "No context available" message

### Scenario 3: Session Changes
**Given** Session có context đang active
**When** Context thay đổi (user thêm file mới)
**Then** Panel tự động cập nhật context list

## Technical Notes

- Sử dụng pattern collapsible giống TaskHistoryPanel
- API: Có thể cần thêm endpoint để lấy session context từ OpenCode
- Component: Tạo trong `ui/src/components/organisms/SessionInfoPanel.tsx`

## Open Questions

- [ ] API endpoint cho context - cần check OpenCode client có sẵn chưa?
- [ ] Format hiển thị context - list hay tree view?
