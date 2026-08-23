# ADR-0052 — Co-refined shared-seam classification for the planar boolean (near-tangent grazing seams)

**Status:** Proposed (2026-08-23). · **Scopes** the architectural fix for
[Oblikovati#2084](https://github.com/Oblikovati/Oblikovati/issues/2084) (a coil/core join tears when
the coil mesh is refined) and unblocks
[Oblikovati#2081](https://github.com/Oblikovati/Oblikovati/issues/2081) (the finer centre-fan
tessellation that trips the same failure). · **Builds on**
[ADR-0047](ADR-0047-planar-boolean-radial-edge-sew.md) (radial-edge tangent sew) and
[ADR-0043](ADR-0043-generalized-provenance-naming.md) (edge/vertex provenance). · **Touches (planned):**
`kernel/brep/boolean.go`, `boolean_split.go`, `boolean_classify.go`, `boolean_stitch.go`,
`boolean_provenance.go`. · **Does NOT supersede** the `maxFacetWarpRatio = 0.03` lower bound in
`model/feature/swept_refine.go`, which remains the shipped mitigation until Phase 3 lands.

## Context

The planar B-rep boolean unions/cuts/intersects two **planar-faceted** solids. For clean transversal
crossings — the entire validated corpus (`kernel/brep/*_test.go`, `kernel/ops/boolean_*_test.go`) —
it is exact and sound. It **tears** when two curved-but-tessellated surfaces meet along a
**near-tangent grazing seam**: the #2084 case is a round-wire coil (a swept tube) joined to a faceted
cylinder core. Refining the coil mesh past `maxFacetWarpRatio ≈ 0.0245` produces thousands of open
edges (`valid=true, closed=false`); the shipped `0.03` lower bound exists only to keep the coil facets
coarse enough to hide it.

### Evidence (instrumented, on a 4.6 s replay of the captured failing operands)

The failure was investigated stage-by-stage; the mechanism is fully pinned and **four intuitive
explanations were falsified by experiment**, each with a one-line probe:

