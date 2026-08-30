# ADR-0058 — The one boolean is the tolerant ANALYTIC B-rep engine (OCCT/Parasolid class); reconstruction is demoted to curved-face recovery + rescue

**Status:** Accepted — on `m48/kernel-ground-rules`. · **Scopes**
[Oblikovati#2247](https://github.com/Oblikovati/Oblikovati/issues/2247) (the "one boolean" goal) and the
M48/C2 boolean cluster (#2248, #2250, #2251, #2252), reframing them. · **Supersedes**
[ADR-0057](ADR-0057-reconstruction-canonical-boolean.md) ("reconstruction is the one boolean;
`brep.Boolean` is deleted"): that direction is **reversed** on performance grounds established below.
**Re-scopes** [ADR-0056](ADR-0056-analytic-face-reconstruction-boolean.md) and
[ADR-0052](ADR-0052-planar-boolean-corefined-seam-classification.md): the exact mesh arrangement returns
to being a *curved-analytic-face recovery* and *last-resort rescue*, not the universal engine. **Builds
on** [ADR-0043](ADR-0043-generalized-provenance-naming.md) (naming) and
[ADR-0045](ADR-0045-curved-boolean-kind-taxonomy.md) (boolean kinds). · **Does NOT delete `brep.Boolean`** — it
**promotes** it to the planar core of the general engine. · **Deletes when complete:** the
`analyticFaceCount==0` framing as a *carve-out* (it becomes a *fast-path selector*), and the exact
per-operation coincidence hacks the mesh path accreted (the coincident-opposite snap and the radial-edge
sew as *ops/reconstruction-layer* code — their behaviour moves into one tolerant fusion pass). · **Touches:**
`kernel/geom` (a general surface–surface intersector + `SameDomain` tolerant fusion), `kernel/brep`
(generalize the imprint→split→classify→stitch to curved intersection curves), `kernel/ops/boolean.go`
(dispatch), `kernel/ops/meshbool_recovery.go` (reconstruction as recovery/rescue).

## Context

ADR-0057 set out to make the exact mesh-arrangement reconstruction (`big.Rat` tessellation → arrangement
→ analytic rebuild, ADR-0052/0056) the ONE boolean, deleting the planar B-rep engine `brep.Boolean`. On
`m48/kernel-ground-rules` the correctness work for that cutover was completed and validated (2026-08-29):
coincident-opposite plane snap (3530e5d8), planar-face outward canonicalization (e3b2ab88), collinear
T-junction dissolve (5aeb26d2), a radial-edge tangent sew ported into the reconstruction stitch, and a
"self-sufficient" merged-coplanar fallback that returns `soupToBody(MergeFaces)` instead of triangle soup.
With those, the full `model/feature` suite passes gate-flipped.

**But the cutover is not performance-viable, and the reason is fundamental.** Routing every planar boolean
through the exact `big.Rat` mesh arrangement is ~1.5× slower than the float planar B-rep engine (feature
suite 233s → 347s) AND does not scale: on a large/complex operand (measured: a tapered-shaft rim chamfer)
the reconstruction path hangs past the test timeout. Deleting `brep.Boolean`'s clean low-face-count result
also re-introduced a **faceting explosion** (a decline → triangle-soup CSG → the next boolean declines on
the soup → soup-on-soup → 1.7M faces → hang), because `brep.Boolean` was load-bearing as both the fast path
AND the clean decline-fallback. The cost is inherent: **exactness is the source of both the mesh path's
robustness and its cost.**

**How the reference kernels solve it (OCCT survey, 2026-08-29).** OpenCASCADE's boolean (`BOPAlgo`) — like
Parasolid and ACIS — is a **tolerant ANALYTIC B-rep engine**, not a mesh arrangement and not exact
arithmetic:

- *Analytic, no mesh.* `BOPAlgo_PaveFiller` intersects sub-shapes pairwise in dimension order (VV→VE→VF
  →EE→EF→**FF**); FF is `IntTools_FaceFace` → `GeomInt_IntSS`/`IntPatch`/`IntSurf` — surface–surface
  intersection producing analytic curves. Triangulation appears only as an incidental helper (seam-edge
  detection), never for the intersection. All arithmetic is `double` + tolerances.
- *Conditioning-gated fast paths (the ground rule, done right).* `IntTools_FaceFace` uses closed forms
  (`PerformPlanes`, quadric–quadric) but **demotes on conditioning, not type**: an eccentric conic
  (`aMajorR < 100000·aMinorR`), near-degenerate cones/tori, or an `ImpImp` that is `!IsDone()` all fall to
  the general **walking-line marcher** (`IntPatch_PrmPrmIntersection`).
- *Tolerant `SameDomain` fusion.* Coincident vertices/edges/faces within summed tolerance (`FuzzyValue`
  slack) are **physically fused into one shared entity** (`MakeSDVertices`+`AddShapeSD`; `BOPDS_CommonBlock`;
  `AreFacesSameDomain`). Tangent/grazing contacts are flagged (`TangentFaces`/`OppositeFaces`), fused, and
  the result is assembled by **point-in-solid classification** (`BRepClass3d_SolidClassifier`) of split
  shapes — a valid manifold/multi-shell without any exact predicate. A `Glue` mode skips the FF
  intersection entirely for known-coincident operands.

Oblikovati **already has this engine's shape**: `brep.Boolean` IS the planar imprint→split→classify→stitch
pipeline (the planar `BOPAlgo`), and `curvedExactPaths` are the closed-form quadric intersectors. It is not
yet *general* (curved). Meanwhile the bespoke coincidence machinery the mesh path grew — the coincident-
opposite snap and the radial-edge sew — are hand-built, exact-arithmetic reinventions of OCCT's one
tolerant `SameDomain` fusion.

**On the exactness ground rule.** CLAUDE.md mandates exact/filtered predicates for topological decisions,
which is what drove the mesh choice. But the same rules already say **"classify a comparison by the origin
of its operands: same computation → `Weld()`, independent sources → `Sew()`"** and **"coincidence is
transitive; tolerances only tighten as operations compose."** Cross-operand coincidence in a boolean is,
by definition, *independent sources* — a `Sew()`/tolerance decision, exactly OCCT's `SameDomain`. So the
tolerant-analytic direction is not a repudiation of the ground rules; it is the correct reading of the
`Sew`-for-independent-sources rule, with exact/filtered predicates retained for *same-computation*
decisions (the tessellator, self-intersection, orientation of one operand's own geometry).

## Decision

**The one boolean is a tolerant, analytic B-rep engine, OCCT/Parasolid class.** Concretely:

1. **`brep.Boolean` is the planar core of the general engine, kept and promoted** — not deleted. ADR-0057's
   deletion is reversed.

2. **Generalize the analytic engine to curved** via a *general surface–surface intersector* on
   `geom.Surface` (`SSI`), dispatched by surface-pair type to closed forms and **demoted on conditioning**
   (an ill-conditioned closed form falls to a general parametric walking-line marcher). This is a method on
   `geom.Surface` (per the ground rule "behaviour many operations need … is a method on `geom.Surface`"),
   extending today's partial `geom.IntersectSurfacesAnalytic`. Intersection curves split edges into
   pave-block-style segments; the existing brep split/classify/stitch consumes them.

3. **Replace the per-operation exact coincidence hacks with ONE tolerant `SameDomain` fusion pass** on the
   analytic B-rep: near-coincident vertices/edges/faces (independent-source `Sew()` tolerance from
   `geom.Resolution`, with a fuzzy override) fuse into one shared entity before classification. The
   coincident-opposite snap and the radial-edge sew are subsumed by it. Tangent/coincident contacts are
   detected here and the result stays a valid manifold/multi-shell by **classification**, not exact math.

4. **The exact mesh arrangement is demoted** from "the one boolean" to two roles it is genuinely good at:
   (a) recovering an **analytic curved face** where the general engine had to facet (#2153's original
   purpose), and (b) a **last-resort robustness rescue** for a case the analytic engine declines. It also
   stays as an **oracle** for tests. It is never the universal path; the `analyticFaceCount==0` decision
   becomes a *fast-path selector* refined by conditioning, not a carve-out to be deleted.

5. **Dispatch is one classification** (per the ground rule): planar-conditioned → planar core; curved →
   general SSI engine; ill-conditioned surface pair → SSI marcher; genuinely unhandled → named decline →
   mesh rescue. No ordered try-list.

## Consequences

- **Fast + robust + one conceptual engine.** The common planar and well-conditioned curved cases run
  analytically (fast), coincidence/tangent are one tolerant fusion, and only the pathological remainder
  touches the exact mesh — matching how every production kernel is built.
- **Exactness scope narrows, deliberately.** Cross-operand incidence becomes a managed-tolerance (`Sew`/
  `SameDomain`) decision. This is a *scoped* change to the exact-predicate rule for the boolean incidence
  layer, justified above and consistent with the `Sew`-for-independent-sources rule; it must be reflected
  in the CLAUDE.md kernel ground-rules block (the "every topological decision is exact" line gains the
  boolean-incidence exception). Same-computation and single-operand decisions stay exact/filtered.
- **Deletions come at the end** (per "a generalization is complete only when the special cases it
  replaces are deleted"): once the general SSI + `SameDomain` engine passes the curved-boolean corpus AND
  OCCT parity, the bespoke `snapCoincidentPlanes`, the reconstruction-layer radial sew, and the
  `curvedExactPaths` N² closed-form ladder collapse into the general engine's conditioning-gated dispatch.
- **Reconstruction keeps its wins.** ADR-0043 multi-parent naming, the analytic-face recovery, and the
  mesh oracle remain; only its universality claim is withdrawn.
- **Net engine count is one** (a general analytic boolean with a mesh *rescue*, not a co-equal second
  engine) — satisfying the "one general pipeline per operation" rule the honest way, at speed.

## Implementation phases (each lands green; corpus + OCCT parity gate the deletions)

0. **This ADR.** Keep `brep.Boolean`; restore gate ON; record the direction. (The 4 correctness commits
   already on the branch stand.)
1. **General `geom.Surface` SSI** — a `SurfaceIntersect(other, res) ([]Curve3, kind)` method: closed forms
   for the quadric pairs (reuse `curvedExactPaths` math), a general parametric **walking-line marcher** for
   the rest, **conditioning-gated** (eccentric conic / degenerate cone-torus / closed-form `!ok` → marcher).
   Gate: OCCT-parity SSI corpus.
2. **Tolerant `SameDomain` fusion** — one pass over the analytic B-rep: cluster and fuse coincident
   vertices/edges/faces at `Sew()` tolerance (+ fuzzy), transitive, tolerances only tighten. Replaces the
   coincident-opposite snap and the radial-edge sew. Gate: the coincident-face + tangent + bowtie corpus
   (the current brep `boolean_sew_golden`/`boolean_tangent`/`boolean_pinch` tests, driven through the
   analytic engine).
3. **Curved imprint/split/classify/stitch** — feed the SSI curves into the brep pave-block edge split and
   the classify/stitch, so a curved boolean produces an analytic result directly (no mesh). Gate: the
   `kernel/ops` curved-boolean corpus (ball-stud, drilled-plate, cone-box, Steinmetz, …) with volumes and
   analytic-face counts against the OCCT oracle.
4. **Dispatch + demote reconstruction** — one classifier selects planar-core / SSI-engine / marcher; the
   mesh arrangement becomes analytic-face recovery + rescue + oracle. Delete the subsumed special cases
   and update the CLAUDE.md ground-rule block. Gate: full kernel + feature suites at NO perf regression vs
   gate-ON baseline (the 233s number), and the OCCT boolean parity corpus.

Progress and the perf/corpus evidence live on #2247; this ADR is superseded by a new ADR, never edited in
place.
