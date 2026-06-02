---
milestone: M06
feature: F03
name: Geometric Constraints
status: planned
---

# M06 · F03 — Geometric Constraints

The geometric (non-dimensional) constraints that relate sketch entities, plus constraint inference during sketching and the 3D-sketch constraint variants.

## In scope

- Coincident/collinear/parallel/perpendicular/tangent/concentric/equal/symmetry/horizontal/vertical/smooth/fix.
- 3D constraint variants.
- Constraint inference & glyphs.

## Out of scope

_None._

## Key API contracts delivered

- `CoincidentConstraint`,`ParallelConstraint`,`PerpendicularConstraint`,`TangentConstraint`,`ConcentricConstraint`,`CollinearConstraint`,`EqualConstraint`,`SymmetryConstraint`,`HorizontalConstraint`,`VerticalConstraint`
- `*Constraint3D` variants, `GeometricConstraints`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-070](PBI-070-geometric-constraints.md) | Geometric constraint set (2D) |
| [PBI-071](PBI-071-constraint-inference.md) | Constraint inference during sketching |
| [PBI-072](PBI-072-constraints-3d.md) | 3D sketch constraints |
