---
milestone: M10
feature: F03
name: Freeform Modeling
status: planned
---

# M10 · F03 — Freeform Modeling

Sub-division-surface free-form modeling: free-form bodies (box/plane/sphere/cylinder/quad-ball primitives) and their editable faces/edges/vertices/cages with smooth/crease editing.

## In scope

- FreeformFeature/AliasFreeform bodies.
- Freeform faces/edges/vertices/bodies.
- Edit ops (move/scale/crease/subdivide/bridge).

## Out of scope

_None._

## Key API contracts delivered

- `FreeformFeature(s)`,`AliasFreeformFeature(s)`,`FreeformBody`,`FreeformFace`,`FreeformEdge`,`FreeformVertex`,`FreeformBodies`

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-113](PBI-113-freeform-bodies.md) | Freeform (sub-D) bodies & primitives |
| [PBI-114](PBI-114-freeform-edit.md) | Freeform edit operations |
