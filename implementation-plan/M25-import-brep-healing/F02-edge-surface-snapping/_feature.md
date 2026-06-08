---
milestone: M25
feature: F02
name: Edge/surface tolerance snapping
status: planned
---

# M25 · F02 — Edge/surface tolerance snapping

Close the ~mm gap between imported edge curves and their faces' surfaces, and merge near-coincident
vertices/edges, so faces genuinely share their boundaries. With F01's pcurves, an edge's "true"
position on a surface is `PointAt(pcurve)`; F02 reconciles the (possibly differing) representations
across the two faces of an edge into one shared curve within the model tolerance.

## In scope

- Snap each edge's 3D curve onto a representative that lies on both adjacent surfaces within
  tolerance (or record the residual as the edge tolerance, OCCT-style), so the boundary the two
  faces mesh is the same.
- Merge near-coincident vertices and degenerate/duplicate edges within the model tolerance.

## Out of scope

- Sewing the faces into shells (F03 — F02 reconciles geometry; F03 builds the shared topology).
- Pcurve computation (F01).

## Key API contracts delivered

- (internal) edge/vertex snapping + merge in the heal path; per-edge tolerance.

## Depends on

F01 (pcurves), `kernel/topo`, the model tolerance from M02/M07.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-324](PBI-324-edge-onto-surface.md) | Snap edge curves onto their faces' surfaces within tolerance |
| [PBI-325](PBI-325-vertex-edge-merge.md) | Merge near-coincident vertices + degenerate edges |
