---
milestone: M21
feature: F07
name: Dimensional Constraints
status: done
---

# M21 · F07 — Dimensional Constraints

The full driving/driven dimension set, each backed by a parameter so editing the value
re-solves the sketch. Expose the existing kinds and add the missing ones, plus
`ConstraintLimits` and the general Dimension tool.

## In scope

- Existing: TwoPointDistance, TwoLineAngle, RadiusDim, DiameterDim, ArcLengthDim.
- New: OffsetDim (point-to-line / parallel-distance), ThreePointAngle, EllipseRadiusDim,
  TangentDistanceDim, OffsetSplineDim, SplineFitPointConstraint.
- Driving ↔ driven toggle; `ConstraintLimits` (min/max); edit/drive value.

## Out of scope

- Auto-dimension (F08).

## Key API contracts delivered

- `types.DimensionConstraintKind` (full set, stable ids)
- `MethodSketchAddDimension`, `MethodSketchDriveDimension`
- `client.Sketch.Dimension.{Distance,Angle,Radius,Diameter,Offset,...}`

## Depends on

F01; M02 parameters.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-213](PBI-213-dimensions-core.md) | Expose linear/angular/radial dimensions |
| [PBI-214](PBI-214-dimensions-advanced.md) | Offset/3-point-angle/ellipse/tangent/spline dimensions |
