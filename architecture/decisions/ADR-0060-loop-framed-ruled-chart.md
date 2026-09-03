# ADR-0060 — A ruled wall is trimmed inside the frame its own loops make

**Status:** Accepted — on `m48/face-sense-invariant`. · **Scopes**
[Oblikovati#3459](https://github.com/Oblikovati/Oblikovati/issues/3459) (delete `ops.Facet`, the permanent
faceting of an analytic operand) and [#3329](https://github.com/Oblikovati/Oblikovati/issues/3329) (its
Validate post-condition, moot once it is gone). · **Builds on**
[ADR-0058](ADR-0058-tolerant-analytic-boolean.md) (the mixed per-face boolean whose wall bucket this
generalises), [ADR-0059](ADR-0059-plane-conic-intersection.md) (the exact conic crossings the frame
injects) and [ADR-0043](ADR-0043-generalized-provenance-naming.md) (a wall keeps its lineage through the trim). ·
**Deletes:** `ruledSideBandOf` and the two-full-circle admission it imposed on a wall; the keyhole
bridge as the representation of a holed tube in the mixed path; the whole-body volume sign as the one
bit that oriented every shell; `ops.Facet`, `boolean.Facet`, the rescue that faceted a boolean's operands
after an invalid analytic result (`facetedBooleanRescue`, `facetedRescue`), and the tests that only
exercised them. · **Touches:** `kernel/geom` (ruled frames, section planes, curve inversion, crossing
candidates, axial extents), `kernel/brep` (the chart, the wall recogniser, the planar exact frame, the
face-pair imprints, shell orientation), `kernel/ops` (the deleted operation), `model/feature` (the
deleted rescue).

## Context

The mixed per-face boolean (ADR-0058) buckets every face by REPRESENTATION: a polygonal planar face
splits in 2D, a planar face with conic edges splits in an exact-frame chart, and a ruled side — a
cylinder or a cone — splits in a `(u,v)` chart. That last chart framed the side with two SYNTHETIC
constant-`v` rims and a seam, and admitted a wall only when the face's loop carried exactly two whole
circles. Every other cylinder face — one whose end an earlier feature had cut obliquely, one already
notched, a partial patch — fell to the pass-through bucket, which can carry a face only when it is
provably clear of the other operand. A tool that met it declined the whole boolean, and the boolean
fell to triangle CSG.

That decline was the last road to `ops.Facet`. The multipoint disk of the Inventor corpus cuts a 2 mm
pin whose lower end a 7.5° plane had trimmed: its side is bounded by a circle and an ellipse, so it
was not a wall, the profile extruded across it declined, the CSG fallback built 12 234 planar faces
that did not close, and the rescue faceted both operands and tried again. Faceting is permanent: every
downstream fillet, thread, mass property and export then reads facets. The kernel ground rules name
this class — a second engine shipped beside the first as a fallback — a defect, and #3459 tracks its
deletion.

The recogniser was the special case. An already-cut side had a second chart (`cutCylinderUV`) that
composed one prior loop as a constraint, gated to imprints disjoint from it; a holed tube had a bridged
"keyhole" outer loop assembled from the synthetic rims; a bare band had the rims themselves. Three
frames for one question: where, on this ruled surface, does the face's own boundary lie.

## Decision

**A wall is any cylinder or cone face whose every boundary edge is a ruling or a plane section, and its
frame IS its loops.** `ruledFaceUV` samples the face's boundary into the same `(u,v)` arrangement the
band chart used, tagged with the exact curve each segment lies on, and injects every frame×imprint and
frame×seam incidence as a shared vertex. Those incidences have closed forms on a ruled quadric: two
plane sections meet where the line their planes share pierces the surface (`geom.SectionCrossingCandidates`),
a ruling meets a section where it pierces that section's plane, and a point on a section curve inverts
to that curve's own parameter (`geom.CurveParamAt`). The chart's material predicate is even-odd
containment in the sampled frame AND the boolean's keep table over the other operand's membership; its
seam is placed clear of every imprint azimuth, every frame vertex and every ruling edge, so it crosses the
frame only through the interior of a smooth section edge, where its incidence is exact. The bare
two-rim band, the obliquely-cut side, the notched side and the partial patch are one case.

**A holed tube is two wrapping loops plus its holes, with no bridge.** The keyhole bridge needs a vertex
on each end loop at one common azimuth. The neighbouring faces carry those ends as whole closed edges
whose single vertex sits at the curve's own parameter origin, and two ends need not share one; a bridge
elsewhere inflates the vertex count and breaks the Euler count. The tessellator already meshes the
two-rim holed band (`twoRimHoledBandMesh`), and the generalised Euler formula admits it.

**Every shell orients itself.** The face signs the winding classifier and the stitch derive were settled
by ONE bit for the whole face set — the sign of the summed signed volume. A pin cut clean through leaves
two lumps, each seeded by its own colouring; the sum of a positive lump and an inverted one stayed
positive and the inverted one was never turned. Each face's sign is now read ORIENTATION-FREE: a point
a stand-off along the face's geometric normal, and one against it, classify by ray parity against the
face's own shell, and the sign is the one that puts the non-material side on the normal. A shell whose
box lies within an already-oriented shell's box and whose point classifies inside it is a VOID, whose
material is outside. The loop handedness that used to decide is the fallback for a face the probe cannot
read (a wall thinner than the stand-off, or every ray grazing), under a bit the probed faces of its
shell vote on. On the stubs a near-pinch cut leaves, the handedness read two of a stub's three faces
wrong; the balance of those errors is what had kept the old sum green.

