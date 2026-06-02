---
milestone: M07
feature: F01
pbi: PBI-080
title: Topology↔geometry binding & containers
status: planned
estimate: M
---

# PBI-080 — Topology↔geometry binding & containers

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F01 Topology Model

## Goal

Bind each face/edge to its underlying transient surface/curve geometry and expose typed collections + range boxes.

## Scope / work

- `Face.Geometry`(surface),`Edge.Geometry`(curve).
- `SurfaceBodies` container; per-entity `RangeBox`.
- Solid/surface/lump/shell organization.

## API contracts (interfaces / enums / collections)

- `Face.Geometry`,`Edge.Geometry`,`SurfaceBodies`,`Box`

## Acceptance criteria

- A planar face returns a `Plane`; a circular edge returns a `Circle`/`Arc3d`.

## Depends on

_See feature dependencies._
