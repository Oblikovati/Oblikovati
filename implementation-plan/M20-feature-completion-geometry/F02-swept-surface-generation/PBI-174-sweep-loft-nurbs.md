---
milestone: M20
feature: F02
pbi: PBI-174
title: Sweep & loft NURBS bodies
status: planned
estimate: L
---

# PBI-174 — Sweep & loft NURBS bodies

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F02 Swept-Surface Generation

## Goal

Sweep a profile along a path and loft between sections into real B-rep solids/surfaces.

## Scope / work

Profile-along-path sweep (translational + path frames); ruled/NURBS loft between 2+ sections; closed/open; rib as a bounded thin sweep.

## API contracts (interfaces / enums / collections)

- `SweepFeature`/`LoftFeature`/`RibFeature` real geometry; `ops.Sweep`/`ops.Loft`.

## Acceptance criteria

- A circle swept along an L-path is a validated solid
- a loft between two squares is manifold
- rib fills to the next face
- recompute on input change.

## Depends on

M07, M08

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.

## Fix 2026-06-05 — twisted sides must be planar-faceted (or booleans break)

`sweptSolid` connects consecutive sections with **quad** side faces. When the sections are
**twisted/rotated** relative to each other (a lofted fan blade, a coil, a swept profile that
rotates), that quad is a **warped ruled surface — non-planar**. Emitted as a single face, the
cage→B-rep converter fits it to an **approximating plane**, which silently breaks the planar
boolean: it imprints against the approximate plane, so the intersection segments land **offset
from the body's true edges** (offset ≈ 0.875·twist — even a **0.001 rad** twist was enough), the
imprint loop fails to close, the face is not split, and a partial-penetration union goes
**non-manifold (deformed mesh / inverted normals)**.

**Fixed** in `model/feature/swept.go`: `sideQuad`/`quadPlanar` split a side quad into two
**exact-planar triangles** when its four corners are not coplanar (tol 1e-8, below the
arrangement's 1e-7 weld grid); planar (untwisted) quads stay single faces for a low face count.
This restores the **planar-faceted invariant** every consumer of `sweptSolid`
(revolve/sweep/loft/coil/rib) relies on. Regression:
`model/feature/blade_twist_min_test.go::TestTwistedLoftUnionStaysManifold` (a twisted blade
unioned into a hub is valid/manifold at every twist 0 → 0.3 rad). This is the *precondition* for
the boolean robustness work; the remaining concave-wall-crossing deformity is **PBI-199**.
