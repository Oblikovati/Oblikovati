---
milestone: M10
feature: F01
name: Surface Creation
status: planned
---

# M10 · F01 — Surface Creation

Features that create surface bodies from curves/edges/surfaces: boundary patches over closed loops, ruled surfaces, sculpting between surfaces, and knitting/stitching surfaces into quilts.

## In scope

- BoundaryPatch (loops, tangency).
- RuledSurface; Sculpt.
- Knit/Stitch surfaces.

## Out of scope

_None._

## Key API contracts delivered

- `BoundaryPatchFeature(s)`,`BoundaryPatchDefinition`,`BoundaryPatchLoop(s)`,`RuledSurfaceFeature(s)`,`SculptFeature(s)`,`StitchFeature(s)`/`KnitFeature(s)`

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-109](PBI-109-boundary-patch.md) | Boundary patch & ruled surface |
| [PBI-110](PBI-110-sculpt-stitch.md) | Sculpt & knit/stitch |
