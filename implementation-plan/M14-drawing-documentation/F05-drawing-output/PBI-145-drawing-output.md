---
milestone: M14
feature: F05
pbi: PBI-145
title: Print/plot & PDF/DWG/DXF export
status: planned
estimate: M
---

# PBI-145 — Print/plot & PDF/DWG/DXF export

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F05 Drawing Output

## Goal

Implement drawing printing/plotting and export to PDF and DWG/DXF with style/layer mapping.

## Scope / work

- `DrawingPrintManager` (scale/area/range).
- PDF export.
- DWG/DXF export with layer/line mapping (via M17).

## API contracts (interfaces / enums / collections)

- `DrawingPrintManager`, PDF/DWG/DXF export

## Acceptance criteria

- A drawing prints to scale and exports to PDF and DWG faithfully.

## Depends on

_See feature dependencies._
