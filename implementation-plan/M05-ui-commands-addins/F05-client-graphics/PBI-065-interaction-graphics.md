---
milestone: M05
feature: F05
pbi: PBI-065
title: Interaction (preview) graphics
status: planned
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

## Depends on

_See feature dependencies._
