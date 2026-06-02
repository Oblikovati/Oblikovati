---
milestone: M14
feature: F03
pbi: PBI-142
title: Centerlines, GD&T & datum frames
status: planned
estimate: L
---

# PBI-142 — Centerlines, GD&T & datum frames

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F03 Dimensions & Annotations

## Goal

Implement centerlines/center marks and GD&T annotations (feature control frames, datum reference frames, surface texture) on annotation planes, associative to topology.

## Scope / work

- Centerline/center-mark auto & manual.
- `FeatureControlFrame` (tolerance/datums).
- `DatumReferenceFrame`/`ModelDatumReferenceFrame`; surface texture.

## API contracts (interfaces / enums / collections)

- `Centerline(s)`,`Centermark(s)`,`FeatureControlFrame(s)`,`DatumReferenceFrame`,`SurfaceTextureSymbol(s)`,`AnnotationPlane(s)`

## Acceptance criteria

- A feature control frame references datums and stays attached to its feature.

## Depends on

_See feature dependencies._
