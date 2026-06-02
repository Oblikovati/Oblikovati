---
milestone: M10
feature: F01
pbi: PBI-110
title: Sculpt & knit/stitch
status: planned
estimate: M
---

# PBI-110 — Sculpt & knit/stitch

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F01 Surface Creation

## Goal

Implement sculpt (define a solid/surface region from bounding surfaces) and knit/stitch (combine surfaces into a quilt or solid).

## Scope / work

- `SculptFeature` region from surfaces.
- `StitchFeature` tolerance & solid-if-closed.

## API contracts (interfaces / enums / collections)

- `SculptFeature(s)`,`StitchFeature(s)`

## Acceptance criteria

- Stitching closed surfaces yields a solid; sculpt fills a bounded volume.

## Depends on

_See feature dependencies._
