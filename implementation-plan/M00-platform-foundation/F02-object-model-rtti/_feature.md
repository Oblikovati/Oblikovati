---
milestone: M00
feature: F02
name: Core Object Model & RTTI
status: planned
---

# M00 · F02 — Core Object Model & RTTI

`ObjectTypeEnum` is the runtime type identification for a weakly-typed, polymorphic object graph. This feature establishes it plus the universal `Type`, `Parent`, and `Application` members and object equality.

## In scope

- `ObjectTypeEnum` with stable explicit numeric ids and reserved ranges.
- Universal `Type` on every object; `Parent`/`Application` back-pointers.
- Safe down-branching from `object` handles via `Type`.

## Out of scope

_None._

## Key API contracts delivered

- `ObjectTypeEnum` (7000+ stable members)
- `Type`/`Parent`/`Application` member convention

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-005](PBI-005-object-type-enum.md) | ObjectTypeEnum: stable numbered RTTI taxonomy |
| [PBI-006](PBI-006-universal-members.md) | Universal Type/Parent/Application members & navigation |
| [PBI-007](PBI-007-weak-typed-branching.md) | Variant handle down-branching helpers |
