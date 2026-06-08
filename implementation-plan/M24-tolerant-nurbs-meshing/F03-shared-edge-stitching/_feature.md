---
milestone: M24
feature: F03
name: Tolerant shared-edge stitching
status: planned
---

# M24 · F03 — Tolerant shared-edge stitching

Make adjacent freeform faces meet without gaps, folds, or double-counting. After F02 each face
meshes smooth on its **own** surface, but its on-surface boundary differs from the neighbour's by
the ~mm edge/surface tolerance — so the assembled body has seams. The classic fix (OpenCASCADE
`BRepMesh`) is to mesh each **edge once** as a single 3D polyline and have **both** adjacent faces
bind to those exact shared points; each face then meshes its interior on its surface and stitches
to the shared edge points. The body stays watertight despite no face's surface passing exactly
through the edge.

## In scope

- A per-edge shared discretization (one polyline per topological edge) cached and reused by both
  faces of the edge — `discretizeEdge` already yields a shared polyline; F03 ensures both faces'
  meshes **use those exact points** as boundary vertices (not each face's own on-surface pcurve
  points).
- A stitch band between the shared edge polyline and the face's on-surface interior nodes (the
  pcurve drives connectivity; the boundary vertices are the shared edge points).
- A watertightness check: no free edge / gap on the assembled body beyond the model tolerance.

## Out of scope

- Healing the underlying B-rep (moving edges onto surfaces) — a model-level effort.
- Non-manifold / degenerate edge handling beyond what import already produces.

## Key API contracts delivered

- (internal) shared-edge polyline cache; face mesher binding boundary to shared points.

## Depends on

F01, F02, `kernel/ops/edge_discretize.go` (`discretizeEdge`/`loopBoundary`), `kernel/topo`.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-317](PBI-317-shared-edge-polyline-binding.md) | Bind face boundary to the shared edge polyline (no gaps) |
| [PBI-318](PBI-318-watertight-assembled-body.md) | Watertight assembled body across the edge/surface tolerance |
