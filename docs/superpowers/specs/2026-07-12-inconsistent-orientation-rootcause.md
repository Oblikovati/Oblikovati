# Inconsistent-orientation cluster — root-cause investigation (2026-07-12)

**Method:** systematic-debugging Phase 1–3 with throwaway instrumentation (an `AssembleFilletOrientDiag`
entry returning the invalid body, plus per-edge face/winding dumps — all reverted). Branch
`feat/occt-blend-parity-corpus` @ `8472c805`, baseline scoreboard TOTAL PASS=31.

## The cluster is THREE root causes, not one

The G3 triage's 26 "real build bugs" split by evidence:

### 1. Import defect — 5 cases (NOT a fillet bug): `simple/{T7,U4,F6,T6,U3}`
The **imported body is already invalid before any fillet**: `Validate(importedBody)` reports
`closed=false` with **boundary (open) edges on a solid** (e.g. T7 edges 2240/2257; F6 four open edges,
χ=1). The STEP importer drops a face (leaving an open shell), and the fillet then operates on an
already-open body. Reclassify these to the **STEP-import gap** (same family as the roadmap's
`SURFACE_OF_LINEAR_EXTRUSION`-drop note), not the fillet engine. (F6/T6/U3 were the "empty-reason"
cases; T7/U4 also showed downstream orientation noise, but the open shell is upstream of it.)

### 2. Fillet-introduced inconsistent orientation — ~20 cases (the genuine fillet bug)
`simple/{K6,K9,L1,L3,L4,L6,L7,R9,S1,S3,S4,S6,S7,S9,T1,T3,T4,T9,X3,Y1}` (T7,U4 fold into #1).
Every **pre-fillet** body here has `badEdges=0` (no pre-existing inconsistent orientation) — the fillet
**introduces** it. Two signatures, one cause:
- **A (K/L):** the bad edge is `orig-Plane ↔ fillet:cyl` (a fillet cylinder welded to an original plane).
- **B (R/S/T/U/X/Y):** the bad edge is `orig ↔ orig`, **curved-heavy** (Plane↔Cylinder/Cone/Torus/Sphere/BSpline).

**Mechanism (confirmed in code + by measurement):** every imported body has faces with
`Reversed()=true` (T3: 3 planes + 1 torus reversed; K6: 6 planes). The fillet result has
`Reversed()=false` on **every** face — `transformFace` (fillet_faces.go:154) re-emits
`filletFace{surface: f.Geometry(), …}` and **never carries `f.Reversed()`**; `filletFace` has no
reversed field; `assembleBody` builds all faces unreversed and derives each edge-use direction from
raw loop point order. Decisive per-edge dump (T3 torus↔plane edge):
`IMPORTED: Plane(useRev=true) | Torus(faceRev=true,useRev=false) → consistent` vs
`RESULT: Plane(useRev=false) | Torus(faceRev=false,useRev=false) → INCONSISTENT`.

### 3. Valid-topology-but-not-IsSolid — `tolblend_simple/D5` (+ the #1 overlap)
D5 builds a body that passes `Validate` (0 bad edges) yet `IsSolid()==false` — a distinct
closure/shell issue, one-off.

## Why the two obvious fixes FAIL (this is the important part)

- **Carry `reversed` → `AddReversedFace` only:** NO-OP. `validate.go:47` compares only edge-use
  directions (`uses[0].Reversed()!=uses[1].Reversed()`), ignoring `face.reversed`; scoreboard unchanged
  (PASS=31).
- **Carry `reversed` + blindly re-wind every reversed face's loop:** CATASTROPHIC — simple PASS
  **31→2**. Since the change only touches `f.Reversed()` faces, the ~29 cases it broke **must have
  reversed faces AND passed at baseline**. So dropping the reversed flag is **correct for most reversed
  faces** and wrong for only the 20. The distinction is NOT "has reversed faces."

**Conclusion:** the imported bodies carry a per-face (winding ⊕ reversed-flag) convention that is
internally consistent (passes pre-fillet `Validate`) but that `assembleBody`'s orientation-agnostic
weld preserves for *most* faces and drops for a specific subset (curved reversed survivors in the 20).
A blanket flag-based rule cannot fix it — the correct fix must be **geometric and per-face**.

## Fix direction (for the next increment — well-grounded, not yet built)
Normalize each survivor face's emitted loop to its EFFECTIVE outward normal: compute the loop's Newell
winding normal (`newellNormal` already exists, fillet_faces.go) and the face's effective outward normal
(surface normal, flipped if `f.Reversed()`); **reverse the loop only when they disagree**, and carry the
reversed flag for tessellation. This handles the 20 (need re-wind) and the 29 (already correct → no-op)
uniformly, instead of the blanket reversal that regressed. Verify case-by-case through the scoreboard;
guard the Newell normal against near-degenerate (near-planar-strip) loops with a model-scaled floor.
Open question the fix must settle: does this convention belong in the fillet rebuild (`assembleBody`/
`transformFace`) or upstream as an imported-body orientation-normalization pass? (Candidate for
`software-architect-advisor` — who owns the winding⊕reversed convention.)

## Fix attempts + architecture correction (3 failed — STOP per systematic-debugging)

The architect brief proposed: topo owns a Canonical Orientation Invariant (COI); normalize each minted
face's winding to its effective outward normal; fold `reversed` into `Validate` (ADR-B). Implementing
and MEASURING it disproved two of its own load-bearing claims:

1. **ADR-B (fold `reversed` into Validate) is WRONG.** Edge-use consistency is a pure *winding*
   property (`assemble_curved.go:189` `ec.use`: two faces are consistent iff they emit the shared
   segment in opposite point-order). The `reversed` flag is *orthogonal* — it only flips the surface
   parametric normal for tessellation. Folding it in (`(u0.Reversed ^ f0.reversed) != …`) would
   MIS-REJECT valid imported bodies (imported T3 e34 is consistent under winding-only but fails the
   folded check). The current winding-only `Validate` is correct; do NOT change it.
2. **The COI normalizer's primitive (`newellNormal` vs effective normal) is UNSOUND for the faces
   that matter.** `newellNormal` is a *planar* winding measure, but the ~20 failing faces are exactly
   the CURVED ones (torus/cone/sphere/bspline) whose outer loops are non-planar 3D space curves —
   Newell has no meaningful sign there. Measured: carry-`reversed` + conditional Newell re-wind
   scored simple PASS **31→28** (regressed 3 planar faces on Newell noise, greened 0 of the curved 20).
3. Earlier: carry-`reversed` alone = no-op; blanket re-wind of every reversed face = 31→2.

**What remains genuinely not understood:** WHY `assembleBody` emits a curved reversed survivor's
shared edge in the SAME point-order as its neighbor (producing the inconsistency), when
`transformLoop`+`useFromVertex` appear to emit each face's loop in its stored traversal order (which
for the valid imported body is opposite on shared edges). The direction flip is inside the
weld/`ec.use` interaction and was not isolated. Until that exact flip is understood, any normalizer is
a guess — and Newell is the wrong primitive for curved faces regardless.

