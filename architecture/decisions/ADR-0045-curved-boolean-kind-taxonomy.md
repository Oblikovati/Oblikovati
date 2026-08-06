# ADR-0045 — Curved-boolean KIND taxonomy

**Status:** Accepted (2026-07-01). · **Closes / concludes** the EPIC
[Oblikovati#1403](https://github.com/Oblikovati/Oblikovati/issues/1403) (fold the bespoke
analytic curved-boolean handlers into the general pipeline). · **Builds on**
[ADR-0027](ADR-0027-curved-face-boolean.md) §M2 (the exact analytic curved boolean),
Oblikovati#1476 (the general SSI → (u,v)-arrangement → classify → stitch pipeline),
[ADR-0043](ADR-0043-generalized-provenance-naming.md) (curved-stitch edge provenance) and
[ADR-0042](ADR-0042-model-scale-and-relative-tolerances.md) (model-relative tolerances). ·
**Touches:** `kernel/ops/boolean.go` (`curvedExactPaths` dispatch),
`kernel/ops/boolean_crossing_cylinder.go`, `kernel/brep/curved_coaxial_cylinder.go`,
`kernel/brep/curved_cylinder_boss.go`, `kernel/brep/drill.go`.

## Context

EPIC #1403 set out to replace the ~21 bespoke per-primitive-pair analytic curved-boolean
handlers with **one general pipeline**: trace the surface–surface intersection (SSI) imprint,
project it into each operand's `(u,v)` band, subdivide that band with a planar arrangement,
classify each cell by 3-D solid membership, and re-emit the kept boundary as exact analytic
edges (`kernel/brep/curved_halfspace_uv_*`). Over #1403 that pipeline absorbed **every
transversal ruled∩ruled crossing** — crossing cylinders, cone∩cone, cone∩cylinder, partial
penetration, and finally the equal-radius Steinmetz bicylinder (intersect, cut and join;
PR#1587/#1589), whose self-intersecting imprint is split at its analytic pinches into four open
arcs so the arrangement never sees the crossing.

Three bespoke handlers remained: **DrillThroughHole** (box − cylinder), **CoaxialCylinderUnion**
(two coaxial equal-radius cylinders), and **JoinCylindricalBoss** (cylinder seated on a plate).
The open question was whether these should also be routed through the `(u,v)` arrangement, or
whether they are something else.

Tracing the actual `uvSide` seam settles it. The `(u,v)` arrangement is built **exclusively for
periodic parametric surfaces**: `paramOf` reads `u` as an azimuth (`atan2` about the axis, minus a
placed `seamU`) and `v` as axial distance between two **rim circles**; the segment tags are
`segImprint`/`segRim`/`segSeam`; `clipParams`/`sampleImprintUV` clip to an axial band
`[vMin, vMax]`; `wrapsAllU` tests azimuthal wrap. This machinery earns its keep only when the
imprint is a **1-D curve that non-trivially subdivides a periodic band** (multi-valued in `v`, or
self-crossing). None of the three remaining handlers is that:

- **Coaxial** — the two cylinders' side surfaces are **coincident** (the same `geom.Cylinder`).
  There is no transversal crossing curve at all; the contact is a 2-D region. SSI is structurally
  undefined, so the arrangement cannot apply.
- **Drill / Boss** — the tool cylinder crosses the slab's/plate's **planar** faces in a circle
  that lies **strictly inside one face**. A flat, bounded, non-periodic polygon face has no
  azimuth seam and no rim circles; a `planeUV` operand would have to *fabricate* both to satisfy
  the interface. And the result of a strictly-interior circle is trivially "add the circle as an
  inner loop" — the arrangement's cell subdivision buys nothing.

Meanwhile the dispatch is already sound: each path is gated once by `gatedCurved` (op guard +
`validBooleanSolid`), and a boolean that reaches triangle-soup CSG is **already recorded** as a
`CodeBooleanCSGFallback` defect (#1407) — not silent. The O(n²) *explosion* the EPIC targeted was
the transversal ruled family (18 near-identical handlers), which is folded. The remaining three
are one general analytic handler **each**, generic over face count and model-relative in
tolerance — not per-pair bespoke growth.

## Decision

**Curved booleans form three KINDs, distinguished by the dimensionality of the operands' contact.
Each KIND has one general analytic handler; only the transversal KIND uses the `(u,v)`
arrangement. The taxonomy is the deliverable, not a forced merge into one function.**

| KIND | Contact | SSI defined? | Handler | Examples |
| --- | --- | --- | --- | --- |
| **Transversal crossing** | 1-D curve, surfaces cross | yes | the general SSI → (u,v)-arrangement → classify → stitch pipeline (+ curved∩convex-planar half-space cuts) | crossing cyl, cone∩cone, cone∩cyl, partial penetration, Steinmetz |
| **Curved-on-planar (interior)** | one closed conic **strictly inside** a planar face, added as an inner loop | degenerate (no band to subdivide) | build the pierced face + wall + cap and `curvedStitch` | drill through-hole, cylinder boss |
| **Curved-on-planar (partial)** | the imprint conic **CLIPS** the planar face boundary (pierced but not clean) | degenerate | trim the pierced face(s) through a bounded non-periodic `planeUV` `(u,v)` arrangement + assemble the partial wall/cap (ADR-0049) | edge scallop (partial drill), straddling boss |
| **Degenerate overlap** | 2-D region of **coincident** surfaces | no | simplify to the merged analytic solid | coaxial cylinder union |
| **Transversal, split by construction** | 1-D curve that is one **planar conic bisecting a closed surface** — the surface has no rims or seam to subdivide | yes, in closed form | name which of the two regions survives and `curvedStitch` | coaxial ball ∪/−/∩ rod (the ball stud), ADR-0045 addendum below |

Concretely:

1. **Do not build a `planeUV` operand.** Forcing drill/boss through the periodic arrangement would
   fabricate a seam and rims that do not exist — ceremony that distorts the abstraction to fit a
   case that needs no subdivision, on correctness-critical, already-tested code.
2. **Recategorize coaxial** as the degenerate-overlap KIND (documentation + dispatch tag); it is a
   coincident-surface *simplification*, not an SSI handler. It stays a gated path (low risk); it is
   not promoted to a pre-normalization pass, because the gated form already produces the exact
   solid and a pre-pass would add control-flow risk for no correctness gain.
3. **Unify the one genuine duplication.** `CutCylindricalHole` carried a second, bespoke planar
   welder (`classifyDrillFaces` + `assembleDrilled`) that re-implemented what the `curvedStitch`
   drill path (`drillThroughCurved`) already does. `CutCylindricalHole` now delegates to that
   shared path — one drill assembly, not two. The shared welder helpers (`weldPlanarFaces`,
   `buildHoleEdges`, …) remain, used by the blind/counterbore/countersink variants.
4. **Name the KIND on every path and file** (dispatch `[T]`/`[P]`/`[D]` tags, file headers) so a
   future primitive pair is classified into an existing KIND, not appended as new bespoke growth.

## Consequences

- **Buys:** an honest, discoverable taxonomy; the removal of ~60 lines of duplicated planar-welder
  code; the load-bearing invariant stated once — *the `(u,v)` arrangement is for transversal
  crossings of periodic surfaces, nothing else*; and EPIC #1403 concluded on its real intent (kill
  the O(n²) bespoke explosion + no silent CSG), both already met.
- **Costs:** the interior drill and boss stay outside the transversal `(u,v)` pipeline (they are a
  distinct KIND, not SSI). The **partial** curved-on-planar contacts (a hole clipping a face edge, a
  boss straddling an edge) were the one gap this taxonomy left to CSG; that gap is now **closed** by
  [ADR-0049](ADR-0049-partial-curved-on-planar-planeuv.md), which added them as their own row above
  via a bounded non-periodic `planeUV` `(u,v)` arrangement — a *new feature* built on this taxonomy,
  not a refactor of the interior handlers.
- **Dispatch order is unchanged** (the try-order within an op is load-bearing); the KIND tags are
  annotations, so this decision carries **zero behavioral risk** and is validated by the existing
  brep/ops/model boolean regression suites staying green.

## Rejected alternatives

- **Build a `planeUV` operand and route drill/boss through `trimByImprint`.** Rejected: distorts a
  periodicity-centric abstraction to fit a flat face, adds machinery to a case that needs no cell
  subdivision, and is high-risk churn on correctness-critical tested code — the opposite of "the
  lightest structure that protects the real invariant." The one legitimate upside (partial-contact
  analytic coverage) is a separable new feature, not a reason to rebuild working handlers.
