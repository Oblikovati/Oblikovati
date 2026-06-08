---
milestone: M24
feature: F02
name: Trim-respecting adaptive interior
status: planned
---

# M24 · F02 — Trim-respecting adaptive interior

The heart of the milestone: generate **interior nodes that follow the surface curvature** so the
face stops folding, **without over-enclosing** the volume. Every naive interior-node attempt
inflated the EDF volume +20–70% because the `(u,v)` grid + CDT spilled past the real trim (into
holes, beyond the boundary on complex faces). F02 makes the interior strictly trim-respecting and
deflection-adaptive, and makes the triangulation curvature-aware so it does not fold in 3D — gated
at every step by the OCC volume oracle (the metric that exposed the inflation).

## In scope

- A **deflection-adaptive** `(u,v)` node spacing from `ops.Quality` (chord/angle tolerance) and
  the surface's local curvature — the OpenCASCADE range-splitter idea (dense where curved, coarse
  where flat).
- **Strict trim clipping**: an interior node survives only if it is inside the outer pcurve and
  outside every hole pcurve, by a robust point-in-polygon with a margin so nodes never sit on or
  just-outside the boundary.
- A **curvature-aware** triangulation of (pcurve boundary + interior) that does not fold: detect
  and repair fold edges (adjacent triangles whose 3D normals oppose) by local re-triangulation
  or node nudging, or reject nodes that would fold.
- The wired NURBS face mesher replacing `trimmedPatchMesh` for B-spline faces, **on-surface**
  (boundary + interior via the pcurve), gated by the oracle.

## Out of scope

- Cross-face consistency / gaps (F03 — shared-edge stitching).
- Analytic surfaces (their interior path already exists: `gridPatchMesh` for spheres,
  `structuredGridMesh` for cylinders/cones).

## Key API contracts delivered

- (internal) `ops` deflection-adaptive interior node generator; the NURBS face mesher.

## Depends on

F01 (pcurves), `kernel/ops/{cdt,refined_patch,tessellate_trim}.go`, the OCC oracle.

## Backlog items

| PBI | Title | Status |
|-----|-------|--------|
| [PBI-314](PBI-314-adaptive-trim-clipped-nodes.md) | Deflection-adaptive, strictly trim-clipped interior nodes | done (merged) |
| [PBI-315](PBI-315-curvature-aware-no-fold.md) | Curvature-aware triangulation with fold detection + repair | done (merged) |
| [PBI-316](PBI-316-wire-nurbs-face-mesher.md) | Wire the NURBS face mesher — **BLOCKED**: (u,v) interior is non-conformal on imported faces | superseded |
| [PBI-321](PBI-321-3d-space-refinement.md) | 3D-space interior refinement (replaces the (u,v) grid) | planned |

> **Finding (PBI-316):** the `(u,v)`-grid interior cannot refine imported NURBS — their
> parameterization is non-conformal (interior `(u,v)` maps ~15% of the face off the trim →
> +33% volume). F01 pcurve + PBI-314 density + PBI-315 fold-repair stay valid; only the interior
> *sampling* moves from `(u,v)` to **3D space** (PBI-321). PBI-314/315 are merged but not yet
> production-wired (their tests exercise them); PBI-321 wires the 3D-refined mesher.
