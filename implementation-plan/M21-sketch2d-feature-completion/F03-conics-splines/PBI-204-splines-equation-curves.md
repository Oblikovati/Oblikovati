---
milestone: M21
feature: F03
pbi: PBI-204
title: Spline families & equation curves
status: planned
estimate: L
---

# PBI-204 — Spline families & equation curves

**Milestone:** M21  ·  **Feature:** F03 Conics & Splines

## Goal

Complete the spline entity family — interpolation/fit (exists), control-point, fixed,
offset — plus expression-driven equation curves, all sampled into the profile pipeline.

## Scope / work

- **/source:** `model/sketch` — `ControlPointSpline` (NURBS control polygon),
  `FixedSpline` (immutable derived from projected/included geometry), `OffsetSpline`
  (offset of a parent curve at a distance), `EquationCurve` (parametric x(t)/y(t) or
  explicit y(x) via the M02 expression engine). Each backed by `kernel/geom`; sampling +
  serialize round-trip; solver hooks for control points.
- **/api:** `addEntity` kinds + `client` helpers (`AddControlPointSpline`,
  `AddFixedSpline`, `AddOffsetSpline`, `AddEquationCurve`).
- **UI:** spline tool (click control/fit points), equation-curve dialog
  (`head/ui/sketch_equation_curve_dialog.go`: expressions + t-range), e2e.

## Acceptance criteria

- Dogfood + UI e2e create each; control-spline edit moves the curve; equation curve
  `x=cos t, y=sin t` samples a unit circle (tolerance). Round-trip preserves all.
- `make ci` green.

## Depends on

PBI-203.
