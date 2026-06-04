---
milestone: M22
feature: F07
pbi: PBI-239
title: ConstraintStatus + defer/solve + DimensionsVisible (3D)
status: planned
estimate: S
---

# PBI-239 — 3D constraint status & DOF

**Milestone:** M22  ·  **Feature:** F07 Constraint Status & DOF

## Goal
Report a 3D sketch's constraint state without moving geometry, and expose defer/solve.

## Scope / work
- `model/sketch`: a `ConstraintStatus3D()` analysis reusing the shared DOF/Jacobian
  rank machinery (extract shared helper from the 2D path if needed, no duplication).
- `/api`: `MethodSketch3DConstraintStatus`; `ConstraintStatus3DResult` (reuse the 2D
  shape); `client.Sketch3D.ConstraintStatus`. DeferUpdates already in F01.
- router case.

## Acceptance criteria
- Unit: under/well/over cases report correct DOF + redundant counts; status is
  non-mutating (geometry unchanged).
- Dogfood; ≥98% on the new analysis; `make ci` green.

## Depends on
PBI-236, PBI-238.
