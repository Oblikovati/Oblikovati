---
milestone: M05
feature: F04
pbi: PBI-062
title: Interaction & mouse/keyboard event pipeline
status: planned
estimate: M
---

# PBI-062 — Interaction & mouse/keyboard event pipeline

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F04 Interaction & Selection

## Goal

Implement the interaction event objects that a command uses to drive point/entity input, including mouse move/click and keyboard.

## Scope / work

- `InteractionEvents` lifecycle (start/stop).
- `MouseEvents`/`KeyboardEvents`.
- Point/entity input modes.

## API contracts (interfaces / enums / collections)

- `InteractionEvents`,`MouseEvents`,`KeyboardEvents`,`InteractionManager`

## Acceptance criteria

- A command collects a point and an entity via interaction events.

## Depends on

_See feature dependencies._
