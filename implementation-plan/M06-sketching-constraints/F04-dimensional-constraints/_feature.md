---
milestone: M06
feature: F04
name: Dimensional Constraints
status: planned
---

# M06 · F04 — Dimensional Constraints

The dimensional constraints that parametrically size a sketch, each backed by a parameter (M02), supporting driving/driven modes, equations, and limits.

## In scope

- Linear/angular/radius/diameter/arc-length dimensions.
- Driving vs driven; parameter backing; `ConstraintLimits`.
- 2D & 3D dimension constraints.

## Out of scope

_None._

## Key API contracts delivered

- `DimensionConstraint`,`DimensionConstraints`,`TwoPointDistanceDimConstraint`,`AngleConstraint`,`RadiusDimConstraint`,`DiameterDimConstraint`,`ArcLengthDimConstraint`
- `DimensionConstraint3D`,`DimensionConstraints3D`,`ConstraintLimits`

## Depends on

F02,F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-073](PBI-073-dimensional-constraints.md) | Dimensional constraints backed by parameters |
| [PBI-074](PBI-074-constraint-limits.md) | Constraint limits & 3D dimensions |
