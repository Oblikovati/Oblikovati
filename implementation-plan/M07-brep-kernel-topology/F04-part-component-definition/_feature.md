---
milestone: M07
feature: F04
name: Part Component Definition Container
status: planned
---

# M07 · F04 — Part Component Definition Container

The `PartComponentDefinition` that holds a part's modeling content — its surface bodies, bounding boxes, model-geometry version, and rollback/end-of-part state — and is the root the feature engine (M08) operates within.

## In scope

- `PartComponentDefinition`; `SurfaceBodies`.
- `RangeBox`/`PreciseRangeBox`/`OrientedMinimumRangeBox`.
- `ModelGeometryVersion`; rollback/`EndOfPart` state.

## Out of scope

_None._

## Key API contracts delivered

- `PartComponentDefinition`,`ComponentDefinition`,`_PartComponentDefinition`
- `SurfaceBodies`,`Box`,`OrientedBox`

## Depends on

F01,F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-085](PBI-085-part-definition.md) | PartComponentDefinition container |
| [PBI-086](PBI-086-rollback-eop.md) | Rollback / End-of-Part state |
