---
milestone: M21
feature: F05
name: Reference & Image Entities
status: done
---

# M21 · F05 — Reference & Image Entities

Sketch geometry derived from *other* geometry: project/include of model edges &
work geometry, offset of sketch chains, and raster sketch images. These create
reference entities that track their source through recompute (reference keys).

## In scope

- Project geometry / Include (`AddByProjectingEntity`, project-to-plane, project cut edges).
- Offset (`OffsetSketchEntitiesUsingDistance` / `OffsetSketchEntitiesUsingPoint`).
- `SketchImage` (raster placed on the sketch plane).

## Out of scope

- DWG geometry projection (excluded from this milestone).

## Key API contracts delivered

- `MethodSketchProject`, `MethodSketchOffset`; `addEntity` kind `image`
- `client.Sketch.{Project,Include,Offset,AddImage}`

## Depends on

F02; M03 reference keys; M08 model edges (project source).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-208](PBI-208-project-include.md) | Project geometry & include |
| [PBI-209](PBI-209-offset.md) | Offset sketch entities |
| [PBI-210](PBI-210-sketch-image.md) | Sketch image |
