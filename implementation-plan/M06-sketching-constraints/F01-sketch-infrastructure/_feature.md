---
milestone: M06
feature: F01
name: Sketch Infrastructure
status: planned
---

# M06 · F01 — Sketch Infrastructure

The sketch container objects, their collections, and the plane/face hosting that locates a 2D sketch in 3D space, including coordinate mapping between sketch and model space.

## In scope

- `PlanarSketch`/`Sketch3D`/`DrawingSketch` + collections.
- Sketch plane/face hosting; coordinate mapping.
- Sketch edit state; visibility.

## Out of scope

_None._

## Key API contracts delivered

- `PlanarSketch`,`Sketches`,`Sketch3D`,`Sketches3D`,`DrawingSketch`,`DrawingSketches`
- `SketchEvents`

## Depends on

M01,M05.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-066](PBI-066-sketch-objects.md) | PlanarSketch/Sketch3D containers & collections |
| [PBI-067](PBI-067-project-geometry.md) | Project geometry & reference into sketch |
