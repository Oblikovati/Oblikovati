---
milestone: M21
feature: F03
name: Conics & Splines
status: planned
---

# M21 · F03 — Conics & Splines

The curved free-form entities: ellipse, elliptical arc, the four spline families
(interpolation/fit, control-point, fixed, offset), and equation curves. Adds the
new model types backed by `kernel/geom` NURBS + an expression-driven parametric curve,
with sampling that feeds region/profile detection.

## In scope

- `SketchEllipse` (exists — bring to API/UI), `SketchEllipticalArc` (new).
- `SketchSpline` interpolation/fit (exists), `SketchControlPointSpline`,
  `SketchFixedSpline`, `SketchOffsetSpline` (new).
- `SketchEquationCurve` (new — parametric/explicit, expression-driven).

## Out of scope

- Offset of arbitrary chains (F05 offset op); fill regions (F04).

## Key API contracts delivered

- `SketchEntityKind` members: `ellipse`, `ellipticalArc`, `spline`, `controlSpline`,
  `fixedSpline`, `offsetSpline`, `equationCurve`
- `client.Sketch.{AddEllipse,AddEllipticalArc,AddSpline,AddControlPointSpline,...}`

## Depends on

F02; `kernel/geom` NURBS/ellipse; M02 expression engine (equation curves).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-203](PBI-203-ellipse-elliptical-arc.md) | Ellipse & elliptical arc |
| [PBI-204](PBI-204-splines-equation-curves.md) | Spline families & equation curves |
