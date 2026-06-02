---
milestone: M07
feature: F03
pbi: PBI-083
title: Tessellation & display faceting
status: planned
estimate: M
---

# PBI-083 — Tessellation & display faceting

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F03 Boolean & Modeling Operations

## Goal

Implement tolerance-driven tessellation of faces/edges to triangle meshes and polylines for the renderer (M16) and export (M17).

## Scope / work

- Chordal-tolerance faceting.
- Edge polylines; normals/UVs.
- LOD/quality settings.

## API contracts (interfaces / enums / collections)

- `SurfaceBody`/`Face` tessellation API

## Acceptance criteria

- Faceting honors chordal tolerance; mesh is watertight per face.

## Depends on

_See feature dependencies._
