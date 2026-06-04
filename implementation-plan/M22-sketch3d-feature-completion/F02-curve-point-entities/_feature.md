---
milestone: M22
feature: F02
name: Curve & Point Entities (3D)
status: done (model+API; UI in F12)
---

# M22 · F02 — Curve & Point Entities (3D)

The base 3D geometry: `SketchPoint3D`, `SketchLine3D`, `SketchCircle3D`,
`SketchArc3D` (incl. bends, all overloads). Each is a thin model wrapper over the
existing `kernel/geom` 3D primitives (`Line`, `Circle`, `Arc3d`), constrainable via the
3D solver (3 vars per point). Full `sketch3d.addEntity` cases + interactive tools.

## In scope

- Entities: point, line (two-point), circle (center-axis-radius, three-point), arc
  (center-start-end, three-point), bend (fillet between two 3D lines).
- `sketch3d.addEntity` `Kind`/`Variant` cases + typed `client` helpers.
- 3D placement tools (pick points in space / on grid).

## Key contracts

- `types.Sketch3DEntity{Point,Line,Circle,Arc}`
- `wire.AddSketch3DEntityArgs` / `AddSketch3DEntityResult`
- `client.Sketch3D.{AddPoint,AddLine,AddCircle*,AddArc*}`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-232](PBI-232-curve-point-entities.md) | 3D point/line/circle/arc entities + addEntity + tools |
