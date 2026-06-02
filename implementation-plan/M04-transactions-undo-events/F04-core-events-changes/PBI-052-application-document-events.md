---
milestone: M04
feature: F04
pbi: PBI-052
title: Application & Document event sets
status: planned
estimate: M
---

# PBI-052 — Application & Document event sets

**Milestone:** M04 Transactions, Undo & Events  ·  **Feature:** F04 Core Events & Change Manager

## Goal

Implement the application-centric and document-specific event sets with before/after and veto.

## Scope / work

- `ApplicationEvents` (OnNewDocument/OnOpen/OnSave/OnClose/OnActivate/OnQuit…).
- `DocumentEvents` (save/close/select…).

## API contracts (interfaces / enums / collections)

- `ApplicationEvents`,`DocumentEvents`

## Acceptance criteria

- Opening/saving/closing fires the right before/after events.
- A close can be vetoed.

## Depends on

_See feature dependencies._
