---
milestone: M07
feature: F04
pbi: PBI-085
title: PartComponentDefinition container
status: planned
estimate: M
---

# PBI-085 — PartComponentDefinition container

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F04 Part Component Definition Container

## Goal

Implement the part content object exposing bodies, bounding boxes, the geometry version string, and the hooks the feature engine and assemblies consume.

## Scope / work

- `SurfaceBodies`; range boxes.
- `ModelGeometryVersion` change-detection string.
- Wire to `PartDocument` (M03).

## API contracts (interfaces / enums / collections)

- `PartComponentDefinition`,`SurfaceBodies`,`Box`,`OrientedBox`

## Acceptance criteria

- A part document exposes its bodies and bounding boxes.
- Geometry version changes on every model edit.

## Depends on

_See feature dependencies._