**Corrected fix direction (for whoever picks this up):** the orientation test for a trimmed CURVED
face cannot be planar-Newell. It must use the SURFACE's own parametric orientation along the boundary
(e.g. at a boundary point, sign of `(∂S/∂u × ∂S/∂v) · (edgeTangent × inwardOffset)`, or integrate the
loop's turning against the surface normal field) — a `geometry-math-advisor` question. OR: preserve
the imported edge-use directions through the re-weld directly (thread each segment's original
`use.Reversed()` into `filletLoop` and have `ec.use` honor it) instead of re-deriving from geometry —
which sidesteps winding entirely and may be the smaller, safer change. Both are unverified.

## DECISIVE REFRAME (targeted ec.use probe) — it was NEVER the reversed flag

A parity probe on T3 (dump the loop-use order + degeneracy of every inconsistent edge) overturned the
whole orientation theory. The `reversed`-flag correlation was a **red herring** (periodic curved faces
happen to be imported reversed). Classifying all 99 inconsistent edges across the 20 cases:

| sub-cluster | cases | edges | true root cause |
|---|---|---|---|
| **B1 seam/pole** | R9,S1,S3,S4,S6,S7,S9,T1,T3,T4,T9,X3,Y1 (13) | 33 (**100% DEGENERATE**, start==end) | periodic-surface **seam/pole edges** collapse to zero-length self-loops in the re-weld; `ec.use` cannot orient a self-loop (`canon2(a,a)`, `rec.from != a` always false → both uses same direction). Faces: Cyl/Cone/Torus/Sphere/BSpline ↔ Plane. |
| **B2 fillet-face winding** | K6,K9,L1,L3,L4,L6,L7 (7) | 66 (**0% degenerate**) | the fillet's OWN generated `cylinder` + `sphere-patch` faces wind inconsistently with planes/each-other (`Plane\|Cylinder`, `Cylinder\|Sphere`, `Cylinder\|Cylinder`). A fillet-face winding issue, distinct from B1. |

Evidence (T3 imported reversed Torus face 41 outer loop): `use[1] from=(27.46,3.93,23.57)
to=(27.46,3.93,23.57)` — a degenerate seam use; the RESULT inconsistent edges e85/e90 have
start==end. Both B1 and B2 defeat "approach (b)" (thread `use.Reversed()` through the weld): a
zero-length self-loop has no meaningful direction (B1), and B2's faces are op-generated (no source
use). So the corrected fix direction earlier in this doc is ALSO wrong; the real work is two separate
hard kernel-topology problems:
- **B1:** periodic-surface seam/pole handling in `assembleBody` — either don't collapse the seam's two
  endpoints (keep the seam edge non-degenerate via UV/param distinction), or drop zero-length edges and
  represent the pole as a single vertex. Needs `geometry-math-advisor` (periodic-surface topology).
- **B2:** the fillet cylinder/sphere-patch face winding (`cylinderFace`/`spherePatchFace`) vs its
  neighbours on the K/L box family — a fillet-engine winding audit.

**Net:** the "one fix unlocks many" premise is dead — this is 5 import defects + 1 IsSolid one-off + 13
periodic-seam + 7 fillet-winding, four unrelated causes. Recommend a deliberate scope decision before
any further fix (each of B1/B2 is its own increment), rather than a fourth patch.

## Recount after reclassification
Of the 26: **5 → STEP-import gap**, **1 (D5) → IsSolid one-off**, **20 → the genuine
orientation fix** above. So the "one fix unlocks many" is ~20 cases (still a strong cluster), plus 5
that belong to the import backlog and shouldn't be chased as fillet bugs.
