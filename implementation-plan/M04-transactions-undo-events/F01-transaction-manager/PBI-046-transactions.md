---
milestone: M04
feature: F01
pbi: PBI-046
title: Transaction lifecycle (start/end/abort, nested, global)
status: planned
estimate: L
---

# PBI-046 — Transaction lifecycle (start/end/abort, nested, global)

**Milestone:** M04 Transactions, Undo & Events  ·  **Feature:** F01 Transaction Manager

## Goal

Implement transactions as the unit of undo: open with a display name, record mutations, end (commit) or abort (rollback), with nesting and cross-document global transactions.

## Scope / work

- `StartTransaction(doc,name)`/`End`/`Abort`.
- Parent/child nesting; `StartGlobalTransaction`.
- Mutation recording (op + inverse).

## API contracts (interfaces / enums / collections)

- `TransactionManager`,`Transaction`,`TransactionStateEnum`

## Acceptance criteria

- Aborting restores pre-transaction state exactly.
- A nested transaction commits into its parent.
- Display names appear as undo labels.

## Depends on

_See feature dependencies._
