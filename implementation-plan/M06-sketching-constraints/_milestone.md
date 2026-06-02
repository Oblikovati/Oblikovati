---
milestone: M06
name: 2D/3D Sketching & Constraint Solver
status: planned
---

# M06 — 2D/3D Sketching & Constraint Solver

The constrained sketch environment that produces the profiles and paths features consume. A sketch is a 2D/3D constraint program: geometry + geometric constraints + dimensional constraints, resolved by a solver, yielding regions/profiles. The 2D solver is decoupled from the 3D kernel behind a clean profile boundary so the two evolve independently.

## Goals

- Planar and 3D sketches hosted on planes/faces with full entity sets.
- Geometric and dimensional constraints with inference.
- A robust 2D/3D constraint solver with DOF/over-under-constraint reporting.
- Profiles and paths derived from sketch regions for downstream features.

## In scope

- `PlanarSketch`/`Sketch3D`/`DrawingSketch`; sketch plane.
- Sketch entities (lines/arcs/circles/ellipses/splines/points); project geometry.
- Geometric & dimensional constraints; inference; `ConstraintLimits`.
- Constraint solver, DOF analysis; profiles/paths/regions.

## Out of scope (handled elsewhere)

- Turning a profile into a solid (M08).
- Drawing-sheet sketches' annotation use (M14).

## Exit criteria

- A fully-constrained sketch solves deterministically and reports 0 DOF.
- A closed region yields a `Profile` usable by extrude/revolve.
- Editing a driving dimension updates sketch geometry via the solver.

## Depends on

M01, M02, M05

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Sketch Infrastructure](F01-sketch-infrastructure/_feature.md) | 2 | Planar/3D/drawing sketches and their hosting plane/face. |
| **F02** | [Sketch Entities](F02-sketch-entities/_feature.md) | 2 | Lines, arcs, circles, ellipses, splines, points in sketches. |
| **F03** | [Geometric Constraints](F03-geometric-constraints/_feature.md) | 3 | Coincident, parallel, tangent, concentric, symmetry, etc. |
| **F04** | [Dimensional Constraints](F04-dimensional-constraints/_feature.md) | 2 | Driving/driven linear, angular, radial, diameter dimensions. |
| **F05** | [Constraint Solver](F05-constraint-solver/_feature.md) | 2 | The 2D/3D solver: DOF analysis, over/under-constraint, auto-dim. |
| **F06** | [Profiles & Paths](F06-profiles-paths/_feature.md) | 2 | Regions, profiles, and paths consumed by features. |
