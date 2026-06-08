---
milestone: M25
feature: F01
name: Robust projection & pcurve reconstruction
status: planned
---

# M25 · F01 — Robust projection & pcurve reconstruction

The core of healing: give every imported edge an accurate **pcurve** (its `(u,v)` curve on each
adjacent face's surface), so the trim region is known exactly in parameter space. Built on a
formalised surface point-inversion projector — M24 verified the existing `ParamNear` is sound
(matches brute-force closest-point), so this is about *using* it to reconstruct pcurves, with the
seam / periodicity / correct-sheet handling that makes the pcurve interior stay on the trim.

## In scope

- A formal `ProjectPointToSurface` (multi-seed + branch/periodicity-aware) over the verified
  `ParamNear`, and a `ProjectCurveToSurface` that marches an edge curve onto a surface (the
  `ops.marchUV` technique), producing a continuous, non-self-intersecting pcurve.
- **Pcurve reconstruction**: for each edge, build its pcurve on each adjacent face and attach it to
  the topology (an `EdgeUse`/`CoEdge` pcurve), so tessellation/operations consume the exact `(u,v)`.
- Seam/periodic handling so the pcurve picks the `(u,v)` sheet whose interior is the actual trim.

## Out of scope

- Snapping the 3D edge onto the surface (F02) — F01 computes the pcurve; F02 moves the geometry.
- Sewing (F03), orientation (F04).

## Key API contracts delivered

- `geom`/`kernel` `ProjectPointToSurface`, `ProjectCurveToSurface`; pcurve storage on the topology.

## Depends on

M24 `ParamNear` (verified projector) + `ops.marchUV`; `kernel/topo` (edge/face/coedge).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-322](PBI-322-projection-api.md) | Robust point/curve-to-surface projection API |
| [PBI-323](PBI-323-pcurve-reconstruction.md) | Reconstruct + attach per-edge pcurves bounding each trim |
