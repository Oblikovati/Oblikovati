---
milestone: M21
feature: F06
pbi: PBI-211
title: Expose existing geometric constraints
status: planned
estimate: M
---

# PBI-211 — Expose existing geometric constraints

**Milestone:** M21  ·  **Feature:** F06 Geometric Constraints

## Goal

Make the ~17 already-implemented geometric constraints reachable through `/api` and the
Constrain ribbon panel, with enumeration/show/delete.

## Scope / work

- **/api:** `types.GeometricConstraintKind`; `wire.AddConstraintArgs` (sketch + Kind +
  entity refs), `MethodSketchAddConstraint/DeleteConstraint`; `client.Sketch.Constrain`
  group with one helper per kind.
- **/source:** `addin/router/sketch_constraints.go` mapping Kind → the existing
  `GeometricConstraints.AddXxx`; constraint enumeration (already partly in F01) + delete.
- **UI:** bring the generic `ConstraintTool` to a full Constrain panel (one command per
  kind, auto-commit on pick); e2e per representative kind.

## Acceptance criteria

- Dogfood + UI e2e: applying coincident/parallel/perpendicular/tangent/equal each reduces
  DOF as expected; delete removes it. `make ci` green.

## Depends on

PBI-200.
