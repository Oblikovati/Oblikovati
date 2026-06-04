---
milestone: M22
feature: F06
pbi: PBI-238
title: Full 3D dimension set + driving/driven + edit/drive API + tools
status: done
estimate: M
---

# PBI-238 — 3D dimensional constraints

**Milestone:** M22  ·  **Feature:** F06 Dimensional Constraints

## Goal
Add the remaining 3D dimensions and expose add/drive over `/api`.

## Scope / work
- `model/sketch/dimension_3d.go` (extend): `AddLineLength`, `AddRadius`,
  `AddPointPlaneDistance`, `AddTwoLineAngle`, `AddSplineLength`, each measuring over the
  3D DOFs and backed by a parameter; `SetDriven`.
- `/api`: `Dimension3DConstraintKind` members; `AddSketch3DDimensionArgs`,
  `DriveSketch3DDimensionArgs`; `Dimension3DInfo`; `client` helpers.
- router: `sketch3d.addDimension/driveDimension/dimensions`.
- UI: dimension tool + value edit dialog + ribbon button.

## Acceptance criteria
- Unit ≥98%: each measured value correct; driving dim pins the DOF, driven reports only.
- Dogfood add/drive; a frame + dims → 0 DOF; round-trip; ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-232.
