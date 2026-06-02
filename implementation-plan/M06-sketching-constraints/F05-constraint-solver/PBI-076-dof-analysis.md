---
milestone: M06
feature: F05
pbi: PBI-076
title: DOF analysis & over/under-constraint reporting
status: planned
estimate: M
---

# PBI-076 — DOF analysis & over/under-constraint reporting

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F05 Constraint Solver

## Goal

Report degrees of freedom, visualize them, and detect/flag over- and under-constrained conditions and redundant constraints.

## Scope / work

- DOF count & visualization.
- Over/under/redundant detection.
- Reject conflicting constraints.

## API contracts (interfaces / enums / collections)

- `Sketch.DegreesOfFreedom`, solver diagnostics

## Acceptance criteria

- A fully-constrained sketch reports 0 DOF.
- Adding a redundant constraint is flagged/rejected.

## Depends on

_See feature dependencies._
