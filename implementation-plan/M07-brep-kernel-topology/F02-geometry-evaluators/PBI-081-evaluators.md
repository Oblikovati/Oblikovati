---
milestone: M07
feature: F02
pbi: PBI-081
title: Topology evaluators (point/normal/tangent/curvature)
status: planned
estimate: L
---

# PBI-081 — Topology evaluators (point/normal/tangent/curvature)

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F02 Geometry Evaluators

## Goal

Implement the evaluator objects for faces/edges with parametric queries used across measurement, snapping, and feature math.

## Scope / work

- `GetPointAtParam`,`GetNormal`,`GetTangent`,`GetCurvature`.
- `GetParamAtPoint`,closest-point.
- Area/length; containment.

## API contracts (interfaces / enums / collections)

- `FaceEvaluator`,`EdgeEvaluator`,`CurveEvaluator`,`SurfaceEvaluator`

## Acceptance criteria

- Evaluator outputs match analytic reference within `PointTolerance`.

## Depends on

_See feature dependencies._
