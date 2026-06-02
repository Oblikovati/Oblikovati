---
milestone: M03
feature: F06
pbi: PBI-044
title: AttributeSets/Attributes on any model object
status: planned
estimate: M
---

# PBI-044 — AttributeSets/Attributes on any model object

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F06 Attributes & Metadata

## Goal

Implement named, typed key/value attribute sets attachable to any object and persisted with the model, keyed by reference key so they survive recompute.

## Scope / work

- `AttributeSets`/`AttributeSet`/`Attribute` CRUD.
- `ValueType`; query/filter by set/name.
- Reference-key anchoring for topology attributes.

## API contracts (interfaces / enums / collections)

- `AttributeSets`,`AttributeSet`,`Attribute`,`ValueTypeEnum`,`AttributeManager`

## Acceptance criteria

- An attribute on a face survives recompute and reload.
- Add-in private data round-trips.

## Depends on

_See feature dependencies._
