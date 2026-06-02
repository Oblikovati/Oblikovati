---
milestone: M01
feature: F03
pbi: PBI-019
title: BSpline/NURBS curves & surfaces with evaluators
status: planned
estimate: L
---

# PBI-019 — BSpline/NURBS curves & surfaces with evaluators

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F03 Transient Surfaces & Splines

## Goal

Implement NURBS curve and surface value types with control nets, knots, weights, and parametric evaluators.

## Scope / work

- Knot/weight/control-point representation.
- Evaluate point/derivatives; fit/approximate helpers.
- `IRxSurfaceEvaluator` interface.

## API contracts (interfaces / enums / collections)

- `BSplineCurve`,`BSplineSurface`,`IRxBSplineSurface`,`IRxSurfaceEvaluator`

## Acceptance criteria

- Evaluated points/derivatives match reference NURBS.
- Round-trips through loft/sweep surface generation.

## Depends on

_See feature dependencies._
