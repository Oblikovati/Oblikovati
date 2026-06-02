---
milestone: M08
feature: F02
name: Work Features (Datums)
status: planned
---

# M08 · F02 — Work Features (Datums)

Parametric construction geometry defined by relationships (offset plane, axis through points, point at intersection) that recompute like features and are valid inputs wherever no model face yet exists.

## In scope

- `WorkPlane`/`WorkAxis`/`WorkPoint` by relationship.
- `UserCoordinateSystem`.
- Adaptivity; fixed vs parametric.

## Out of scope

_None._

## Key API contracts delivered

- `WorkPlane`,`WorkPlanes`,`WorkAxis`,`WorkAxes`,`WorkPoint`,`WorkPoints`,`UserCoordinateSystem`,`UserCoordinateSystems`
- `WorkPlaneDefinition`,`WorkAxisDefinition`,`WorkPointDefinition`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-090](PBI-090-work-features.md) | Work planes/axes/points by relationship |
| [PBI-091](PBI-091-ucs.md) | User coordinate systems |
