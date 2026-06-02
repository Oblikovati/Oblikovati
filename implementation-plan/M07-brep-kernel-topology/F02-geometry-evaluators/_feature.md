---
milestone: M07
feature: F02
name: Geometry Evaluators
status: planned
---

# M07 · F02 — Geometry Evaluators

The evaluators that compute points, normals, tangents, curvatures, parameters, and containment on model topology — the numeric services snapping, measurement, and downstream features rely on.

## In scope

- `FaceEvaluator`/`EdgeEvaluator`/`CurveEvaluator`/`SurfaceEvaluator`.
- Point/normal/tangent/curvature/area/length.
- Param↔point; closest-point; containment.

## Out of scope

_None._

## Key API contracts delivered

- `FaceEvaluator`,`EdgeEvaluator`,`CurveEvaluator`,`SurfaceEvaluator`,`IRxSurfaceEvaluator`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-081](PBI-081-evaluators.md) | Topology evaluators (point/normal/tangent/curvature) |