**The planar exact frame takes every conic edge, and two exact-framed faces imprint each other.** The
plane chart solved crossings only on whole-circle edges and refused any pair of two conic-framed faces
outright; the pin's oblique cap is an elliptical edge, and the tool faces that receive the wall's
sections are conic-framed too. The crossing is solved on the whole conic and kept within the arc's own
span; the pair's imprint is the plane∩plane line clipped to both faces' exact intervals. A bounded arc
of a conic is an OPEN imprint (it ends on the frame it was clipped to), whatever conic it runs on — filing
it as straight emitted its chord, and the chord did not weld to the arc the wall re-emits.

**A flush contact on an exact-framed face is modelled, not refused.** The disk's next feature rests a tool
face in the plane of a stub's cap — a circle rim with a rectangular hole — and every coplanar pair carrying
a conic loop used to decline outright. Now a polygonal face coplanar with a conic-edged face is promoted
to the exact-frame chart (`promoteCoplanarReceivers`), the pair exchanges outlines EXACTLY — the other's
straight edges clipped to the face's conic even-odd intervals, its whole conic edges entered as islands or
clipped arcs — and a cell covered by a coplanar face of the other operand follows the ON/ON table
(`coplanarKeep`) through exact containment (`faceContainsExact`) before the membership oracle is asked. A
wall imprint lying IN a frame edge's plane is a boundary contact and is dropped, as the polygonal split drops
a segment on its own boundary. A partial conic edge on a coplanar face is the one shape this does not yet
take (#3508 keeps that).

**Shared edges subdivide identically, whichever chart built each side.** Every incidence a chart solves —
an imprint end on a frame edge, a frame×imprint crossing — is a vertex on the neighbour built from the same
interaction, so the chart restores it on its re-emitted edges even where the imprint between two kept
cells dissolved (`splitLoopsAtPoints`), and the mixed stitch conforms every pass-through face's edge, conic
or straight, to the welded vertex set (`weldPassVertices`, `splitPassTJunctions`), the discipline its
polygonal rings already had.

**A wrapping rim is read as one full turn.** Both (u,v) containment readers sampled a rim one step short of
its turn and left that step open, so a query in it read a band's interior as outside — the wall then
counted no crossing, a parity probe flipped a cap, and the analytic vector area no longer closed. Each
reader now closes a wrapping ring on its periodic image and reduces the query into that full-period span
(`closingImage`, `reduceIntoRingSpan`, `ringsEnclose`).

## Consequences

- Recognisers: `ruledFaceOf` replaces `ruledSideBandOf` (net 0). `fullCylinderSideBand` and
  `fullConeSideBand` remain for the half-space and ruled∩ruled paths, which still frame by band; the
  ticket that migrates them to the loop frame and deletes the band frame is the follow-up this ADR
  names, and `cutCylinderUV` goes with it.
- Cost. The chart admits many more faces into the wall and uv buckets, so a configuration it cannot
  model is now found after the arrangement rather than at `passClearOf`. On the fine-pitch coil join
  that is 21.7 s spent on a decline (45.98 s → 70.53 s for one pitch, 7 012 faces), tracked by
  [#3511](https://github.com/Oblikovati/Oblikovati/issues/3511) — it closes by MODELLING the contact
  (ADR-0061 stages 2 to 5) or by classifying scope before the imprints, never by narrowing the buckets
  back. Two costs that were NOT inherent were fixed here instead: the shell orientation measured each
  face's boundary band once per ray rather than once per face, and the chart's edge recovery scanned
  every segment per dedge (now grid-indexed, `uvSegIndex`) — together 253 s → 70 s on the same case.
- Kernel type assertions: −9 (774 → 765), `kernel/brep` geom-kind switches: 102 → 93 (the switches
  moved behind `geom.RuledFrameOf`, `geom.SectionPlane`, `geom.CurveParamAt`, `geom.AxialExtent`,
  `geom.AsConic`); both ratchets lowered. Fallback sites are net 0 here: the two rescues go, and the
  invalid-analytic degradation ADR-0061 reports takes their place.
- Tolerances: one new dimensionless ratio (`probeOffsetRel`, the probe stand-off as a fraction of the
  shell diagonal); none absolute.
- The Inventor multipoint disk rebuilds as ONE closed solid, which it had not done since the part
  entered the corpus. Reaching that took one more defect, in the membership classifier rather than in
  any chart, and ADR-0061 carries it: a sub-face point lying in a plane the other operand shares, but
  on no face of it, is answered by probing both sides rather than by a query the point itself makes
  degenerate. A partial conic edge on a coplanar face is still out of scope (#3508); the band-frame
  consumers (half-space, ruled∩ruled) migrate to this chart under #3509; the STEP writer for an
  elliptical-arc edge is #3510.
- `ops.Facet` has no caller. It is deleted, with its alias, its tests, and the two rescue functions.
  Tests that pin a planar-only path build their faceted fixture through `kernel/internal/testcage`.
