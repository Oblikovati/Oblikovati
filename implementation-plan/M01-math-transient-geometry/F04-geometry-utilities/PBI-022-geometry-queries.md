---
milestone: M01
feature: F04
pbi: PBI-022
title: Geometric queries: intersection, distance, projection
status: planned
estimate: M
---

# PBI-022 — Geometric queries: intersection, distance, projection

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F04 Geometry Utilities & Factory

## Goal

Implement core geometric queries between transient entities used by snapping, constraints, and measurement.

## Scope / work

- Curve-curve / curve-surface intersection.
- Minimum distance; closest point; projection.

## API contracts (interfaces / enums / collections)

- `GeometryUtilities` / intersection methods on TransientGeometry

## Acceptance criteria

- Queries match reference within `PointTolerance`.

## Depends on

_See feature dependencies._
