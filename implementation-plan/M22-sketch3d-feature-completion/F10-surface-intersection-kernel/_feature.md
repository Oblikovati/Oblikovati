---
milestone: M22
feature: F10
name: Surface-Intersection Kernel
status: partial (PBI-243 surface↔curve done; PBI-244 surface↔surface TODO)
---

# M22 · F10 — Surface-Intersection Kernel

The novel kernel work this milestone requires: in `kernel/geom`, **surface↔curve**
intersection, **surface↔surface** intersection, **point-projection-to-surface**, and
**silhouette** extraction — over both the analytic surfaces (`Plane`/`Cylinder`/`Cone`/
`Sphere`/`Torus`, closed-form where possible) and `BSplineSurface` (numerical
marching/subdivision). Pure, GPU-free, property/metamorphic tested. This is the
foundation F11's surface-derived curves stand on.

## Depends on
M01/M07 `kernel/geom` surfaces + curves, the `nurbs_eval` evaluator.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-243](PBI-243-surface-curve-projection.md) | Surface↔curve intersection + point projection to surface |
| [PBI-244](PBI-244-surface-surface-silhouette.md) | Surface↔surface intersection + silhouette extraction |
