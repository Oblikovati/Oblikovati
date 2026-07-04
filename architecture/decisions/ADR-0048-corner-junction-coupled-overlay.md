# ADR-0048 — Corner-junction analytic via a coupled (u,v) overlay

**Status:** Proposed (2026-07-04). · **Follow-on EPIC** from the closed
[Oblikovati#1724](https://github.com/Oblikovati/Oblikovati/issues/1724) /
[Oblikovati#1732](https://github.com/Oblikovati/Oblikovati/issues/1732), whose acceptance
deliberately scoped the corner-junction to an **observable CSG decline**; this ADR records the
design for the analytic result and is tracked by a new EPIC (TBD). · **Builds on**
[ADR-0045](ADR-0045-curved-boolean-kind-taxonomy.md) (the T/P/D KIND taxonomy),
[ADR-0046](ADR-0046-curved-boolean-cap-crossing.md) (the cap-crossing arrangement + the
OCCT-certification protocol it defines), [ADR-0043](ADR-0043-generalized-provenance-naming.md)
(curved-stitch edge provenance) and [ADR-0042](ADR-0042-model-scale-and-relative-tolerances.md)
(model-relative tolerances). · **Touches (when built):** the shared
`kernel/brep/curved_halfspace_uv_arrangement.go` walk (behind a new pre-split pass, not modified in
place), a new `kernel/brep/curved_corner_junction*.go`, the tangency gate, the arrangement golden
`kernel/brep/curved_arrangement_golden_test.go` (the byte-identical regression guard), and the OCCT
oracle harness (`experiments/occ-boolean-oracle`, `kernel/ops/*_certification_test.go`).

## Context

The disjoint partial-rim second cut ships and is OCCT-certified (#1735 build, #1736 moment +
60³-membership certification): a second curved cut whose SSI imprint lies **clear** of the prior
notch composes the prior boundary as constraint edges and reuses the shared `(u,v)` arrangement.

The **corner-junction** sub-case — the new imprint **crosses** the prior notch boundary — currently
declines observably, and must, because forcing it through the analytic path builds a
**non-watertight, manifold-but-geometrically-wrong solid** (instrumented: χ = 1 with 5 free edges —
a collapsed back entry-ellipse hole plus a dead-ended front bite arm). ADR-0046 already named this
the exact hazard the internal `validBooleanSolid` + volume-bracket nets cannot certify against.

The root cause is a **topological inconsistency introduced by an approximate combinatorial
decision**. The rod↔cylinder SSI imprint arrives as a **sampled polyline**; `Arrange` computes the
crossing of the imprint with the prior section conic on the polyline **chord**. On re-emission each
surviving edge is re-anchored to *its own analytic curve* (the prior arm to the ellipse, the bite
arc to the imprint), and the chord-crossing point is **not on the ellipse** — so the single junction
that should be one vertex splits into ≥2 near-coincident vertices. The input planar subdivision is
then inconsistent, and the de Berg DCEL face-traversal walk — which is *provably correct only on a
consistent subdivision* — dead-ends. This is precisely the failure Yap's Exact-Geometric-Computation
thesis and Kettner et al.'s robustness-failure catalogue document: **a combinatorial decision (which
curves share a vertex) made from an approximate numerical value corrupts the arrangement.**

## Decision

**Build the corner-junction as a fourth-and-hardest slice of KIND [T] via a coupled (u,v) overlay
with an exact curve–curve pre-split (Method C).** The combinatorial junction `∂B ∩ ∂R` is computed
exactly as a **shared vertex on both analytic curves**, both curves are split there, and only then
does the arrangement → Hoffmann face-tagging → de Berg walk run on the now-consistent subdivision.
An arrangement-based boolean is correct **only if** its input subdivision is topologically
consistent; the exact shared junction is that precondition, not an optimization. Every non-transversal
configuration keeps the observable `CodeBooleanCSGFallback` decline. Correctness is certified against
OCCT `BOPAlgo_BOP` on an independent moment set plus a point-membership audit, per ADR-0046.

### Exact combinatorial pre-split

The kept region is the regularized boolean `K = B ∩* R` on the periodic band
`(u = azimuth mod 2π, v = axial)`, `B` = below the prior notch, `R` = outside the rod. At each
crossing of the imprint polyline with the prior section conic:

- Substitute the polyline segment `p(t) = pᵢ + t(pᵢ₊₁ − pᵢ)` into the prior ellipse's implicit
  quadratic, yielding a scalar `a·t² + b·t + c = 0`.
- Solve with the **cancellation-safe** formulation (Goldberg 1991) — a grazing segment gives
  `b² ≫ 4ac`, where the naïve quadratic formula loses all precision.
- The root `t* ∈ [0,1]` defines a point that lies **exactly on the ellipse** (the prior arm
  re-anchors to it with zero error) **and exactly on the polyline chord** (the imprint re-anchors to
  it with zero error, since the imprint *is* that polyline downstream). Split both curves there → one
  shared vertex, exact to the imprint's own representation.

The kept region occupies **exactly one** of the four sectors the two boundaries cut at the crossing
(below ∧ outside), so each junction is a **simple degree-2 corner**, not a pinch — the CW
angular-successor walk (de Berg §2.3) closes it cleanly once the vertex is shared.

### Complete loop ingestion

The rod axis spans the full diameter, so the imprint has **two** components — a back **entry
ellipse** (below the notch, a clean full inner hole) and a front **exit ellipse** (crossing the
notch). Both are fed as constraint loops; the back entry ellipse must be ingested as a proper
`Holes` ring of the kept cell. Its collapse in the forced-analytic experiment was exactly a
missing-loop-ingest, not a walk-rule error.

### Tool-side weld guarantee

The rod bite arc is shared between the target's kept face and the tool's tunnel wall; both must split
at the **same** junctions. The tool split already uses `cutCylinderSolidMembership` (cylinder radius
∧ the notch half-spaces), so the same notch plane bounds the tool wall at the same conic — the
junctions coincide by construction. Acceptance is Mäntylä's Euler–Poincaré invariants on the welded
shell, plus the ADR-0046 cross-face weld-orientation contract (every edge shared by two opposite
half-edges).

### Tangency decline gate — scale-invariant by metric normalization

Non-transversal intersections (tangencies, cusps) are **declined observably**, never forced. The gate
is made consistent across cylinders of vastly different radii **not by scaling the threshold** but by
quotienting the radius out of the compared quantity.

- **Two distinct degeneracies, two detectors** — never conflated:

  | Degeneracy | Detector | Meaning |
  |---|---|---|
  | the two **surfaces** tangent | `‖n_target × n_rod‖ → 0` | the SSI itself is singular (Patrikalakis–Maekawa §5–6 tracing degeneracy) |
  | imprint **curve** tangent to prior **curve** | `sinθ_metric → 0` | the corner-junction cusp gated here |

- **Exact tangents, no finite differencing.** The imprint tangent is the analytic SSI tangent
  `T = n_target × n_rod` at the junction (projected into the target tangent plane); the prior-boundary
  tangent is the analytic derivative of the section conic. A chord estimate off the polyline carries
  `O(step·curvature)` error that would floor the trustable angle.

- **Metric-normalized angle.** The cylinder's first fundamental form is `I = diag(R², 1)`
  (`ds² = R²du² + dv²`), so a raw `(u,v)` cross product measures an angle sheared by a factor of `R`.
  Measure the true geometric angle instead:

  ```
  cosθ = (aᵀ I b) / (√(aᵀ I a) · √(bᵀ I b)),   sinθ = √(1 − cos²θ)
  ```

  Implementation: multiply each tangent's `u`-component by `R` before the cross product. `sinθ` is
  then dimensionless and identical on an `R = 0.1` and an `R = 1000` cylinder.

- **Backward-error threshold.** The radius re-enters in exactly one principled place — a resolution
  ratio, not a coefficient:

  ```
  ε_effective = max( ε_ang,  τ / h_local )
  ```

  `ε_ang` = a fixed dimensionless angular floor (a crossing shallower than this is not worth trusting
  even with exact arithmetic); `τ = planeCoef·epsRel·size ≈ 1e-7·size` = the existing model-scaled
  weld tolerance (ADR-0042), **not** a magic `1e-9`; `h_local` = the shortest incident arc length at
  the junction. Two curves crossing at angle `θ` separate at `~sinθ` per unit arc length, so
  `h·sinθ > τ` is the condition for the arrangement to *distinguish* the crossing from a coincidence —
  the sole, correct home for scale sensitivity (Higham's backward-error framing).

- **Bias toward decline.** A false accept ships a wrong solid; a false decline only falls back to
  observable CSG. So `ε_ang` is set slightly high.

## Consequences

- **Buys:** an exact analytic result for the coupled corner-junction — the last declining sub-case of
  the partial-rim / cap-crossing family — and a reusable exact-pre-split overlay seam that the future
  cap-crossing rim-corner and cone sub-cases can share.
- **Costs:** this touches the shared arrangement that all four certified curved-boolean slices depend
  on, so it carries the highest regression risk in the family. It is therefore gated behind the
  byte-identical arrangement golden and engaged only when the exact pre-split fires; the common path
  stays literally unchanged.
- **Definition of Done (first EPIC iteration):**
  1. developed as a separate EPIC, isolated behind the arrangement golden — the common 2-use path
     byte-identical, the pre-split/overlay logic engaged only on a detected crossing;
  2. the tangency gate distinguishes surface–surface singularity (`‖n_target × n_rod‖ → 0`) from
     curve–curve tangency (`sinθ_metric → 0`), and declines the latter observably;
  3. the de Berg walk traverses the degree-2 corners cleanly and ingests the back full ellipse as a
     hole without collapse;
  4. no shared-vertex decision relies on an approximate combinatorial value (EGC invariant);
  5. the newly-classifying config is OCCT-certified (moments + centroid + area + χ + point-membership,
     ADR-0046 protocol) via a two-cut oracle, and every still-declining config (tangency, imprint
     through a prior corner vertex, removed bottom circle) keeps its observable decline with a
     regression fixture.

## Rejected alternatives

- **Method A — post-hoc vertex snapping in the DCEL.** Rejected: snapping near-coincident junction
  vertices relocates the inconsistency downstream — it collapses short edges and can invert tiny
  faces, and its tolerance is unprincipled. Kettner et al. (2008) is the standing evidence that
  snapping is not a fix; it passes tests and ships wrong solids.
- **Method D — full 3D exact B-rep boolean (Nef complexes, Hachenberger–Kettner–Mehlhorn 2007).**
  Rejected: rigorously correct but computationally heavy and it **destroys the pipeline's primary
  advantage** — maintaining exact analytic cylinder/cone surfaces — which the entire #1403/#1724
  recognizer line exists to preserve.
- **Method B — general polygon clipping (Vatti / Greiner–Hormann–Foltz) in place of the arrangement
  walk.** Not rejected outright but **not a substitute for the pre-split**: clipping still requires
  the exact intersection point to be consistent, and classic Greiner–Hormann mishandles the
  vertex-on-edge degeneracy (needs the Foltz / Hormann–Agathos correction). It remains a legitimate
  *cell-extraction engine* on the already-consistent pre-split input; it cannot manufacture the shared
  junction, so it is downstream of Method C, not an alternative to it.
- **Force the analytic path and rely on the internal manifold/volume nets.** Rejected: the forced
  build is manifold-but-geometrically-wrong (right χ achievable, wrong shape), exactly the class
  ADR-0046's certification protocol exists because the internal nets cannot catch. Observable decline
  is correct until the analytic build is OCCT-certified.

## References

- Requicha (1980), *Representations for rigid solids* — regularized set operations / r-sets.
- Hoffmann (1989), *Geometric and Solid Modeling* §3 — boolean via arrangement face-tagging and
  planar conic arrangements.
- de Berg, Cheong, van Kreveld & Overmars (2008), *Computational Geometry* §2, §8 — plane-sweep
  arrangements, subdivision overlay, DCEL face traversal.
- Patrikalakis & Maekawa (2002), *Shape Interrogation for CAD/CAM* §5–6 — quadric SSI and
  tangency/singularity degeneracies; the SSI tangent `T = n₁ × n₂`.
- Yap (1997), *Towards Exact Geometric Computation* — the EGC paradigm (exact combinatorial
  decisions).
- Kettner, Mehlhorn, Pion, Schirra & Yap (2008), *Classroom examples of robustness problems in
  geometric computations*.
- Mäntylä (1988), *An Introduction to Solid Modeling* — Euler–Poincaré validity.
- Greiner & Hormann (1998), *Efficient clipping of arbitrary polygons*, with the Foltz /
  Hormann–Agathos degeneracy corrections — the alternative cell-extraction engine.
- Goldberg (1991), *What every computer scientist should know about floating-point arithmetic* —
  cancellation-safe quadratic.
- Higham (2002), *Accuracy and Stability of Numerical Algorithms* — backward-error framing of the
  resolution threshold.
