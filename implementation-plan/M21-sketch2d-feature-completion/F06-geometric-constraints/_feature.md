---
milestone: M21
feature: F06
name: Geometric Constraints
status: planned
---

# M21 · F06 — Geometric Constraints

Expose the full Inventor geometric-constraint set through `/api` and the Constrain ribbon
panel, and add the few the model lacks: ground (distinct from the internal fix), offset,
horizontal/vertical align, and pattern constraints. Includes constraint enumeration,
show, and delete.

## In scope

- Expose existing: coincident, collinear, concentric, parallel, perpendicular, horizontal,
  vertical, tangent, smooth (G2), symmetric, equal (length/radius), fix.
- Add: `Ground`, `Offset`, `HorizontalAlign`, `VerticalAlign`, `PatternConstraint`.
- `addConstraint(Kind)` discriminated method; delete/show.

## Out of scope

- Dimensional (driving) constraints (F07).

## Key API contracts delivered

- `types.GeometricConstraintKind` (full set, stable ids)
- `MethodSketchAddConstraint`, `MethodSketchDeleteConstraint`
- `client.Sketch.Constrain.{Coincident,Parallel,Tangent,Ground,Offset,...}`

## Depends on

F01; solver (`model/sketch/solver.go`).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-211](PBI-211-geometric-constraints-core.md) | Expose existing geometric constraints |
| [PBI-212](PBI-212-geometric-constraints-new.md) | Ground/offset/align/pattern constraints |
