---
milestone: M06
feature: F05
name: Constraint Solver
status: planned
---

# M06 · F05 — Constraint Solver

The numerical/geometric constraint solver that resolves a sketch from its constraints and dimensions, reports degrees of freedom and over/under-constrained states, and supports auto-dimensioning.

## In scope

- 2D & 3D solve.
- DOF computation; over/under-constrained detection.
- Auto-dimension; redundant-constraint rejection.

## Out of scope

_None._

## Key API contracts delivered

- (internal) ConstraintSolver
- `Sketch.SolveSketch`,`DegreesOfFreedom`

## Depends on

F03,F04.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-075](PBI-075-solver-core.md) | 2D/3D constraint solver core |
| [PBI-076](PBI-076-dof-analysis.md) | DOF analysis & over/under-constraint reporting |
