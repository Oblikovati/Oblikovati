---
milestone: M22
feature: F06
name: Dimensional Constraints (3D)
status: done (model+API; UI in F12)
---

# M22 · F06 — Dimensional Constraints (3D)

The full Inventor 3D dimension set, extending `model/sketch/dimension_3d.go` (which has
`AddDistance` only): two-point distance, line length, radius, point-and-plane distance,
two-line angle, spline length. Each backed by a model parameter, driving or driven,
editable via `sketch3d.driveDimension`.

## Depends on
F02, F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-238](PBI-238-dimensions.md) | Full 3D dimension set + driving/driven + edit/drive API + tools |
