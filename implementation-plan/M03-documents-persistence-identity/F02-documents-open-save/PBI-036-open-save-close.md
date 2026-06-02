---
milestone: M03
feature: F02
pbi: PBI-036
title: Open/OpenWithOptions/Save/Close lifecycle
status: planned
estimate: M
---

# PBI-036 — Open/OpenWithOptions/Save/Close lifecycle

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F02 Documents Collection & Open/Save

## Goal

Implement open (with `NameValueMap` options & visibility), save variants, and close-all (incl. unreferenced-only).

## Scope / work

- `Open`,`OpenWithOptions`,`Save`,`SaveAs`.
- `CloseAll(UnreferencedOnly)`.
- Dirty handling on close.

## API contracts (interfaces / enums / collections)

- `Documents.Open/OpenWithOptions`,`CloseAll`

## Acceptance criteria

- A saved document reopens identically.
- Options bag is honored; unreferenced-only close works.

## Depends on

_See feature dependencies._
