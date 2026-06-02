---
milestone: M06
feature: F04
pbi: PBI-073
title: Dimensional constraints backed by parameters
status: planned
estimate: L
---

# PBI-073 — Dimensional constraints backed by parameters

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F04 Dimensional Constraints

## Goal

Implement the dimensional constraint set, each creating/owning a model parameter so its value is an editable expression.

## Scope / work

- Distance/angle/radius/diameter/arc-length.
- Parameter creation & binding (M02).
- Driving vs driven mode.

## API contracts (interfaces / enums / collections)

- `DimensionConstraint`s,`Parameter`(M02)

## Acceptance criteria

- Editing a dimension's parameter drives sketch geometry via the solver.
- Driven dimensions report but don't constrain.

## Depends on

_See feature dependencies._
