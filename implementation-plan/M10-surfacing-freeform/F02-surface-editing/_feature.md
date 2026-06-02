---
milestone: M10
feature: F02
name: Surface Editing
status: planned
---

# M10 · F02 — Surface Editing

Features that edit existing surfaces: trimming/extending to boundaries, offsetting faces, extracting mid-surfaces for analysis, and thickening surfaces into solids.

## In scope

- Trim/Extend surfaces.
- FaceOffset/Offset surfaces.
- MidSurface extraction.
- Thicken (surface→solid).

## Out of scope

_None._

## Key API contracts delivered

- `TrimFeature(s)`,`ExtendFeature(s)`,`FaceOffsetFeature(s)`,`MidSurfaceFeature(s)`,`MidSurfaceThickness(es)`,`ThickenFeature(s)`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-111](PBI-111-trim-extend.md) | Trim & extend surfaces |
| [PBI-112](PBI-112-midsurface.md) | Mid-surface & offset |
