---
milestone: M00
feature: F03
pbi: PBI-010
title: Typed enumerators & TransientObjects factory
status: planned
estimate: S
---

# PBI-010 — Typed enumerators & TransientObjects factory

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F03 Transient Object Collections

## Goal

Provide read-only enumerator snapshots returned by queries and a single factory to create all transient containers.

## Scope / work

- `*Enumerator` immutability semantics.
- `TransientObjects.CreateObjectCollection/CreateNameValueMap`.

## API contracts (interfaces / enums / collections)

- `TransientObjects`
- `ObjectsEnumerator`, `*Enumerator`

## Acceptance criteria

- Query results are immutable enumerators distinct from owning collections.
- Factory creates each transient type.

## Depends on

_See feature dependencies._
