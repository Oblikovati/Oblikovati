---
milestone: M21
feature: F04
name: Composite & Parametric Entities
status: done
---

# M21 · F04 — Composite & Parametric Entities

The compound entities Inventor builds from primitives + auto-constraints: rectangles
(all variants), slots, polygons, sketch fillet/chamfer corner blends, fill regions, and
text. Each is a *builder* that composes existing lines/arcs and adds the geometric
constraints that hold the shape parametric.

## In scope

- Rectangle: two-point (exists), three-point, center.
- Slots: arc-by-center, arc-by-3-point, straight center-to-center / overall / slot-center.
- Polygon: inscribed / circumscribed, N sides.
- Sketch fillet & chamfer: trim two adjacent lines, insert arc/line, add tangent/coincident.
- `SketchFillRegion`; `TextBox` (sketch text).

## Out of scope

- Project/offset/image (F05); patterns (F10).

## Key API contracts delivered

- `SketchEntityKind`: `rectangle`, `slot`, `polygon`, `fillet`, `chamfer`, `fillRegion`, `text`
- `client.Sketch.{AddRectangle*,AddSlot*,AddPolygon,AddFillet,AddChamfer,AddText,AddFillRegion}`

## Depends on

F02 (primitives), F06 (auto-constraints).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-205](PBI-205-rectangles-slots-polygons.md) | Rectangles, slots, polygons |
| [PBI-206](PBI-206-sketch-fillet-chamfer.md) | Sketch fillet & chamfer |
| [PBI-207](PBI-207-fill-region-text.md) | Fill regions & text |
