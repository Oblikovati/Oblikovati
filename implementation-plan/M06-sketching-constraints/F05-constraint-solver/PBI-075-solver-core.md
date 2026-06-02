---
milestone: M06
feature: F05
pbi: PBI-075
title: 2D/3D constraint solver core
status: planned
estimate: XL
---

# PBI-075 — 2D/3D constraint solver core

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F05 Constraint Solver

## Goal

Implement the constraint solver that resolves entity positions from geometric + dimensional constraints, robustly and deterministically.

## Scope / work

- Constraint equation assembly; Newton/decomposition solve.
- Deterministic results; under-constrained freedom handling.
- Incremental re-solve on edit.

## API contracts (interfaces / enums / collections)

- (internal) ConstraintSolver; `Sketch.Solve`

## Acceptance criteria

- A fully-constrained sketch solves to a unique solution.
- Editing a dimension re-solves stably.

## Depends on

_See feature dependencies._

## Notes

Second only to reference keys in difficulty. Consider an established 2D solver approach (graph decomposition + numeric). Keep it behind the sketch boundary.
