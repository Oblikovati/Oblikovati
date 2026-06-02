---
milestone: M04
name: Transactions, Undo & Events
status: planned
---

# M04 — Transactions, Undo & Events

Make edits safe and observable. Undo is a transaction log over model mutations integrated with recompute — not a geometry snapshot stack. Events use the object/sink split with before/after timing and veto. This milestone delivers the `TransactionManager`, checkpoints/undo/redo, the event infrastructure, and the core application/document/modeling event sets plus the `ChangeManager`.

## Goals

- Every user-visible edit wrapped in a named, possibly nested transaction.
- Undo/redo and checkpoints that restore state via inverse ops + recompute.
- A uniform event infrastructure (object/sink, before/after, veto, context).
- The foundational application/document/modeling event sets and change processing.

## In scope

- `TransactionManager`, transactions (nested/global/merge).
- Checkpoints, undo/redo, committed/undone stacks.
- Event object/sink split, before/after, `HandlingCode` veto.
- `ApplicationEvents`/`DocumentEvents`/`ModelingEvents`, `ChangeManager`.

## Out of scope (handled elsewhere)

- Domain-specific events (sketch/assembly/drawing) ship with their milestones.
- UI command events — M05.

## Exit criteria

- A batch of edits forms one named undo step; undo/redo restores model + geometry via recompute.
- A before-event handler can veto a document close.
- Change notifications can be suppressed during batch operations.

## Depends on

M03

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Transaction Manager](F01-transaction-manager/_feature.md) | 2 | Named, nested, global transactions wrapping every edit. |
| **F02** | [Undo/Redo & Checkpoints](F02-undo-redo/_feature.md) | 2 | Replay-based undo/redo and lightweight checkpoints. |
| **F03** | [Event Infrastructure](F03-event-infrastructure/_feature.md) | 2 | Object/sink split, before/after timing, veto, context bag. |
| **F04** | [Core Events & Change Manager](F04-core-events-changes/_feature.md) | 2 | Application/Document/Modeling events and change processing. |
