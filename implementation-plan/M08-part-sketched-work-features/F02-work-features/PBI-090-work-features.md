---
milestone: M08
feature: F02
pbi: PBI-090
title: Work planes/axes/points by relationship
status: planned
estimate: L
---

# PBI-090 — Work planes/axes/points by relationship

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F02 Work Features (Datums)

## Goal

Implement datum features with the full set of definitional relationships (offset, angle, three-point, tangent, intersection…), each recomputing parametrically.

## Scope / work

- `WorkPlanes.AddByPlaneAndOffset`/`ByThreePoints`/etc.
- `WorkAxes`/`WorkPoints` relationship constructors.
- Definition objects; adaptivity.

## API contracts (interfaces / enums / collections)

- `WorkPlane(s)`,`WorkAxis/Axes`,`WorkPoint(s)`,`*Definition`

## Acceptance criteria

- An offset work-plane moves when its driving parameter changes.
- Datums serve as sketch planes and feature inputs.

## Depends on

_See feature dependencies._
