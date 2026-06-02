---
milestone: M10
feature: F04
pbi: PBI-115
title: Mesh features & mesh topology
status: planned
estimate: M
---

# PBI-115 — Mesh features & mesh topology

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F04 Mesh, Imported Geometry & Mold

## Goal

Implement mesh features wrapping imported tessellated geometry with mesh face/edge/vertex access and feature-set grouping.

## Scope / work

- `MeshFeature`/`MeshFeatureSet`.
- Mesh topology entities.
- Selection & conversion hooks.

## API contracts (interfaces / enums / collections)

- `MeshFeature(s)`,`MeshFeatureSet(s)`,`MeshFace`,`MeshEdge`,`MeshVertex`

## Acceptance criteria

- An imported STL appears as a mesh feature with selectable facets.

## Depends on

_See feature dependencies._
