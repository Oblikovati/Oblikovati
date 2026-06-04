---
milestone: M22
feature: F03
pbi: PBI-234
title: 3D splines (interp/control/fixed) + equation curve
status: planned
estimate: M
---

# PBI-234 — 3D splines + equation curve

**Milestone:** M22  ·  **Feature:** F03 Conics & Splines

## Goal
Add the 3D spline family and the parametric equation curve.

## Scope / work
- `model/sketch/splines_3d.go`: `Spline3D` (interpolation through fit points),
  `ControlPointSpline3D` (control polygon), `FixedSpline3D` (immutable), each over
  `kernel/geom.BSplineCurve`; `EquationCurve3D` (x(t)/y(t)/z(t) over [t0,t1], reuse the
  2D expression sampler extended to z).
- `/api`: entity kinds + `AddSketch3DEntityArgs` spline/equation fields (points, closed,
  zExpr, t0/t1); `client` helpers.
- router cases; UI spline tool + ribbon button.

## Acceptance criteria
- Unit ≥98% (sampling, closed loops, fit-point pass-through); dogfood; round-trip;
  ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-233.
