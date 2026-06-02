---
milestone: M06
feature: F04
pbi: PBI-074
title: Constraint limits & 3D dimensions
status: planned
estimate: M
---

# PBI-074 — Constraint limits & 3D dimensions

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F04 Dimensional Constraints

## Goal

Implement min/max limits on dimensional constraints (for drive/animation) and the 3D dimension variants.

## Scope / work

- `ConstraintLimits` on dimensions.
- `DimensionConstraint3D` set.

## API contracts (interfaces / enums / collections)

- `ConstraintLimits`,`DimensionConstraint3D`,`DimensionConstraints3D`

## Acceptance criteria

- A limited dimension respects min/max; 3D dimensions solve.

## Depends on

_See feature dependencies._