- **Fold coaxial into the SSI pipeline via an ε-perturbation** to manufacture a crossing. Rejected:
  invents a spurious sliver imprint at exactly the tolerance ADR-0042 warns about.
- **A heavyweight `classifyContact` pre-classifier replacing the gated try-list.** Rejected: the
  gated list already declines cleanly and logs its CSG fallback; a classifier would be ceremony
  isolating no new invariant.

## Addendum — 2026-08-06: a transversal crossing that is still not an arrangement (#2036)

`ops.Boolean` had no entry for **sphere ∪ cylinder**, so a ball stud (a ball head on a coaxial shank)
fell through to triangle-soup CSG and shipped an inscribed polyhedron 1.3% under volume. Closing that
gap adds a row to the table above rather than bending an existing one, because the pair is transversal
— the surfaces genuinely cross in a 1-D curve — yet the `(u,v)` arrangement still does not apply.

The load-bearing invariant this ADR states is *"the `(u,v)` arrangement is for transversal crossings of
**periodic** surfaces"*, and the emphasis is doing the work: `paramOf` reads `u` as an azimuth and `v`
as axial distance **between two rim circles**. A sphere has no rims. It also has no meaningful seam —
`geom.Sphere`'s is a parameterisation artefact fixed to world `+Z`, unrelated to where the rod enters.
And when the rod is COAXIAL with the ball, the contact is a single **planar circle**, which splits a
sphere into exactly two caps by construction. There is nothing for a cell classifier to decide: naming
which cap survives *is* the split. So the handler is a direct assembly (three analytic faces) in
`kernel/brep/curved_coaxial_sphere_rod*.go`, in the same spirit as the drill and the boss.

