---
milestone: M05
feature: F06
pbi: PBI-070
title: Native ImGui style/picker/combo verbs
status: done
estimate: S
---

# PBI-070 — Native ImGui style/picker/combo verbs

**Milestone:** M05 · **Feature:** F06 UI Theming & Appearance

## Goal

Add the Dear ImGui verbs theming needs to the native seam.

## Scope / work

- `head/internal/native` (C++ `imgui_wrap.cpp` + Go `imgui.go`):
  `SetStyleColor(name, rgba)` mapping an ImGui color **name** → enum (version-robust),
  `ColorEdit4`, `BeginCombo` / `EndCombo`.
- `head/internal/native/viewport.{cpp,go}`: `SetViewportClear` (themed 3D pass clear).

## API contracts

- None (internal native binding).

## Acceptance criteria

- Head builds with cgo; `SetStyleColor` with an unknown name is a no-op; existing in-window
  tests stay green.

## Depends on

ADR-0004.
