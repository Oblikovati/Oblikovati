---
milestone: M22
feature: F05
pbi: PBI-236
title: Core 3D constraints (parallel/perp/tangent/smooth/midpoint/ground) + API
status: planned
estimate: M
---

# PBI-236 — Core 3D geometric constraints

**Milestone:** M22  ·  **Feature:** F05 Geometric Constraints

## Goal
Add the core 3D geometric constraints and expose add/list/delete over `/api`.

## Scope / work
- `model/sketch/constraints_3d.go` (extend): `Parallel3D`, `Perpendicular3D`,
  `Tangent3D`, `Smooth3D` (G2), `Midpoint3D`, `Ground3D` (fix a point's DOFs).
  Each implements `Constraint` (residuals + variables over `*math.Scalar`).
- `/api`: `Geometric3DConstraintKind` members; `AddSketch3DConstraintArgs` (kind +
  entity refs); `Constraint3DInfo`; `client` helpers.
- router: `sketch3d.addConstraint/constraints/deleteConstraint`.
- UI: constraint tools + ribbon buttons.

## Acceptance criteria
- Unit ≥98%: each residual zero when satisfied; a fully-constrained frame → 0 DOF.
- Dogfood add/list/delete; round-trip; ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-232.
