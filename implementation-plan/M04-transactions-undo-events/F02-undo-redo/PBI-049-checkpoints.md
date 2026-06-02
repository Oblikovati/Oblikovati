---
milestone: M04
feature: F02
pbi: PBI-049
title: Checkpoints (SetCheckPoint/GoToCheckPoint)
status: planned
estimate: S
---

# PBI-049 — Checkpoints (SetCheckPoint/GoToCheckPoint)

**Milestone:** M04 Transactions, Undo & Events  ·  **Feature:** F02 Undo/Redo & Checkpoints

## Goal

Implement lightweight checkpoints to capture and return to a model state without a named transaction.

## Scope / work

- `SetCheckPoint`/`GoToCheckPoint`.
- Checkpoint enumeration & disposal.

## API contracts (interfaces / enums / collections)

- `CheckPoint`,`CheckPointsEnumerator`

## Acceptance criteria

- Returning to a checkpoint restores the captured state.

## Depends on

_See feature dependencies._
