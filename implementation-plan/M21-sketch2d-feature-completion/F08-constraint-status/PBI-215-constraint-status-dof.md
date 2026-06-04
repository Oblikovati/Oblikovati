---
milestone: M21
feature: F08
pbi: PBI-215
title: Constraint status, DOF reporting, defer/solve, auto-dimension
status: planned
estimate: M
---

# PBI-215 — Constraint status, DOF reporting, defer/solve, auto-dimension

**Milestone:** M21  ·  **Feature:** F08 Constraint Status & DOF

## Goal

Expose the solver's constraint analysis so the UI can report whether a sketch is fully /
over / under-constrained, show DOF, defer solving during batch edits, and auto-dimension.

## Scope / work

- **/source:** `model/sketch/status.go` — compute `ConstraintStatus` per entity + per
  sketch from the solver's DOF partition (rank of the constraint Jacobian); list
  under-constrained entities; `DeferUpdates` gate around `Solve`; `AutoDimension` (greedy
  add dimensions until DOF=0). Reuse the existing solver/DOF in `solver.go`.
- **/api:** `types.ConstraintStatus`; `wire.ConstraintStatusResult` (per-sketch status, DOF
  count, under-constrained entity ids); `MethodSketchConstraintStatus`;
  `client.Sketch.ConstraintStatus/AutoDimension` + `SetDeferUpdates`.
- **UI:** DOF readout + under-constrained highlight in `head/ui/sketch_overlay.go`;
  Finish-Sketch warns when under-constrained; e2e asserting status transitions.

## Acceptance criteria

- A fully-constrained rectangle reports `WellConstrained`, DOF=0; removing a dimension →
  `UnderConstrained` with the right entity flagged; an over-constrained add → `Over`.
- `DeferUpdates` solves once on resume; `AutoDimension` reaches DOF=0. `make ci` green.

## Depends on

PBI-213 (dimensions), F05 solver.
