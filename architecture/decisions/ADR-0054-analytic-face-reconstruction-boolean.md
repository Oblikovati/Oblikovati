# ADR-0054 — Analytic-face reconstruction from the mesh-arrangement boolean

**Status:** Accepted — building on `feat/analytic-boolean-reconstruction`. · **Scopes**
[Oblikovati#2153](https://github.com/Oblikovati/Oblikovati/issues/2153) (the arrangement emits faceted
bodies) and closes [Oblikovati#2167](https://github.com/Oblikovati/Oblikovati/issues/2167) (visible
seam between two cocylindrical extrude walls). · **Builds on**
[ADR-0052](ADR-0052-planar-boolean-corefined-seam-classification.md) (the exact mesh-arrangement
boolean core `kernel/meshbool`) and [ADR-0043](ADR-0043-generalized-provenance-naming.md) (edge/vertex
provenance). · **Touches:** `kernel/meshbool/*`, `kernel/ops/meshbool_body.go`,
`kernel/ops/meshbool_soup.go`, a new `kernel/ops/reconstruct_*.go`.

## Context

The ADR-0052 core (`kernel/meshbool`) is a complete, exact, watertight mesh-arrangement boolean: it
co-refines the two operands on exact predicates, classifies by exact ray-cast, and emits a closed
2-manifold triangle soup with exact volume. It is wired as a robustness **fallback** in
`ops.booleanGeneral`, never the default.

It cannot become the default because the OUT adapter (`soupToBody` → `meshbool.MergeFaces`) rebuilds
the body by fitting **one `geom.Plane` per coplanar facet region**. Every curved boolean output is
therefore faceted: a crossing-cylinder union becomes ~108 planar facets instead of 7 analytic faces.
Defaulting the mesh engine would faceted-ize every curved boolean, breaking:

- **fillet/chamfer on a boolean edge** (needs an analytic edge curve, not a facet chain),
- **exact mass properties** (a facet body under-reports curved volume),
- **STEP/native export** (surfaces, not facets),
- **chained booleans and TNP provenance** (a facet fan has no face identity),
- and it produces the visible **#2167 seam**: two cocylindrical walls faceted **independently** leave
  mismatched facet grids at their z-interface. Each wall is smooth; the boundary between two
  independently-faceted copies of the *same* cylinder is not.

## Key insight — provenance, not fitting

Co-refinement (`kernel/meshbool/mesh.go` `refineAgainst` → `RefineFace`) only ever **splits** an input
facet against the other operand's crossing segments. It never fuses two input facets into one output
triangle, and it never creates a triangle off an original surface. **Every output triangle descends
from exactly one input face of one operand, whose exact analytic `geom.Surface` is already known.**

