---
milestone: M22
feature: F11
name: Surface-Derived Curves
status: planned
---

# M22 · F11 — Surface-Derived Curves

The 3D-sketch curves defined by reference to part surfaces, built on F10's kernel:
`IntersectionCurve` (two faces), `OnFaceCurve` (a sketched curve lying on a face),
`ProjectToSurfaceCurve` (project a curve onto a face), `SilhouetteCurve` (face outline
for a direction), and `OffsetCurve3` (offset of a 3D curve). Each is a reference-key-
bound, recompute-driven entity with its own collection + `/api` + tool.

## Depends on
F10 (kernel), F08 (reference keys / Include), M07 (faces).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-245](PBI-245-intersection-onface-project.md) | IntersectionCurve + OnFaceCurve + ProjectToSurfaceCurve |
| [PBI-246](PBI-246-silhouette-offset.md) | SilhouetteCurve + OffsetCurve3 |
