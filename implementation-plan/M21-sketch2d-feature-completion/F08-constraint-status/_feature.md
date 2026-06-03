---
milestone: M21
feature: F08
name: Constraint Status & DOF
status: planned
---

# M21 · F08 — Constraint Status & DOF

Surface the solver's analysis to the user: per-entity and per-sketch `ConstraintStatus`
(well/over/under-constrained), the remaining degree-of-freedom count and which entities
carry them, `DeferUpdates`, explicit `Solve`, and auto-dimension.

## In scope

- `ConstraintStatus` enum + per-entity/per-sketch reporting from the solver Jacobian/DOF.
- DOF count + the under-constrained entities (for UI coloring).
- `DeferUpdates` (batch edits, solve once); explicit `Solve`.
- Auto-dimension (add the minimal dimension set to reach 0 DOF).

## Out of scope

- The actual ImGui colors (handled in `head/ui`); this delivers the data + overlay hooks.

## Key API contracts delivered

- `types.ConstraintStatus`; `MethodSketchConstraintStatus`
- `client.Sketch.{ConstraintStatus,AutoDimension}` + `SetDeferUpdates`

## Depends on

F05 solver (`model/sketch/solver.go`, DOF analysis).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-215](PBI-215-constraint-status-dof.md) | Constraint status, DOF reporting, defer/solve, auto-dimension |
