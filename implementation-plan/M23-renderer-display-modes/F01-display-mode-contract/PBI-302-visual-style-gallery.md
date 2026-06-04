---
milestone: M23
feature: F01
pbi: PBI-302
title: Visual Style ribbon gallery + command + end-to-end select
status: planned
estimate: M
---

# PBI-302 — Visual Style ribbon gallery + command + end-to-end select

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F01 Display-Mode Contract & Style Plumbing

## Goal

Let a user pick any display mode in the running app — the UI half of the Definition of
Done for the display-mode contract.

## Scope / work

- A **Visual Style gallery** in the View tab (`head/ui/`): one entry per
  `DisplayModeEnum` member, current selection highlighted, picking one sets the active
  view's `DisplayMode` through the same path the `api/client` uses (PBI-300).
- A `CommandDefinition` registered in `app/commands_standard.go`
  (`NewCommand(id,name,panel,…).WithTab("View").WithIcon(…).WithTooltip(…)`), plus the
  interaction that applies the chosen mode to the viewport.
- Modes whose passes are not yet implemented (pre-F02/F03/F04) are shown but visibly
  marked unavailable, so the gallery is complete and honest as features land.

## API contracts (interfaces / enums / collections)

- (UI) Visual Style gallery control; View-tab `CommandDefinition`
- Reuses PBI-300 `view.setDisplayMode` — **no** new wire surface.

## Acceptance criteria

- End-to-end test (mirroring `TestExtrudeViaCommandAlias`): drive the command → select a
  mode in the gallery → assert the active view's `DisplayMode` and the renderer's active
  `VisualStyle` both reflect it.
- Selecting an already-implemented mode (Shaded/Wireframe/ShadedWithEdges) changes the
  rendered draw stream accordingly (asserted on the offscreen/null backend).
- Gallery selection and `api/client` `setDisplayMode` converge on the same state (the UI
  is not a second source of truth).

## Depends on

PBI-300 (contract), PBI-301 (the styles to select), M05 (command framework, ribbon).
