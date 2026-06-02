---
milestone: M10
feature: F02
pbi: PBI-112
title: Mid-surface & offset
status: planned
estimate: M
---

# PBI-112 — Mid-surface & offset

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F02 Surface Editing

## Goal

Implement mid-surface extraction (paired-face midsurfaces with per-pair thickness) and surface offset.

## Scope / work

- `MidSurfaceFeature` face pairing & `MidSurfaceThickness`.
- Offset/face-offset surfaces.

## API contracts (interfaces / enums / collections)

- `MidSurfaceFeature(s)`,`MidSurfaceThickness(es)`,`FaceOffsetFeature(s)`

## Acceptance criteria

- A thin solid yields a mid-surface with recorded thicknesses for FEA.

## Depends on

_See feature dependencies._
