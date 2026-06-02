---
milestone: M05
feature: F03
pbi: PBI-060
title: Dockable windows & environments/workspaces
status: planned
estimate: M
---

# PBI-060 — Dockable windows & environments/workspaces

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F03 Ribbon, Browser & Docking UI

## Goal

Implement dockable windows for add-in panels and the environment system that switches ribbon/browser per document type and editing mode.

## Scope / work

- `DockableWindows` create/dock/float.
- `Environment`/`EnvironmentManager` activation.
- UI swap on document/mode change.

## API contracts (interfaces / enums / collections)

- `DockableWindows`,`DockableWindow`,`Environment`,`EnvironmentManager`,`Environments`

## Acceptance criteria

- A dockable panel docks/floats and persists layout.
- Switching documents swaps the active environment UI.

## Depends on

_See feature dependencies._
