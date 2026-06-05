---
milestone: M20
feature: F01
pbi: PBI-199
title: Boolean robustness — partial penetration & concave-wall crossing
status: planned
estimate: XL
---

# PBI-199 — Boolean robustness: partial penetration & concave-wall crossing

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F01 Intersecting Booleans

## Goal

Make the planar B-rep boolean (`kernel/brep`) resolve **partial penetrations** and **concave
faceted-wall crossings**, not just clean through-overlaps. Today such a union/cut produces a
**non-manifold or inverted-normal** body because the 2D arrangement only cuts a face when the
imprint segments close into a loop *on that face* — a tool that pokes part-way in, or crosses a
re-entrant faceted wall, leaves dangling segments and the overlap is never removed. See
PBI-171 "Findings 2026-06-05" for the full diagnosis.

## Root cause (instrumented)

- `imprint` is computed per **face pair** and clipped to each face's extent, so a tool's
  footprint on a target face arrives as a **soup of open segments**; when the footprint loop
  closes only *across* a face boundary (multi-facet crossing) or ends *inside* the face (partial
  penetration), the segments dangle.
- `Arrange` → `planarize` → `traceCycles` keeps only **closed cycles**; dangling edges are
  dropped, so `splitFace` returns the face whole.
- Evidence: a fan blade JOIN reported `imprintSegs=86`, `split=75`, `dropped=13`, `kept=62` =
  inputs (no net split) → blade welded as a coincident shell (edges used by 3–4 faces).

## Scope / work

- **Assemble the imprint into closed loops** spanning multiple faces: chain the per-face-pair
  segments end-to-end (welding shared endpoints), closing each loop against the target's own
  face-boundary edges (insert **T-vertices** where a loop crosses a boundary).
- **Split with dangling/T-vertex support** in the arrangement so a partial-penetration footprint
  (open polyline closed by the face boundary) produces inside/outside regions.
- Propagate split vertices to the neighbour face sharing each boundary edge (the stitch already
  does this for through-cuts via `splitRingTJunctions`; extend to the partial case).
- Harden the **coincident / near-tangent** face handling for faceted curved walls (the
  "coplanar follow-up" in `boolean_coplanar.go`).
- Keep the BSP-CSG fallback as the safety net for anything still unhandled.

## Acceptance criteria

- Unskip and pass `oblikovati-mcp-bridge/bridge/e2e_fan_validate_test.go`:
  `TestBladeJoinBooleanIsTheDefect` (minimal: valid body + valid blade crossing a concave bore
  wall → JOIN is a valid manifold solid) and `TestFanBodyStaysManifold` (full fan valid after
  every feature, incl. the 7-blade circular pattern).
- A bar that pokes part-way into a face (does not exit the opposite side) → Cut/Join is a valid
  manifold solid with the correct volume.
- A blade that crosses a concave faceted cylindrical wall → valid manifold, consistent normals.
- No regression on the existing planar-boolean tests (`kernel/brep`, `kernel/ops`).

## Do NOT (recorded dead-end)

- A post-stitch orientation-repair pass (flood-fill + flip the minority) does **not** fix this —
  the mis-orientation is a symptom of the un-cut overlap, not a re-orientable manifold. Fix the
  imprint/arrangement, not the output.

## Depends on

PBI-171 (the planar pipeline). Related: the warped-quad precondition fix landed in PBI-174.

## Notes

This is the genuinely large piece behind the "deformed mesh / inverted normals" a user hit with
lofted-blade unions. The warped-quad triangulation (PBI-174) removed the *gross* deformity (the
twisted blade itself was non-planar); this PBI removes the *localised* deformity where the tool
crosses a concave faceted wall.
