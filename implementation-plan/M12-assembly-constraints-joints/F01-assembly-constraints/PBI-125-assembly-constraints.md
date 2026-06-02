---
milestone: M12
feature: F01
pbi: PBI-125
title: Assembly constraint set & solver
status: planned
estimate: XL
---

# PBI-125 — Assembly constraint set & solver

**Milestone:** M12 Assembly: Constraints, Joints, Motion & Representations  ·  **Feature:** F01 Assembly Constraints

## Goal

Implement the assembly constraints (proxy-based geometry inputs) and the solver that positions occurrences, with limits and redundancy/health reporting.

## Scope / work

- Mate/flush/angle/tangent/insert/symmetry `Add` methods.
- Constraint solver positioning occurrences.
- Limits; redundant/over-constrained detection → health.

## API contracts (interfaces / enums / collections)

- `AssemblyConstraint(s)`,`MateConstraint`,`FlushConstraint`,`AngleConstraint`,`TangentConstraint`,`InsertConstraint`,`ConstraintLimits`

## Acceptance criteria

- Mating two faces positions a component; over-constraint is flagged; solver is stable.
- Constraint inputs are proxies (assembly-space).

## Depends on

_See feature dependencies._