Two consequences worth recording:

- **The recognizer is a port, not an invention.** OCCT special-cases exactly this pair in
  `IntAna_QuadQuadGeo::Perform(gp_Cylinder, gp_Sphere)`: an axis through the sphere centre yields
  `IntAna_Circle` at ±√(R_s²−R_c²) along the axis, and every other configuration is
  `IntAna_NoGeometricSolution` — handed to the numeric marcher. We decline the same three cases OCCT
  declines (off-axis, ball no larger than the rod, equal radii). An off-axis sphere∩cylinder is a
  quartic space curve and stays on the CSG fallback.
- **Winding, not sense, names the region on a closed surface.** For a face bounded by one circle on a
  sphere, the loop direction is what selects which cap survives, and `kernel/ops` reads exactly that
  (`capAxis`, `sphere_cap_mesh.go`). The cylindrical band and the planar disc cover the same region
  either way, so they take whatever direction the cap leaves them — that is how the assembly satisfies
  `ops.Validate`'s anti-parallel edge-use invariant while still meshing the intended cap. Getting it
  backwards costs nothing at build time and yields a closed, manifold solid of the RIGHT volume that
  `Validate` rejects only on orientation.

**Scope left open.** A rod passing right THROUGH the ball meets it in *two* circles, and the surviving
ball face is then the belt between them — a spherical zone straddling the equator of its own band axis.
`kernel/ops` has no analytic mesh for that shape (measured: ~75% of its area goes missing), which is why
`revolution.go`'s `sphereZoneAnalytic` also gates equator-crossing zones out. That configuration keeps
the faceted CSG fallback until the zone mesh exists.
