---
milestone: M22
feature: F05
pbi: PBI-237
title: Parallel-to-axis/plane, spline-fit-points, bend constraints + API + tools
status: planned
estimate: M
---

# PBI-237 — Axis/plane/bend 3D constraints

**Milestone:** M22  ·  **Feature:** F05 Geometric Constraints

## Goal
Add the orientation-to-origin-frame constraints, the spline fit-points constraint, and
the bend constraint.

## Scope / work
- `model/sketch/constraints_3d.go` (extend): `ParallelToAxis3D` (X/Y/Z),
  `ParallelToPlane3D` (XY/XZ/YZ) — residuals over the origin frame; `SplineFitPoints3D`;
  `BendConstraint3D` (ties a bend's radius/tangency).
- `/api`: kinds + args (axis/plane selector); `client` helpers.
- router cases; UI tools + ribbon buttons.

## Acceptance criteria
- Unit ≥98%; dogfood; round-trip; ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-236.
