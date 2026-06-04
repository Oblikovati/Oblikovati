---
milestone: M05
feature: F05
pbi: PBI-065
title: Interaction (preview) graphics
status: done
estimate: M
---

# PBI-065 — Interaction (preview) graphics

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F05 Client Graphics

## Goal

Implement transient interaction graphics for command previews and manipulators that exist only during an interaction.

## Scope / work

- `InteractionGraphics` lifecycle tied to a command.
- Rubber-band/preview update on mouse move.

## API contracts (interfaces / enums / collections)

- `InteractionGraphics`

## Acceptance criteria

- A command shows a live preview that disappears on commit/cancel.

## Delivered

- `interactionGraphics.update` replaces a transient lane (overlay/preview) wholesale —
  the rubber-band/manipulator update path — and `interactionGraphics.clear` drops both
  lanes (`api/wire`, `api/client` `Graphics().Interaction()`).
- The overlay/preview lanes are command-scoped: the session clears them on tool
  commit/cancel and when a new tool starts (`app/session_input.go`), so previews vanish
  on commit/cancel. Regression tests in `app/client_graphics_test.go`.
- The declarative bulk-group model is shared with PBI-064 (one `clientgraphics.Store`,
  one `Build(cam)` builder).

## Depends on

_See feature dependencies._
