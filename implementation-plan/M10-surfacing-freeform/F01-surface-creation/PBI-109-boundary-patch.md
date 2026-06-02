---
milestone: M10
feature: F01
pbi: PBI-109
title: Boundary patch & ruled surface
status: planned
estimate: L
---

# PBI-109 — Boundary patch & ruled surface

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F01 Surface Creation

## Goal

Implement boundary-patch (fill closed boundaries with tangency/curvature conditions) and ruled-surface features.

## Scope / work

- `BoundaryPatchDefinition`/`Loops` with conditions.
- `RuledSurface` (normal/tangent/perpendicular).

## API contracts (interfaces / enums / collections)

- `BoundaryPatchFeature(s)`,`BoundaryPatchDefinition`,`BoundaryPatchLoop(s)`,`RuledSurfaceFeature(s)`

## Acceptance criteria

- A closed boundary patches with G1/G2 conditions honored.

## Depends on

_See feature dependencies._
