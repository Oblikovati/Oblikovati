---
milestone: M01
feature: F02
name: Transient Curves
status: planned
---

# M01 · F02 — Transient Curves

Ownerless analytic curve value types with multiple construction conventions (by points, by center/radius/angles) used as sketch input geometry, feature paths, and query results.

## In scope

- Lines & line segments.
- Arcs (center/three-point) & circles.
- Ellipses & elliptical arcs (full & partial).
- Polylines.

## Out of scope

_None._

## Key API contracts delivered

- `Line`,`Line2d`,`LineSegment`,`LineSegment2d`
- `Arc3d`,`Arc2d`,`Circle`,`Circle2d`
- `EllipseFull`,`EllipseFull2d`,`EllipticalArc`,`EllipticalArc2d`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-016](PBI-016-lines-arcs-circles.md) | Lines, arcs, circles (2D & 3D) with multiple constructors |
| [PBI-017](PBI-017-ellipses-polylines.md) | Ellipses, elliptical arcs & polylines |
