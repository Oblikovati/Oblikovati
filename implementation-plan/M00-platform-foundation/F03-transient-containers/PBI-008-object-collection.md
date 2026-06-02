---
milestone: M00
feature: F03
pbi: PBI-008
title: ObjectCollection (ordered mutable bag)
status: planned
estimate: S
---

# PBI-008 — ObjectCollection (ordered mutable bag)

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F03 Transient Object Collections

## Goal

Implement the general-purpose ordered, 1-based, heterogeneous collection used to pass entity sets into operations.

## Scope / work

- Add/Remove/RemoveByObject/Clear/Count/indexer.
- `IEnumerable` support.

## API contracts (interfaces / enums / collections)

- `ObjectCollection`

## Acceptance criteria

- Round-trips a set of mixed entities into a feature operation (e.g. SetAffectedBodies).
- Enumeration order is stable.

## Depends on

_See feature dependencies._
