# ADR-0059 — Two conics in a plane meet through ONE intersector, bucketed by representation

**Status:** Accepted — on `m48/finish-ops-structure`. · **Scopes**
[Oblikovati#3503](https://github.com/Oblikovati/Oblikovati/issues/3503) (a wrapped pad's cap faces bounded
by chords) and part of [#3459](https://github.com/Oblikovati/Oblikovati/issues/3459) (the flush-contact
half of the `ops.Facet` deletion). · **Builds on**
[ADR-0058](ADR-0058-tolerant-analytic-boolean.md) (the tolerant analytic boolean, whose imprint layer this
serves) and [ADR-0042](ADR-0042-model-scale-and-relative-tolerances.md) (tolerance classes). · **Deletes:** the
`allStraightFace` gate in `uvWallSharedImprint`; the ring-polyline walk in `conicCrossesFaceBoundary`; the
copy of the Ferrari quartic solver that sat in `kernel/ops/blend` where `kernel/geom` could not reach it;
and `clipSectionToFace`'s "one arc between the outermost crossings" contract. · **Touches:** `kernel/geom`
(the intersector and the relocated solver), `kernel/brep` (edge crossings, the section clip),
`model/feature` (wrapped pad edges).

## Context

Nothing in the kernel could meet a conic with a conic.

Every planar routine that had to decide whether a section curve enters a face walked that face's boundary
as a POLYLINE. That is exact only while every boundary edge is straight, and `allStraightFace` was the
gate that admitted it: a face carrying an arc was refused outright, whatever the operation.

The gate is reachable from ordinary modelling. A wrapped emboss pad lays its profile on a cylinder or a
cone and skins between two offset caps. Its loop edges were straight segments between points of that cap —
chords through the solid, not curves on it — so the cap faces they bounded were not valid geometry however
small the sag. On a 1 cm glyph edge at radius 15 the chord left the cone by 8.3e-3 cm, which put the pad's
outward vector area 7.1e-3 short of closing and sent every analytic mass property of a wrapped emboss to
the tessellated fallback (#3503). Cutting each edge from its own wall's plane fixes that exactly — the
section passes through both endpoints by construction, so the edge lies on BOTH faces it bounds — but the
result is a planar face with conic boundary edges, which is precisely what the gate refused.

A conic meeting a conic is a quartic. The kernel already had a Ferrari solver for one, in
`kernel/ops/blend`, where `kernel/geom` cannot import it.

## Decision

**Pair conics by REPRESENTATION, not by kind.** One conic is taken IMPLICITLY, as the quadratic form it
satisfies (`geom.Conic2dImplicit`); the other PARAMETRICALLY, as a point moving along it
(`geom.EllipticalParams2d`). Substituting the second into the first collapses every pair — circle ×
ellipse, ellipse × hyperbola, arc × arc — to one scalar equation in one parameter.

This is the kernel rule already on the books ("Intersectors are bucketed by representation (implicit /
parametric), not by N² type pairs") and it is how OpenCASCADE does it: `IntAna2d_Conic` carries the
implicit coefficients and each `IntAna2d_AnaIntersection::Perform` substitutes its parametric partner into
them, handing the result to one trigonometric root finder.

- An **ellipse** substitutes to `a·cos²t + 2b·cos t·sin t + c·cos t + d·sin t + e = 0`, which the
  Weierstrass substitution `u = tan(t/2)` turns into a quartic. The half-turn `u` cannot reach is
  recovered exactly, not approached: at `t = π` the equation reduces to the quartic's own leading
  coefficient, so that root is present exactly when the quartic drops to cubic.
- A **hyperbola branch** substitutes to a quartic in `w = eᵗ`, since cosh and sinh are that exponential's
  half-sum and half-difference. Only positive roots are branch parameters.
- **Coincident** conics are reported as such. A caller that read "no crossings" would treat two identical
  curves as disjoint.

**The Ferrari solver moves down into `kernel/geom`.** A quartic is what every conic question reduces to;
the blend engine's line-vs-offset-torus tangency and this substitution are two callers of one solver, at
the layer both reach. It is not copied.

**`brep` walks a face's own EDGES, not its ring points.** Those agree while every edge is straight and
part company the moment one is an arc, where a ring walk chords it — exact for neither the crossing count
nor the tangency.

## Consequences

The cone-wrapped emboss closes to 1.2e-13 of its own area, where it managed 3.0e-8 with chorded edges and
7.1e-3 before the pad's orientation was fixed, and integrates to 45029.796846186 against an identical
mesh.

Two conic-boundary defects became reachable and were fixed with it. `clipSectionToFace` bounded a section
to the span between its OUTERMOST crossings, which spans the gaps when a section enters and leaves a face
more than once — a bore's section circle crossing a bar's footprint produced one arc running through
material the bar does not cover. And it could not bound a CLOSED conic at all, because `ConicSubArc`
declines those by contract, so a circular section was refused whatever its placement.

**What this does NOT do.** It does not deliver the flush-contact half of #3459. The coplanar path
(`boolean_coplanar.go`) works in 3-D segments, so a conic-bounded face still cannot be imprinted onto a
coplanar partner; building that in curve currency was attempted on this branch and reverted, because
wiring it into the mixed dispatch moved the decline through four successive gates and ended at the
polygonal split itself. That is the planar/curved unification ADR-0058 leaves for last, and it wants its
own decision record rather than another widened gate. The measurement and the four decline points are
recorded on issue #3459.

**Net delta.** Kernel type assertions 776 → 774, `kernel/brep` geometry type switches 104 → 102: the
curve-kind switch this capability needs lives in `geom.PlaneConicParams`/`IsPlaneConicCurve`, so `brep`
ends with fewer than it started, not more. Both archguard pins are lowered in the same commit.

**Scope discipline.** Two capabilities built on this branch were deleted rather than shipped, because
neither had a production caller: uv×uv imprint pairing (instrumented across `kernel/brep` and
`kernel/ops/boolean`, zero overlapping pairs in the whole corpus) and a parametric-LINE bucket for the
intersector (its caller died with the coplanar revert). An engine with no caller is the thing the ground
rules forbid, whatever its quality.
