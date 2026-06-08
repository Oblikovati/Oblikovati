---
milestone: M24
feature: F01
pbi: PBI-313
title: March-projected per-edge pcurves (non-self-intersecting boundary)
status: planned
estimate: M
---

# PBI-313 — March-projected per-edge pcurves (non-self-intersecting boundary)

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F01 On-surface pcurves & boundary

## Goal

Build a face's boundary `(u,v)` by marching `ParamNear` along each loop, producing a smooth,
non-self-intersecting pcurve where independent `ParamAt` self-intersects.

## Scope / work

- In `kernel/ops`, a `marchUV(s geom.BSplineSurface, loop []math.Point3) []math.Point2`: first
  point via `ParamAt` (grid-seeded), each subsequent point via `ParamNear` seeded from the
  previous. Apply to the outer loop and each hole.
- Guard the loop closure (last point's `(u,v)` should be near the first's) and a re-seed if a
  step diverges (the projection failed) — fall back to `ParamAt` for that point.
- Not yet wired into tessellation — exposed for F02/F03. (Keep `trimmedPatchMesh` as-is this PBI.)

## API contracts (interfaces / enums / collections)

- (internal) `ops.marchUV(surface, loop) []Point2`.

## Acceptance criteria

- On a constructed B-spline patch whose boundary makes independent `ParamAt` self-intersect,
  `marchUV` returns a boundary `(u,v)` polygon with **zero self-intersections** (segment-pair
  intersection test) — committed unit test.
- The pcurve's `PointAt` stays within tolerance of the input loop points (it is the same curve,
  re-parametrised smoothly).
- `go test ./kernel/ops/...` green; lint clean. No change to existing tessellation output yet
  (the OCC oracle volume is byte-identical to pre-PBI).

## Depends on

PBI-312 (`ParamNear`).
