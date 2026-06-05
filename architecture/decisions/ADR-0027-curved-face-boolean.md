# ADR-0027 — Curved-face B-rep boolean (K1b)

**Status:** accepted, in progress (slice 1 landed 2026-06-04)
**Context:** REPORT.md §6 (G-09), DEFERRALS.md, PARTDOC-PLAN.md Phase 4.

## Problem

`kernel/brep` does an exact boolean only for **planar** faces (`facesOf` returns
`ok=false` on any curved face; `ops.Boolean` then falls back to triangle-soup **BSP CSG**).
Consequences:

- A drilled **Hole** / **Boss** is a faceted 32-gon, not a clean cylinder face.
- **Fillet** results and any curved boolean are tessellated soup: high face count, **not
  exact under chaining**, and their faces carry no stable lineage (reference keys don't
  survive — even K1a doesn't help, since K1a lives in the planar path).

K1b makes analytic curved faces (cylinder, cone, sphere, torus) **first-class** in the
exact B-rep boolean, so curved results are single faces with surviving reference keys and
are exact under chaining.

## Decision

Generalize the planar boolean's pipeline rather than write a parallel one. The four stages
stay (imprint → split → classify → stitch); each is lifted from "plane + 3D point loops" to
"any analytic surface + curved trim loops":

1. **Face model.** Replace `planarFace{plane, normal, loops}` with a face carrying its
   `geom.Surface` + boundary loops as **`geom.Curve3`** edges (lines *and* circles/arcs),
   keeping the source lineage (K1a threading already in place).
2. **Imprint = exact surface∩surface curves.** Use **`geom.IntersectSurfacesAnalytic`**
   (slice 1, done) for the analytic pairs it solves in closed form (plane∩plane → line,
   plane∩cylinder ⟂ axis → circle, plane∩sphere → circle); fall back to the numeric tracer
   `IntersectSurfaceSurface` (faceted) for the rest (oblique ellipse, cylinder∩cylinder,
   torus). Exact curves are what keep a hole a single clean face.
3. **Split in parameter space.** Map a face's boundary + imprint curves into the surface's
   (u,v) domain via `Surface.ParamAt`, run the existing 2D arrangement there, lift sub-face
   loops back to 3D. (Cylinders/cones unroll to a periodic u; spheres/tori are doubly
   periodic — handle the seam.)
4. **Classify + stitch.** Classification (inside/outside via ray cast; coplanar table) is
   unchanged. Stitch welds curved edges (shared circle/arc per undirected vertex pair) and
   emits faces carrying the source surface + lineage (reuse K1a's `faceLineage`).

## Slicing (incremental, each shippable + tested)

- **Slice 1 — exact analytic intersection curves.** `geom.IntersectSurfacesAnalytic`:
  plane∩plane → line, plane∩cylinder → circle (⟂) / ellipse (oblique), plane∩sphere →
  circle, plane∩cone → circle (⟂); line-pair / curved-curved pairs defer to the numeric
  tracer. + tests. **DONE 2026-06-04.**
- **Slice 2 — curved face model + edge representation** in `kernel/brep` (surface + curve3
  loops); planar path keeps working through it. **DONE 2026-06-04:** `SolidCylinder` builds
  the first analytic curved solid (true cylinder side face + closed-circle edges + periodic
  seam); it validates (manifold, watertight) AND tessellates with correct area/volume.
- **Slice 2b — periodic-face tessellation. DONE 2026-06-04.** The trimmed-face tessellator
  (PBI-32) only handled partial (fillet) bands; a full 2π-periodic side fell back to the
  surface's whole (unbounded) UV domain → wrong area/volume. Fixed with `ops.periodicBandGrid`:
  a full-seam-wrap loop is gridded over the entire period (reusing the boundary's own circle-
  edge samples, so it stays watertight with the caps) and the boundary's bounded range in the
  other direction. Tested at the ops level (face area = 2π·r·h) and brep level (volume = π·r²·h).
- **Slice 3 — plane∩cylinder boolean. DONE 2026-06-04.** `brep.CutCylindricalHole` drills a
  clean through-hole in a planar slab: the two pierced faces gain a circular hole, a single
  true cylinder face forms the wall, and the result is a watertight solid with the right
  volume (slab − inscribed bore). Needed a new primitive — **face sense**: `topo.Face.Reversed`
  + `Builder.AddReversedFace`, honored by `ops.TessellateFace` (negates normals + flips
  winding). A cut wall's surface normal points into the removed material, so its face is
  reversed; planar cuts fake this by building a flipped plane, but a cylinder can't be flipped,
  so the sense flag is the general mechanism (the planar boolean can adopt it later). Copied
  and pierced faces keep their source lineage, so **every original face's reference key survives
  the curved cut** (extends K1a to curved booleans). Partial/blind holes still error (later
  slices). Tested: validity/watertight/face-count, volume, key survival, oversize rejection.
- **Slice 4 — sphere/cone**, then **cylinder∩cylinder** (numeric-curve imprint). Also
  generalize beyond the through-hole specialization to the arrangement-based curved boolean.
- **Slice 5 — curved edge/vertex key survival**; retire the BSP-CSG fallback for the
  analytic cases; route Hole/Fillet through it. **Partly done 2026-06-04:** the **Hole
  feature** routes a Through-All hole on a planar slab through `CutCylindricalHole` (true
  cylinder wall, full DoD: model + Hole UI Through-All option + head checkbox + persistence +
  e2e), **blind holes** through `CutBlindCylindricalHole` (cylinder wall + flat bottom disk),
  and **counterbores** through `CutCounterboreHole` (recess wall + annular shoulder + bore wall,
  built in one assembly — the exact cutters can't be chained because they need an all-planar
  input), **countersinks** through `CutCountersinkHole` (a true CONE frustum recess sharing
  a transition circle with the bore wall; the periodic-band tessellator handles the cone's
  varying radius), and **conical drill points** through `CutBlindConicalHole` (cone tip closing
  the bore; needed `ops.coneApexFan`, a fan to the cone's apex pole for a cone whose only
  boundary is one rim circle). The **Hole feature now has full Inventor parity** (drilled
  flat/point, through, blind, counterbore, countersink) with real curved geometry. Remaining for
  K1b: the general (non-slab) curved arrangement boolean, curved edge/vertex key survival, Fillet.

Until a slice lands, the affected case keeps using the BSP-CSG fallback (correct, just
faceted) — no regression.

## Consequences

- Hole/Boss/Fillet become clean, low-face-count, chain-exact, with stable face keys.
- Larger than K1a; spans several PBIs. The BSP-CSG fallback stays as the safety net for
  unhandled pairs.
