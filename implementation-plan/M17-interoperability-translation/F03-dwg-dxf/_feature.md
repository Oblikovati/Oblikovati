---
milestone: M17
feature: F03
name: AutoCAD DWG/DXF
status: planned
---

# M17 · F03 — AutoCAD DWG/DXF

AutoCAD interoperability: importing/exporting DWG/DXF entities (lines/arcs/polylines/splines/blocks), used for 2D import into sketches and drawings, flat-pattern export (M13), and drawing export (M14).

## In scope

- DWG/DXF entity read/write.
- Block definitions/references.
- Layer/line/unit mapping; 2D into sketches/drawings.

## Out of scope

_None._

## Key API contracts delivered

- `DWGEntity`,`DWGLine`,`DWGArc`,`DWGPolyline(2D/3D)`,`DWGSpline`,`DWGBlockDefinition`,`DWGBlockReference`,`AutoCADBlock(s)`,`AutoCADBlockDefinition(s)`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-161](PBI-161-dwg-dxf.md) | DWG/DXF import/export & blocks |
