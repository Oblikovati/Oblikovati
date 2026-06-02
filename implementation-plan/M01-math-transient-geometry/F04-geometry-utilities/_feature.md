---
milestone: M01
feature: F04
name: Geometry Utilities & Factory
status: planned
---

# M01 · F04 — Geometry Utilities & Factory

The single `TransientGeometry` factory (the one discoverable construction point for all value geometry) plus bounding boxes and core geometric queries (intersection, distance, containment) used throughout modeling.

## In scope

- `TransientGeometry` factory for all value types.
- `Box`/`Box2d`/`OrientedBox`.
- Curve/surface intersection, min-distance, projection.

## Out of scope

_None._

## Key API contracts delivered

- `TransientGeometry`
- `Box`,`Box2d`,`OrientedBox`
- `GeometryUtilities` / intersection APIs

## Depends on

F01-F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-020](PBI-020-transient-geometry-factory.md) | TransientGeometry factory (single construction point) |
| [PBI-021](PBI-021-boxes.md) | Bounding boxes (Box, Box2d, OrientedBox) |
| [PBI-022](PBI-022-geometry-queries.md) | Geometric queries: intersection, distance, projection |
