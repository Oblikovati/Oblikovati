---
milestone: M22
feature: F07
name: Constraint Status & DOF (3D)
status: done (model+API; UI in F12)
---

# M22 · F07 — Constraint Status & DOF (3D)

Non-mutating constraint analysis for 3D sketches: `sketch3d.constraintStatus` (status +
DOF + variables/equations/redundant), `DeferUpdates`, `DimensionsVisible`, and the solve
outcome (well/under/over, converged). Reuses the 2D status machinery (the solver is
dimension-agnostic).

## Depends on
F05, F06.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-239](PBI-239-constraint-status.md) | ConstraintStatus + defer/solve + DimensionsVisible (3D) |
