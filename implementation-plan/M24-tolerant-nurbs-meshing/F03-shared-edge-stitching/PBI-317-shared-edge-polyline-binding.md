---
milestone: M24
feature: F03
pbi: PBI-317
title: Bind face boundary to the shared edge polyline (no gaps)
status: planned
estimate: L
---

# PBI-317 — Bind face boundary to the shared edge polyline (no gaps)

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F03 Tolerant shared-edge stitching

## Goal

Both faces of an edge use the SAME 3D points (the shared `discretizeEdge` polyline) as their
boundary vertices, while the face interior meshes on its own surface — so the two face meshes meet
exactly at the edge.

## Scope / work

- In the NURBS face mesher (PBI-316), the boundary loop's 3D vertices are the **shared edge
  polyline points** (`loopBoundary` / `discretizeEdge`, already shared between neighbours), not
  `PointAt` of the pcurve. The pcurve `(u,v)` still drives the 2D CDT connectivity + interior
  placement; only the boundary VERTICES bind to the shared points.
- The first interior ring sits a controlled step inside the boundary so the stitch band triangles
  (shared edge ↔ first interior ring) are well-shaped and do not fold (they span the ~mm
  edge/surface offset uniformly).
- Keep the on-surface interior; the boundary lip is the tolerance, distributed evenly.

## API contracts (interfaces / enums / collections)

- (internal) NURBS face mesher boundary = shared edge points; interior = on-surface nodes.

## Acceptance criteria

- Two adjacent freeform faces sharing an edge produce boundary vertices that **coincide exactly**
  (same shared points) — a committed test on a synthetic two-face patch.
- No fold introduced in the stitch band (fold detector = 0).
- EDF volume stays within oracle tolerance; OCC oracle green.
- `go test ./kernel/...` green; lint clean.

## Depends on

PBI-316, `kernel/ops/edge_discretize.go`.
