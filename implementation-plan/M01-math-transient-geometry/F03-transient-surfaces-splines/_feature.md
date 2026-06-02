---
milestone: M01
feature: F03
name: Transient Surfaces & Splines
status: planned
---

# M01 · F03 — Transient Surfaces & Splines

The surface value types (analytic and free-form) underpinning B-rep faces and surfacing features, including NURBS curves/surfaces with evaluators.

## In scope

- Plane, cylinder, cone, sphere, torus surfaces.
- `BSplineCurve`/`BSplineSurface` (NURBS).
- Surface/curve evaluators.

## Out of scope

_None._

## Key API contracts delivered

- `Plane`,`Cylinder`,`Cone`,`Sphere`,`Torus`
- `BSplineCurve`,`BSplineSurface`
- `IRxBSplineSurface`,`IRxSurfaceEvaluator`

## Depends on

F01,F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-018](PBI-018-analytic-surfaces.md) | Analytic surfaces (plane/cylinder/cone/sphere/torus) |
| [PBI-019](PBI-019-nurbs.md) | BSpline/NURBS curves & surfaces with evaluators |
