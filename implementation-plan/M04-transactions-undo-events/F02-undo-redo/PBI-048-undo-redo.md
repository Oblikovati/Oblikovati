---
milestone: M04
feature: F02
pbi: PBI-048
title: Undo/redo stacks with recompute restore
status: planned
estimate: L
---

# PBI-048 — Undo/redo stacks with recompute restore

**Milestone:** M04 Transactions, Undo & Events  ·  **Feature:** F02 Undo/Redo & Checkpoints

## Goal

Implement undo/redo over the transaction log, restoring model state by replaying inverse mutations and triggering recompute.

## Scope / work

- `UndoTransaction`/`RedoTransaction`.
- Committed/undone enumerators.
- Redo invalidation on new edit.

## API contracts (interfaces / enums / collections)

- `TransactionManager`,`TransactionsEnumerator`

## Acceptance criteria

- Undo then redo returns to identical model + geometry.
- A new edit clears the redo stack.

## Depends on

_See feature dependencies._
