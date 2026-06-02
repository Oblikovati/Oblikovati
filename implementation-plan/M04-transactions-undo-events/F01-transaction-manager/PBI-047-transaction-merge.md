---
milestone: M04
feature: F01
pbi: PBI-047
title: Transaction merge & suppression of change notifications
status: planned
estimate: M
---

# PBI-047 — Transaction merge & suppression of change notifications

**Milestone:** M04 Transactions, Undo & Events  ·  **Feature:** F01 Transaction Manager

## Goal

Support merging a transaction with the previous one (single undo step) and suppressing change events during long batch operations.

## Scope / work

- `MergeWithPrevious`.
- `SuppressChangeNotifications`.
- Coalesced post-batch update.

## API contracts (interfaces / enums / collections)

- `Transaction.MergeWithPrevious`,`SuppressChangeNotifications`

## Acceptance criteria

- Two merged edits undo as one step.
- A 1000-edit batch fires one coalesced update.

## Depends on

_See feature dependencies._
