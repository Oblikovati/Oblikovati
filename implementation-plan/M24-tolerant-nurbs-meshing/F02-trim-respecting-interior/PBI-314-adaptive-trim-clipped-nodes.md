---
milestone: M24
feature: F02
pbi: PBI-314
title: Deflection-adaptive, strictly trim-clipped interior nodes
status: planned
estimate: L
---

# PBI-314 — Deflection-adaptive, strictly trim-clipped interior nodes

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F02 Trim-respecting adaptive interior

## Goal

Generate interior `(u,v)` nodes for a NURBS face whose density follows the surface curvature and
which lie **strictly inside the trim** — no spill into holes or past the boundary (the cause of
the +20–70% over-enclosure).

## Scope / work

- A node generator over the pcurve `(u,v)` bbox: step from `ops.Quality` (chord/angle tolerance)
  scaled by local curvature, so flat regions are coarse and curved regions dense
  (OpenCASCADE range-splitter style). Staggered rows for triangle quality.
- **Strict trim clipping**: keep a node only if `insideTrim(outerPcurve, holePcurves, node)` with
  a margin ≥ a fraction of the local step, so no node sits on or just outside the boundary or
  inside a hole. Use the smooth F01 pcurves (reliable point-in-polygon), not the jittery raw
  `(u,v)`.
- Pure function (nodes from pcurves + quality); unit-testable without a full face.

## API contracts (interfaces / enums / collections)

- (internal) `ops` interior-node generator `(outerPcurve, holePcurves, surface, quality) → []uv`.

## Acceptance criteria

- **No spill**: on a NURBS patch with a hole (synthetic), every generated node passes an
  independent point-in-trim test; **zero** nodes land inside the hole or outside the outer loop
  (committed test).
- **Adaptive density**: a high-curvature patch gets more nodes than a near-flat one of equal
  `(u,v)` extent at the same `Quality` (asserted by node-count ratio).
- Determinism: same inputs → same nodes.
- `go test ./kernel/ops/...` green; lint clean.

## Depends on

F01 (pcurves for the boundary + point-in-trim).
