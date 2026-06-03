---
milestone: M21
feature: F02
name: Curve & Point Entities
status: planned
---

# M21 · F02 — Curve & Point Entities

Bring the core analytic sketch entities — lines, circles, arcs, points — to full
Inventor creation parity through `/api` and interactive tools, building on the F01
`addEntity(Kind)` scaffold.

## In scope

- `addEntity` kinds: `line`, `point`, `circle` (center-radius, by-3-tangents),
  `arc` (center-start-end, three-point, tangent-to-curve).
- Construction-geometry flag on each.
- Ribbon Create-panel tools + e2e for each.

## Out of scope

- Conics/splines (F03); rectangles/slots/polygons (F04).

## Key API contracts delivered

- `types.SketchEntityKind` members for line/point/circle/arc + overload selectors
- `wire.AddSketchEntityArgs/Result`; `client.Sketch.{AddLine,AddCircle,AddArc,AddPoint}`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-202](PBI-202-curve-point-entities.md) | Lines/circles/arcs/points — API + tools + e2e |
