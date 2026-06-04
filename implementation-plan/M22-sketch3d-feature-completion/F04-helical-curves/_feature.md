---
milestone: M22
feature: F04
name: Helical Curves
status: done (model+API; UI in F12)
---

# M22 · F04 — Helical Curves

A new analytic `kernel/geom.Helix3d` primitive and the `HelicalCurve` 3D-sketch entity
with Inventor's four definition modes (pitch+height, pitch+revolution, revolution+
height, spiral), plus the `HelicalConstraint3D` that ties a helix to its axis. Helices
are the canonical sweep path for threads/springs (feeds M10).

## Depends on
F02, `kernel/geom` curve interface (`Curve3`).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-235](PBI-235-helical-curves.md) | Helix3d kernel + HelicalCurve entity (4 modes) + constraint + tool |
