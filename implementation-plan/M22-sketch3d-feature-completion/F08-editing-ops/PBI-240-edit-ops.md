---
milestone: M22
feature: F08
pbi: PBI-240
title: Move/rotate/copy/delete 3D entities + API + tools
status: done (model+API; UI in F12)
estimate: M
---

# PBI-240 — 3D editing operations

**Milestone:** M22  ·  **Feature:** F08 Editing & Reference Ops

## Goal
Transform and delete 3D sketch entities through `/api` + tools.

## Scope / work
- `model/sketch/edit_ops_3d.go`: apply a `math.Matrix` (translate/rotate) to selected
  entities' points; `Copy` duplicates; `Delete` removes + detaches constraints.
- `/api`: `MethodSketch3DTransform` (`Sketch3DTransformArgs`: entity ids, op, vector/
  axis/angle, copy); `client` helpers.
- router case; UI move/rotate/copy/delete tools + ribbon buttons.

## Acceptance criteria
- Unit ≥98%: transform composes correctly; delete detaches dependents.
- Dogfood; round-trip; ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-232.
