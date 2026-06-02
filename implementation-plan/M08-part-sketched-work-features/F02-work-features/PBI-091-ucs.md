---
milestone: M08
feature: F02
pbi: PBI-091
title: User coordinate systems
status: planned
estimate: M
---

# PBI-091 — User coordinate systems

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F02 Work Features (Datums)

## Goal

Implement UCS objects (a triad of plane/axis/point) for local modeling frames.

## Scope / work

- `UserCoordinateSystem` creation & transform.
- Component datums for assembly use.

## API contracts (interfaces / enums / collections)

- `UserCoordinateSystem`,`UserCoordinateSystems`

## Acceptance criteria

- A UCS provides local planes/axes/origin usable by features and constraints.

## Depends on

_See feature dependencies._
