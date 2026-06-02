---
milestone: M00
feature: F03
name: Transient Object Collections
status: planned
---

# M00 · F03 — Transient Object Collections

Ownerless, non-persisted container and option types that are the lingua franca of method signatures: ordered bags, string→variant maps, and read-only enumerators, produced by a `TransientObjects` factory.

## In scope

- `ObjectCollection` ordered mutable bag.
- `NameValueMap` string→variant options bag.
- Typed `*Enumerator` read-only snapshots.
- `TransientObjects` factory.

## Out of scope

_None._

## Key API contracts delivered

- `ObjectCollection`
- `NameValueMap`
- `ObjectsEnumerator*`
- `TransientObjects`

## Depends on

F01, F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-008](PBI-008-object-collection.md) | ObjectCollection (ordered mutable bag) |
| [PBI-009](PBI-009-namevaluemap.md) | NameValueMap (string→variant options bag) |
| [PBI-010](PBI-010-enumerators-factory.md) | Typed enumerators & TransientObjects factory |