So reconstruction must NOT fit surfaces from a facet point cloud (fragile, tolerance-bound, and unable
to recover the exact radius/axis). It must **carry the originating surface as an exact provenance tag**
through the arrangement and group the output by tag. The face's surface is then the *original* surface,
bit-for-bit — no fitting, no error. This also restores ADR-0043 face identity for free (a reconstructed
face knows it came from operand A's face 3).

## Decision

Thread an exact per-triangle provenance tag through `kernel/meshbool`, then reconstruct the body from
tag-grouped faces on their original analytic surfaces.

### Layer 1 — provenance tagging (foundation)

- `meshbool` gains a `TaggedSoup{Tris [][3]Point; Tags []int}` and `BooleanTagged(a, b TaggedSoup, op)`.
  A tag is a caller-defined surface id; `refineAgainst` copies the parent face's tag onto every child,
  `CoRefine`/`Boolean` propagate it through keep/coplanar/orient. The untagged `Boolean` stays for
  callers that do not care (it delegates with a single-tag soup).
- The IN adapter (`ops/meshbool_soup.go`) builds the tagged soup from `tessellateBodyFaces`, which
  already meshes **one `*Mesh` per face**: operand A's faces get ids `0..na-1`, operand B's `na..na+nb-1`,
  and a side table maps each id → `(operand, *topo.Face, geom.Surface)`.

### Layer 2 — tag-grouped analytic faces

- Group the result soup by tag; each group is a 2-manifold patch on one original surface. Trace its
  boundary loops (the existing `boundaryEdges`/`traceLoops`, keyed on tag not coplanarity) and build a
  `topo.Face` with the **exact original `geom.Surface`** + traced loops.
- Adjacent groups on the *same* surface (cocylindrical walls, #2167) merge into one face.

### Layer 3 — analytic edge curves

Each boundary loop is a polyline of arrangement vertices. Reconstruct analytic edges:

- a run of vertices lying on an **original operand edge** reuses that edge's exact `geom.Curve`
  (circle, line, arc, ...) — identified by provenance (both endpoints original, from that edge);
- a **new intersection run** (operand A's surface ∩ operand B's surface) becomes the true
  surface-surface intersection curve (Layer 4), interim a faithful spline through the arrangement
  vertices that lie on both surfaces.

### Layer 4 — exact surface-surface intersection (OCCT parity)

Per surface-pair family produce the exact `geom.Curve`: plane∩plane = line; plane∩cylinder = line-pair
/ ellipse; sphere∩plane = circle; cylinder∩cylinder, cone∩plane, torus∩plane, etc. General/untractable
pairs keep the faithful spline. This is what raises boolean-edge geometry to OCCT parity.

### Layer 5 — cutover

Shadow-validate the reconstructed body against the existing analytic boolean AND the OCCT oracle
(`boolean_occ_oracle_test.go`): strict-superset gate — valid, closed, exact volume/area, and analytic
face/edge counts matching the analytic engine. Flip `ops.booleanGeneral` to prefer the reconstructed
mesh engine only when green on the full `kernel/brep` + `kernel/ops` corpus. Flip-back is the valve.

## Consequences

- The mesh engine can finally become the **default** boolean (ADR-0052 Phase 2 unblocks): watertight by
  construction AND analytic-faced, so downstream fillet/mass-props/export keep working.
- **Provenance is exact, not fitted.** The reconstructed surface is the original surface; the only
  approximation is the trim boundary where a general SSI curve stays a spline (Layer 4 removes it for
  the tractable families).
- Corpus safety is the gating invariant, as in ADR-0052: no cutover until the reconstructed engine is a
  strict superset of the analytic engine on the full boolean corpus and the OCCT oracle.

## Status (findings from the first build)

Layers 1–2c and the (gated-off) recovery hook are landed and green. Reconstruction is
proven on gluing unions (a stepped coaxial shaft → two analytic walls, exact volume),
planar booleans (box union, exact volume), and a curved-bar ∪ tool. It is **not yet the
default** — `reconstructionCutover = false` — because two frontiers remain:

1. **Cocylindrical cap-on-wall (#2167) — root-caused, blocked on tessellation.** When two
   operands share the radius-R circle at a join plane (a cylinder's top-cap rim and a
   D-prism's bottom-cap arc are the SAME circle) but tessellate it INDEPENDENTLY — one
   reaching the true rim at R, the other inscribed below R — the thin ring between the two
   approximations is a **zero-volume, opposite-facing membrane**: kept because
   `coplanarPartner` cannot see it as coincident and `insideExact` reports it outside the
   *tessellated* other operand. The soup is 2-manifold and correct-volume (the membrane is
   edge-paired and volume-neutral), but it is degenerate, so BOTH `soupToBody` and the
   provenance reconstruction group it to a non-manifold B-rep. **Fix:** conform the two
   operands' tessellations of the shared cocylindrical circle — a canonical absolute-angle
   sampling of circle/arc edges so cocylindrical edges share vertices. This is a change to
   `kernel/ops/tessellate*.go` (the highest-priority subsystem) and must be golden-image
   validated, so it is its own careful step. Guarded by
   `ops.TestReconstructCocylindricalCapOnWall` (skipped, with the full diagnosis).

2. **Curved SSI-edge welding (e.g. cyl ∪ box).** Reconstruction recovers the analytic
   surfaces but the plane∩cylinder ellipse/line edges do not yet weld watertight; Layer 4
   (exact SSI conics welded consistently on both incident faces) closes it.

Until both land, the recovery gate stays off so the corpus is unchanged.

## Alternatives considered and rejected

- **Fit surfaces from the facet cloud (RANSAC / least-squares).** Rejected: tolerance-bound, cannot
  recover the exact radius/axis, and needless — the exact surface is already known via provenance.
- **Re-attribute output facets to surfaces by geometric on-surface test.** Workable but tolerance-based
  for curved surfaces (facet vertices are chord points, not on the true surface); exact tag propagation
  is strictly better and also feeds TNP identity.
- **Keep the mesh engine faceted and only special-case #2167.** Rejected: leaves every other curved
  boolean faceted; #2167 is a symptom of the general faceting, not a special case.
