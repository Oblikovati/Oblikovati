---
milestone: M25
feature: F01
pbi: PBI-323
title: Reconstruct + attach per-edge pcurves bounding each trim
status: done
estimate: L
---

# PBI-323 — Reconstruct + attach per-edge pcurves bounding each trim

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F01 Robust projection & pcurve reconstruction

## Goal

Give every imported edge an accurate pcurve on each adjacent face, attached to the topology, so the
trim region is exact in `(u,v)` — the data SolidWorks omits and the mesher needs.

## Scope / work

- For each face, project its boundary edges onto its surface (`ProjectCurveToSurface`) to build the
  outer + hole pcurve loops; pick the `(u,v)` sheet (for periodic/seam surfaces) whose enclosed
  region is the actual trim (e.g. by testing an interior sample's on-surface residual).
- Attach the pcurve to the topology per edge-use (a `CoEdge.Pcurve` on the face), so
  `kernel/ops` tessellation reads the exact `(u,v)` instead of re-deriving it.
- Validate each reconstructed pcurve: continuous, non-self-intersecting, and its interior `(u,v)`
  samples have small on-surface residual AND lie within the face's 3D bounds (the trim, not a
  folded elsewhere region).

## API contracts (interfaces / enums / collections)

- `topo` `CoEdge`/`EdgeUse` pcurve field + accessor; pcurve population in the STEP import/heal path.

## Acceptance criteria

- EDF.STEP: every freeform face gets outer + hole pcurves; an interior `(u,v)` sample inside the
  outer pcurve (outside holes) has on-surface residual within tolerance AND maps within the face's
  3D bbox — i.e. the pcurve interior IS the trim (the M24 over-enclosure root, now fixed by data).
- A committed check: the reconstructed pcurve round-trips the boundary points (`PointAt` ≈ edge).
- OCC oracle green (pcurve attachment does not change existing tessellation that already worked).

## Depends on

PBI-322 (projection), `kernel/topo`, the STEP import path.
