---
milestone: M04
feature: F01
name: Transaction Manager
status: planned
---

# M04 · F01 — Transaction Manager

The `TransactionManager` that opens/ends/aborts named units of undo, supports nested and cross-document global transactions, and merges adjacent transactions.

## In scope

- `StartTransaction`/`End`/`Abort`.
- Nested + `StartGlobalTransaction`.
- `MergeWithPrevious`; transaction state & ids.

## Out of scope

_None._

## Key API contracts delivered

- `TransactionManager`,`Transaction`,`TransactionsEnumerator`,`TransactionStateEnum`

## Depends on

M03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-046](PBI-046-transactions.md) | Transaction lifecycle (start/end/abort, nested, global) |
| [PBI-047](PBI-047-transaction-merge.md) | Transaction merge & suppression of change notifications |
