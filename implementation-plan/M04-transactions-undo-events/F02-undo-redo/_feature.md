---
milestone: M04
feature: F02
name: Undo/Redo & Checkpoints
status: planned
---

# M04 · F02 — Undo/Redo & Checkpoints

Undo/redo driven by inverse operations + recompute (restoring model state, not raw geometry), plus cheap checkpoints to return to a state without naming a transaction.

## In scope

- Undo/redo stacks (committed/undone).
- `SetCheckPoint`/`GoToCheckPoint`.
- Clear-all; redo invalidation.

## Out of scope

_None._

## Key API contracts delivered

- `TransactionManager.UndoTransaction/RedoTransaction`
- `CheckPoint`,`CheckPointsEnumerator`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-048](PBI-048-undo-redo.md) | Undo/redo stacks with recompute restore |
| [PBI-049](PBI-049-checkpoints.md) | Checkpoints (SetCheckPoint/GoToCheckPoint) |
