---
milestone: M22
feature: F05
name: Geometric Constraints (3D)
status: done (model+API; UI in F12)
---

# M22 · F05 — Geometric Constraints (3D)

The full Inventor 3D geometric constraint set, extending `model/sketch/constraints_3d.go`
(which already has coincident/collinear/concentric/equal/custom): parallel,
perpendicular, tangent, smooth, midpoint, ground; parallel-to-X/Y/Z-axis;
parallel-to-XY/XZ/YZ-plane; spline-fit-points; bend. Exposed via `sketch3d.addConstraint`
(discriminated on `Kind`), `sketch3d.constraints` (list), `sketch3d.deleteConstraint`.

## Depends on
F02 (entities to constrain), F01 (spine).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-236](PBI-236-geometric-core.md) | Core constraints: parallel/perp/tangent/smooth/midpoint/ground + API |
| [PBI-237](PBI-237-axis-plane-bend.md) | Parallel-to-axis/plane, spline-fit-points, bend + API + tools |
