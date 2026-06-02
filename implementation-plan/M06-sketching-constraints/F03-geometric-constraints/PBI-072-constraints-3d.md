---
milestone: M06
feature: F03
pbi: PBI-072
title: 3D sketch constraints
status: planned
estimate: M
---

# PBI-072 — 3D sketch constraints

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F03 Geometric Constraints

## Goal

Implement the 3D-sketch constraint variants for spatial geometry.

## Scope / work

- `*Constraint3D` set.
- 3D solver integration.

## API contracts (interfaces / enums / collections)

- `CoincidentConstraint3D`,`CollinearConstraint3D`,`ConcentricConstraint3D`,`EqualConstraint3D`,`CustomConstraint3D`

## Acceptance criteria

- 3D constraints relate 3D sketch entities and solve.

## Depends on

_See feature dependencies._
