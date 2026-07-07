# ADR-0050 — OCCT-parity blend engine: one section-driven pipeline for fillet/chamfer, a separate modifier-visitor for draft

**Status:** Accepted (2026-07-07). · **Sets the direction for** fillet/chamfer/draft feature
parity with OpenCASCADE (OCCT), reported from the Oblikovati Discord `#bug-tracker`:
[Oblikovati#1797](https://github.com/Oblikovati/Oblikovati/issues/1797) (fillet all-around drops
tangency → faceted top), [#1798](https://github.com/Oblikovati/Oblikovati/issues/1798)
(no tangent-chain selection), [#1800](https://github.com/Oblikovati/Oblikovati/issues/1800)
(large-radius distortion, no validity check), [#1801](https://github.com/Oblikovati/Oblikovati/issues/1801)
(draft pull/neutral-plane inputs), [#1802](https://github.com/Oblikovati/Oblikovati/issues/1802)
(draft-after-fillet crash). · **Builds on** [ADR-0027](ADR-0027-curved-face-boolean.md) /
[ADR-0043](ADR-0043-generalized-provenance-naming.md) (SSI machinery + order-independent face-pair
provenance naming, reused for blend edge naming), [ADR-0042](ADR-0042-model-scale-and-relative-tolerances.md)
(model-relative tolerances). · **Touches (new, over the roadmap):** `kernel/blend/` (new pure
engine package), `kernel/ops/modify/` (new draft modifier-visitor), `model/feature/fillet.go` /
`chamfer.go` / `draft.go` (orchestrate the engine), the `Oblikovati.API` dressup wire DTOs (draft
pull/neutral, chamfer modes), `app` selection (tangent-chain), `head/ui` dialogs. · **Supersedes**
the plane-only draft retopo (`kernel/ops/retopo.go rebuildWithPlanes` for the draft path) and the
`planarizedFillet` faceting fallback — retired case-by-case as the marcher covers each.

## Context

Our fillet/chamfer/draft suite is a **catalog of analytic special-cases with a faceting fallback**,
not an engine. Fillet emits exact `geom.Cylinder`/`Sphere`/`Torus`/`Cone` (+ rational ruled NURBS
for plain-ended variable) for the primitive cases it recognizes and otherwise **planarizes the body
and facets** (`model/feature/fillet.go:167`). Chamfer is an analytic cone for a simple cylinder rim
or a **planar-wedge boolean cut** that re-facets curved edges. Draft is routed through a
**plane-only retopo** (`rebuildWithPlanes`) that casts every face to `geom.Plane` and every edge to
a line segment — so it **panics on any curved face** (#1802) and cannot taper a body that carries a
fillet. There is no tangent-edge chain propagation (tools append single picks), and no geometric
max-radius/self-intersection validity (`Validate` is topological only).

This shape is the failure, not any single bug: a special-case catalog can never reach general
variable-radius blends, curved neighbours, or robust corners. OCCT solves the same problem with a
**seam**, verified in its source (`TKFillet`, `TKOffset`, `TKBRep`): fillet and chamfer are **one
`ChFi3d_Builder` pipeline** that differs only by a pluggable **section functional**
(`BlendFunc_ConstRad`/`EvolRad`/chamfer families) driving a generic ODE **marcher + B-spline
approximation** (`BRepBlend_Walking` + `BRepBlend_AppSurface`) along a tangent-continuous
**spine** (`ChFiDS_Spine`); an analytic **known-part** fast path (`ChFiKPart_ComputeData`)
short-circuits the marcher for primitive face pairs. **Draft is a wholly separate mechanism** — a
`BRepTools_Modifier` visitor (`Draft_Modification`) that walks every face/edge/vertex, tilts the
changed surfaces, and re-intersects neighbours analytically (`IntAna_QuadQuadGeo`), preserving
topology with no marching. Errors are localized per-stripe/per-vertex (`ChFiDS_ErrorStatus`),
yielding a partial result (`HasResult`/`BadShape`) rather than a global failure.

Crucially, the numerical foundation OCCT's engine needs, **we already own**: a general SSI marching
intersector over any surface incl. NURBS (`geom.IntersectSurfaceSurface`, predictor-corrector,
tangency-singularity aware), analytic SSI for plane-involving pairs, a general `OffsetSurface` with
fold rejection, the exact analytic blend surfaces, and the curved-boolean trim/classify/stitch +
ADR-0043 provenance naming. What is absent is the **blend-surface abstraction and the driver** that
compose those pieces — plus tangent-chain grouping, geometric validity, curved-neighbour
re-intersection, and a curved-capable draft.

## Decision

Adopt OCCT's **architecture** (the seam) and implement it on **our existing numeric assets** — a
**hybrid**, not a wholesale C++ port and not more special-cases.

- **One pipeline, pluggable section.** A new pure package `kernel/blend/` builds a blend from a
  **`Spine`** (tangent-continuous edge chain) by driving a generic **`Marcher`** parameterized by a
  **`SectionFunctional`** port (`ConstRadiusSection`, `EvolRadiusSection`, `ChamferSection`
  sym/two-dist/dist-angle) and fitting the sampled sections to a `geom.BSplineSurface`
  (`approx.go`). Fillet and chamfer are the same pipeline with different section functionals.
- **Known-part fast path retained.** The existing well-tested analytic paths
  (`fillet_analytic`/`arc`/`rim`, `chamfer_analytic`) are **re-seated behind `kernel/blend/knownpart.go`**
  as the closed-form short-circuit — exact where it applies, marcher elsewhere. They are wrapped,
  not rewritten; goldens lock them byte-identical.
- **Draft is a separate modifier-visitor.** A new `kernel/ops/modify/` package provides a
  `Modifier` driver + `Modification` port (`NewSurface`/`NewCurve`/`NewPoint`) and a
  `draft_mod.go` that tilts selected planar/cyl/cone faces and re-intersects neighbours via our
  analytic + general SSI. It replaces `rebuildWithPlanes` for draft; `shell`/`move-face` may migrate
  onto it later (they share the same latent panic).
- **Shared tangent-chain primitive.** The G1 face-tangency predicate + chain walk lives once in
  `kernel/blend/tangent.go` and is consumed by **both** the engine (spine building) and `app`
  selection (expand a pick to its tangent chain). No duplication.
- **Localized error, partial result.** Mirror `ChFiDS_ErrorStatus`: a per-stripe/per-vertex failure
  yields a Sick-with-hole result naming the faulty entity, not a global abort — no pre-flight
  validity oracle. A geometric max-radius/self-intersection check (absent today) gates each segment.
- **Dependency rule.** `kernel/blend` and `kernel/ops/modify` import only `kernel/geom` +
  `kernel/topo`; `model/feature` orchestrates them; nothing in the engine imports `model`, `app`,
  or `head`. New public surface (draft pull/neutral, chamfer modes) is contract-first in
  `Oblikovati.API` per ADR-0018.

Delivered as a **phased milestone** (each phase independently shippable + tested): **0** draft
crash-guard (stopgap, feature goes Sick) → **1** tangent-chain selection/spine (feeds the existing
analytic paths first) → **2** geometric max-radius validity → **3** engine skeleton (`SectionFunctional`
port + known-part re-seat, zero behavior change) → **4** marcher core (general constant-radius on
curved neighbours; retires faceting case-by-case) → **5** variable-radius laws + chamfer modes →
**6** corner/setback reconstruction → **7** curved-draft modifier-visitor (the real #1802 fix).
Phases 4 and 6 are the hard IP and route their numerics through `geometry-math-advisor`.

## Consequences

- **(+)** Reaches general **variable-radius**, **curved-neighbour**, and **tangent-chain** blends
  that the special-case catalog structurally cannot — the actual parity goal.
- **(+)** Reuses the SSI tracer, offset surface, NURBS, and ADR-0043 naming we already own; the hard
  new code is the marcher/approximation and corner processor, not the whole stack.
- **(+)** Existing analytic results ship **unchanged** (re-seated as known-parts, golden-locked); the
  faceting fallback is retired only where the marcher demonstrably covers the case, so no regression.
- **(+)** Draft-after-fillet stops crashing at phase 0 and becomes correct at phase 7; the modifier
  framework then generalizes shell/move-face off the plane-only retopo.
- **(−)** The marcher (phase 4) and corner processor (phase 6) are genuinely hard numeric/topology
  work — OCCT's own weak points (≥4-edge vertices unsupported). We inherit that difficulty and adopt
  its partial-result honesty rather than promising totality.
- **(−)** We temporarily carry two code paths (known-part + marcher, and the retiring faceting
  fallback) until the marcher matures; the diagnostic counters (`CodeFilletFacetedBlend`) track the
  crossover.
- **(0)** Leaning on the curved boolean for blend re-intersection inherits its in-progress
  ~28-handler state (ADR-0027/#1403); the blend engine uses SSI + trim/stitch directly where the
  general boolean is incomplete.

## Rejected

- **Clean-room wholesale port of `ChFi3d`/`BRepBlend`.** Reimplementing `TopOpeBRepDS`, OCCT's
  approximator, and ~15k lines of corner code in Go duplicates SSI/NURBS machinery we already have —
  multi-quarter, high regression risk, no reuse. We adopt OCCT's *seam*, not its line-for-line
  internals.
- **Incremental-only (keep growing the analytic catalog).** Cheapest short-term but never reaches
  general variable-radius/curved-neighbour/robust corners — it is the very shape failing in
  #1797/#1798/#1800/#1802.
- **Guard the draft assertion and stop there.** Fixes the panic but leaves draft plane-only forever;
  kept as **phase 0** (the honest stopgap), not the destination.
- **Fold draft into the blend pipeline.** Draft is geometry substitution with topology preserved,
  not a marched blend; OCCT keeps them separate and so do we — a shared marcher would be the wrong
  abstraction.

## References

OCCT `TKFillet` (`ChFi3d_Builder`, `ChFiDS_Spine/Stripe/SurfData`, `BRepBlend_Walking`,
`BRepBlend_AppSurface`, `BlendFunc_ConstRad/EvolRad`, `ChFiKPart_ComputeData`), `TKOffset`
(`BRepOffsetAPI_DraftAngle`, `Draft_Modification`), `TKBRep` (`BRepTools_Modifier`/`Modification`). ·
Krivoshapko & Ivanov (2015), *Encyclopedia of Analytical Surfaces* (canal/pipe blends). ·
software-architect-advisor and geometry-math-advisor briefs recorded on the parity milestone. ·
ADR-0027 (curved boolean / SSI), ADR-0043 (provenance naming), ADR-0018 (API split).
