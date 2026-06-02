---
milestone: M01
feature: F02
pbi: PBI-017
title: Ellipses, elliptical arcs & polylines
status: planned
estimate: S
---

# PBI-017 — Ellipses, elliptical arcs & polylines

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F02 Transient Curves

## Goal

Implement full/partial ellipses and polyline value types for sketch and import use.

## Scope / work

- `CreateEllipseFull(2d)`, `CreateEllipticalArc(2d)`.
- Polyline value type.

## API contracts (interfaces / enums / collections)

- `EllipseFull(2d)`,`EllipticalArc(2d)`,`Polyline*`

## Acceptance criteria

- Minor/major ratio and axis vectors honored.
- Sweep/parameterization correct.

## Depends on

_See feature dependencies._
