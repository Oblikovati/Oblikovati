---
milestone: M17
feature: F03
pbi: PBI-161
title: DWG/DXF import/export & blocks
status: planned
estimate: L
---

# PBI-161 — DWG/DXF import/export & blocks

**Milestone:** M17 Interoperability & Translation  ·  **Feature:** F03 AutoCAD DWG/DXF

## Goal

Implement DWG/DXF read/write of entities and blocks with layer/line/unit mapping, feeding 2D sketch/drawing import and flat/drawing export.

## Scope / work

- Entity read/write (line/arc/ellipse/polyline/spline).
- Block defs/refs; layers.
- Map to/from sketch & drawing geometry.

## API contracts (interfaces / enums / collections)

- `DWGEntity`,`DWGLine/Arc/Polyline/Spline`,`DWGBlockDefinition`,`DWGBlockReference`,`AutoCADBlock(s)`

## Acceptance criteria

- A DWG imports entities/blocks onto layers; a drawing/flat exports to DWG/DXF.

## Depends on

_See feature dependencies._
