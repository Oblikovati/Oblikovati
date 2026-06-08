---
milestone: M24
feature: F02
pbi: PBI-315
title: Curvature-aware triangulation with fold detection + repair
status: planned
estimate: L
---

# PBI-315 — Curvature-aware triangulation with fold detection + repair

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F02 Trim-respecting adaptive interior

## Goal

Triangulate (pcurve boundary + interior nodes) so the lifted 3D mesh does **not fold** — no
adjacent triangles whose surface normals oppose — even where the `(u,v)→3D` map is strongly
non-conformal.

## Scope / work

- CDT the point set in `(u,v)` (reuse `cdt.go`), lift to 3D via `PointAt`, wind by vertex normals
  (`windingOpposesNormals`).
- **Fold detector**: an interior edge whose two triangles' 3D geometric normals oppose
  (dot < threshold·|n||n|) is a fold. After triangulation, find folds and repair: edge-flip the
  offending diagonal, or insert a node at the fold and re-triangulate locally, until no fold
  remains or a bounded number of passes elapse.
- If folds persist (a genuinely degenerate `(u,v)` region), increase local node density there
  (driving the F02 generator) rather than leaving a fold.

## API contracts (interfaces / enums / collections)

- (internal) `ops` fold detector + repair pass over a `Mesh`.

## Acceptance criteria

- On the constructed strongly-curved patch that folds today (body-0-like), the meshed result has
  **0 fold-edges** (committed fold-detector test).
- Triangle winding stays consistent (every triangle agrees with its vertex normals).
- Area conservation: the meshed surface area is within tolerance of a densely-sampled reference
  area for the patch (no overlap from a bad repair).
- `go test ./kernel/ops/...` green; lint clean.

## Depends on

PBI-314 (interior nodes), `kernel/ops/cdt.go`.
