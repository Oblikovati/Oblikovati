---
milestone: M22
feature: F03
name: Conics & Splines (3D)
status: partial (conics done; splines TODO)
---

# M22 · F03 — Conics & Splines (3D)

`SketchEllipse3D`, `SketchEllipticalArc3D`, and the 3D spline family
(`SketchSpline3D` interpolation, `SketchControlPointSpline3D`, `SketchFixedSpline3D`)
plus `SketchEquationCurve3D` (parametric x(t)/y(t)/z(t)). Thin wrappers over
`kernel/geom` (`EllipseFull`, `EllipticalArc`, `BSplineCurve`).

## Depends on
F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-233](PBI-233-conics.md) | 3D ellipse + elliptical arc entities + tools |
| [PBI-234](PBI-234-splines.md) | 3D splines (interp/control/fixed) + equation curve |
