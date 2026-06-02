---
milestone: M01
feature: F02
pbi: PBI-016
title: Lines, arcs, circles (2D & 3D) with multiple constructors
status: planned
estimate: M
---

# PBI-016 — Lines, arcs, circles (2D & 3D) with multiple constructors

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F02 Transient Curves

## Goal

Implement line/segment/arc/circle value types with the by-points and by-geometry construction overloads exposed on `TransientGeometry`.

## Scope / work

- `CreateLine(Segment)`, `CreateArc3d`/`ByThreePoints`, `CreateCircle`.
- 2D equivalents.
- Evaluation hooks (point at param, tangent).

## API contracts (interfaces / enums / collections)

- `Line(2d)`,`LineSegment(2d)`,`Arc3d`,`Arc2d`,`Circle`,`Circle2d`

## Acceptance criteria

- Three-point arc reproduces expected center/radius.
- All curves marshal by value.

## Depends on

_See feature dependencies._
