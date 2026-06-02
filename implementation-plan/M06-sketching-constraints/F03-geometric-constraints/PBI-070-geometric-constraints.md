---
milestone: M06
feature: F03
pbi: PBI-070
title: Geometric constraint set (2D)
status: planned
estimate: L
---

# PBI-070 — Geometric constraint set (2D)

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F03 Geometric Constraints

## Goal

Implement the full set of 2D geometric constraints with creation, query, and deletion, each as a solver relation.

## Scope / work

- All constraint types + `Add` methods.
- Constraint enumeration on entities.
- Delete & redundancy handling.

## API contracts (interfaces / enums / collections)

- `CoincidentConstraint`…`SymmetryConstraint`,`GeometricConstraints`

## Acceptance criteria

- Each constraint correctly relates entities and is enforced by the solver.

## Depends on

_See feature dependencies._