| Suspected locus | Probe | Result |
| --- | --- | --- |
| Imprint boundary-filter (`interiorSegments`) | keep boundary-coincident segments on both faces | no change → not it |
| Vertex weld (`welder3`, #879 family) | nearest other vertex to every open-edge endpoint | 0 within 1e-5 → not it |
| Stitch T-junction tolerance (`collectSegHits`) | widen perpendicular tol 1e-6 → 1e-4 (100×) | byte-identical → not it |
| Winding classification (`solidProbe.inside`) | winding margin of every kept/dropped sub-face, both operands | all clear (>0.05 rad), 0 near the 0.5 knife-edge → classification is correct |

What the evidence positively shows (failing pitch 0.30):

- **Imprint is symmetric and complete.** Core receives 2283 seam segments, coil 2264 — essentially
  equal. Every one of the 930 unpaired coil seam edges **has a matching core imprint segment**
  (`noCoreImprint = 0`). The atomic intersection segment `s{Fj,Ci} = Fj-plane ∩ Ci-plane` lies on
  both operands identically.
- **The arrangement keeps the seam.** Core facets: 2283 input segments → 2544 surviving
  region-boundary edges; nothing is discarded as dangling.
- **The tear is a cross-operand *classification inconsistency*.** Open edges split into
  `{used-once coil seam edges}` and `{used-3× a coil edge exactly coincident with a retained core
  edge}`. **682 of the 930 unpaired coil seam edges coincide with an edge bordering a core region
  that was *dropped*** (classified inside the coil), and **no kept core edge is collinear** (nearest
  2.5e-2 away). The coil face is legitimately kept (its sample point is outside the core) while the
  co-located core region is legitimately dropped (its sample point is inside the coil): there is **no
  shared kept boundary to weld**, so the kept coil face protrudes into removed core material.

### Root cause

The round wire grazes the faceted core at a shallow angle, producing a thin **interpenetration
lens**. Each operand is arranged **and classified against the other operand's facets independently**.
For a transversal crossing the two independent kept/dropped boundaries coincide (same atomic seam
segments), so they weld. At a grazing seam the two independent classifications disagree at facet
resolution — the coil's kept boundary and the core's kept boundary diverge by up to a facet width —
and the stitch has no coincident geometry to reconcile. It is coincidence-driven: the coil facet
phase only lines up to produce the divergence at particular pitches (0.30 breaks; 0.25 and 0.20 are
clean).

### Why the existing fallback does not save it

`ops.booleanGeneral` already declines an invalid planar result to triangle-soup BSP CSG
(`shouldFallbackBoolean` → `booleanCSG`), but only for operands under `csgFallbackFaceLimit = 256`
faces (perf). **Raising the limit was tested** (limit → 1e6, warp budget → 0.02): the coil-join was
**still open (2551 edges)** and the run took **6 minutes** — BSP CSG **also fails to close** the
near-tangent grazing seam, and is far too slow at 24 578 faces. The CSG fallback is therefore **not**
a viable fix for this class. This is a property of the geometry (two smooth surfaces faceted near
tangency), not of any one boolean engine.

## Decision (proposed)

**The seam is already shared.** `pairImprints` computes `imprint(a, b)` **once** and hands the
identical `Point3` values to both operands (`interiorSegments(a, segs)`, `interiorSegments(b, segs)`
over the same `segs`), so the atomic seam segments `s{Fj,Ci}` and their endpoints are **bit-identical**
on the core and the coil. Sharing seam *geometry* is therefore not the problem, and no cheap
"weld the seam across operands" step exists — it would be a no-op.

The defect is **not a classification error that a better classifier can repair.** `booleanOnce` runs
`selectFaces(fa, …)` and `selectFaces(fb, …)` as two independent passes, each labelling its faces
inside/outside the other operand. The evidence shows **both labellings are individually correct and
unambiguous** — every kept/dropped winding margin, on both operands, is clear (>0.05 rad). At the
grazing lens "coil face is outside the core" and "the co-located core region is inside the coil" are
*both true* for the faceted inputs, yet they describe overlapping space with no shared boundary: the
faceted union is **genuinely non-manifold there**. A lighter re-classification (e.g. seam-adjacency
flood propagation) therefore **cannot help** — it would reproduce the same per-operand labels, which
are already correct. The fix must change how the output *surface* is derived, so it is manifold **by
construction**. Two tracks:

### Track A — Unified cell-arrangement extraction (general fix, planar path)

Adopt the mesh-arrangement discipline (Zhou et al. 2016, *Mesh Arrangements for Solid Geometry*;
libigl `mesh_boolean`; Cork). Do not classify each operand against the other and weld the survivors.
Instead, from the merged co-refined complex of **all** sub-faces from both operands (seam vertices are
already shared), build the arrangement of 3-space **cells**, label each cell by both operands' winding
numbers, and emit the output surface as exactly the faces **between cells whose in/out status differs**
per the operation. This is manifold by construction — there is a single cell complex, never two
independently-made verdicts to disagree — which is precisely why libigl/Cork are robust on
near-tangent tessellated inputs. It is effectively a **new boolean core** and the largest change here;
it retires the per-sub-face `classifySubFace` while preserving the `keep`/`coplanarKeep` semantics,
ADR-0043 provenance, and the ADR-0047 radial-edge sew as the manifold-extraction step.

### Track B — Route curved-on-curved through the exact seam (targeted, aligned with ADR-0045)

The coil (swept round wire) and core (cylinder) are **curved** surfaces forced through the faceted
path only because the sweep tessellates them. Extending `curvedExactPaths`
([ADR-0045](ADR-0045-curved-boolean-kind-taxonomy.md)) to compute the exact swept-surface ∩ cylinder
seam (SSI) removes the faceted grazing entirely for this family — it dissolves the whole class of
"curved thing joined to curved thing," not just coil/core.

**Recommendation:** **Track A (unified cell arrangement)** is the general robustness fix and hardens
every future near-tangent faceted case; **Track B** is the higher-leverage targeted fix for thread
families where analytic surfaces survive to the boolean. Both are large; pick per appetite (Track B is
narrower and reuses the existing curved SSI machinery).

## Phased execution plan (each phase gated on the FULL boolean corpus + the 4.6 s repro)

- **Phase 0 — DONE.** Root cause pinned; four loci falsified; the CSG shortcut proven dead; the seam
  confirmed already-shared; **and it is established that no re-classification can fix it** (both
  per-operand labellings are already correct). A fast operand-replay repro harness was built (capture
  the two operands to OBJ at the failing union, reload via `subd.ToBody`, replay `brep.Boolean` —
  4.6 s). Recorded on #2084 and in the session notes.
- **Phase 1 — Track A cell-arrangement core (or Track B curved SSI).** A **new boolean core** (or a
  new `curvedExactPaths` entry). **There is no corpus-safe partial step** and no shortcut: it lands as
  one reviewed unit on its own branch, gated on the full `kernel/brep` + `kernel/ops` boolean corpus
  with per-case volume/manifold assertions, with wholesale revert as the safety valve.
- **Phase 2 — Remove the mitigation.** Once the repro is watertight at `maxFacetWarpRatio = 0.02` and
  #2081's centre fan, lower/retire the `swept_refine.go` lower bound and run
  `TestCoilJoinFinePitchWatertight` at the finer budget.

Phase 1 is a **dedicated multi-session effort**, not a batch drive-by: the honest scope is a
from-scratch boolean core plus a full corpus-gating cycle. Landing a rushed or partial version would
violate the corpus-safety invariant this ADR itself establishes, so it is deliberately **not**
attempted inside the current sheet-metal batch.

## Consequences

- **Corpus safety is the gating invariant.** No phase lands unless the entire
  `kernel/brep` + `kernel/ops` boolean corpus stays green; a single regression reverts the phase. The
  planar boolean is load-bearing for booleans, features, and exports, so speculative changes are not
  acceptable — this is why the fix is phased rather than a single rewrite.
- **The mitigation stays until Phase 2.** `maxFacetWarpRatio = 0.03` remains the shipped guard; it is
  a correct engineering trade (keeps the grazing lens below arrangement resolution), not a hack.
- **Provenance (ADR-0043) and the radial-edge sew (ADR-0047) are preserved.** Track A changes how the
  output surface is *derived* (cell boundaries of one merged complex), not how intersection
  edges/vertices are named or how the radial-edge sew extracts the manifold.

## Alternatives considered and rejected

- **A better/consistent per-operand classifier (e.g. seam-adjacency flood).** Cannot help: both
  per-operand labellings are already correct and unambiguous (all winding margins clear); the
  non-manifoldness is in the faceted geometry, not in a mislabel, so re-labelling reproduces the tear.
- **Widen a tolerance (weld grid / T-junction perpendicular).** Falsified: 100× wider changes
  nothing; the counterpart geometry genuinely is not there to weld.
- **Keep/drop the offending faces post-hoc in the stitch.** Symptom-patching; creates holes or
  spurious walls and cannot restore a consistent seam.
- **Rely on the CSG fallback for large operands.** Proven dead: BSP CSG also fails to close this
  geometry and is prohibitively slow at coil scale.
- **Only raise `maxFacetWarpRatio`'s floor further.** Caps achievable tessellation quality and still
  blocks #2081; treats the symptom.
