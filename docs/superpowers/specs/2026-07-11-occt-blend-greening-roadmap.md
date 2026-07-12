# OCCT Blend Parity — Greening Roadmap

**Status:** backlog + sequencing (not an implementation plan). Each work package below becomes
its own spec → plan, grounded in **ADR-0050** (OCCT-parity blend engine) and routed through
**`geometry-math-advisor`** for the hard geometry. The `occtparity` corpus
(`model/feature/occtparity/`) is the gate: a package is *done* when its listed cases turn green
and stay green. The single milestone PR lands only when the whole corpus is green (excluding
OCCT-TODO skips) — per the user's "no PR until all pass" rule.

**Baseline (2026-07-11, branch `feat/occt-blend-parity-corpus`):** 475 cases →
**13 PASS / 238 FAIL / 224 SKIP**. The two harness-surfaced bugs (imported-solid normal
orientation, seam-fragile locator) are already fixed. The 238 reds below are genuine engine
gaps. Counts are from the failure messages; a case can shift buckets once diagnosed.

## The backlog, by engine gap

| # | Package | Cases | ADR-0050 | Root cause (one line) |
|---|---------|------:|----------|-----------------------|
| G1 | **Area-parity bugs** | CLOSED | DONE | **Was 12 → 0 G1 residuals.** All fixed: 1b box shared-corner (`P8,V8`, corner-solve orientation); apex sextet (`A9,B4,B8,C3,D2,D6`) + `Q1` (curved survivor-edge straightening — one fix, see note); `Y2` (conformance-repair guard, Bug A). Only `W2,H6` remain and are **true G6** (fillet ON an arc edge bordering a cylinder), not G1 |
| G2 | **Radius > max / degenerate** | 19→**14 core + 5 split** | P2 | **Triaged 2026-07-12 (systematic-debugging): three families, not one.** (a) **~14 genuine max-width bug** — `availableWidth` samples the fillet edge *including its endpoints*; at an endpoint an adjacent boundary edge shares that vertex, so its ray crossing is at x=0 but **float roundoff makes it x≈1e-14**, slipping past the `x<=0` guard and poisoning the min width to ~0 → `rMax≈0` → every real radius rejected. Cases: `complex/E3`, `simple/{N5,R7,T3,V3,V5,X2,X8}`, `tolblend_simple/{A9,B1,B2,E6,E7,E8}`. **This is the "one fix unlocks many" cluster.** (b) **G3,G4,G6 → STEP-import gap, NOT fillet**: importer drops the `SURFACE_OF_LINEAR_EXTRUSION` face (B-spline profile), leaving an open 3-face shell with degenerate 2-edge coincident-edge lunes; guard then *correctly* reads ~0 width. Reassign to a STEP-import work item. (c) **complex/D6, simple/Q6 → G6**: "arc radius > cylinder radius" — curved-neighbour large-radius, import is clean |
| G3 | **Generic invalid-solid (triage)** | 30 | — | Fillet ran but the assembled solid is invalid with no specific reason; diagnose → reassign to G4/G5/G6 or a real assembly bug. **−4:** `A2,A6,K7,W1` were the same corner-solve orientation bug as G1 1b and turned green with that fix |
| G4 | **Miter blends** (2 convex edges meet) | 61 | P6 | Two filleted edges meet and the corner/outer face is non-planar, or no outer face exists — miter reconstruction gap |
| G5 | **Corner blends** (vertex convergence) | 54 | P6 | n-way / trihedral vertices, mixed radius, arc-end-not-tangent, endpoint-no-end-face — corner reconstruction (IntersectionAtEnd, n-way). (The 6 apex cases briefly parked here were NOT corner reconstruction — they were the curved survivor-edge bug, now FIXED in G1; see note.) |
| G6 | **Curved-neighbour + fillet-into-fillet** | 53 | P4/P6 | Filleting an edge adjacent to a cylinder/cone/sphere/bspline face, or running into an existing round (#1797). **+2 from G1 1c:** `W2,H6` (arc edge bordering a cylinder face) |
| G7 | **Topology edge≠2-planar + misc** | 5+ | — | Edge bounds ≠ 2 planar faces; a few one-offs |
| G8 | **Variable radius (buildevol)** | 108* | P?/M44 | *SKIP today.* Feature API exists; blocker is mapping OCCT's edge-parameter `updatevol` law to our reparameterized edge (arc-length) |

\* G8's 108 are currently `SKIP(varradius)`, not fails — but they are coverage that must be
turned on (correctly) for true completeness.

## Recommended sequence (rationale: correctness & quick wins first, hardest IP last)

**Phase A — independent quick investigations (do first, in parallel):**

- **G1 Area-parity bugs (12).** Highest value/lowest effort: these fillets already *build*, so
  the defect is subtle geometry (wrong shoulder, off-by tolerance, tessellation-vs-analytic
  area). Each is a focused numeric investigation and a strong ADR-0050 grounding exercise.
  Cases: `simple/{A9,B4,B8,C3,D2,D6,H6,P8,Q1,V8,W2,Y2}`.
- **G2 Radius>max / degenerate (19).** The near-zero max-width strongly suggests a single
  computation bug (probably the in-face available-width at a specific dihedral/scale). Diagnose
  one, likely fix many. Cases: `complex/{D6,E3}`, `simple/{G3,G4,G6,N5,Q6,R7,T3,V3,V5,X2,X8}`,
  `tolblend_simple/{A9,B1,B2,E6,E7,E8}`.
- **G3 Generic invalid-solid triage (34).** Not a fix package yet — a diagnosis pass that runs
  each case, captures *why* the solid is invalid, and reassigns it to G4/G5/G6 or flags a real
  assembly bug. Do before committing effort to G4–G6 so their scope is accurate. Cases:
  `simple/{A2,A6,A8,F6,K6,K7,K9,L1,L3,L4,L6,L7,R9,S1,S3,S4,S6,S7,S9,T1,T4,T6,T7,T9,U3,U4,V1,W1,X3,X9,Y1,Y3}`,
  `tolblend_simple/{C4,D5}`.

**Phase B — the corner-reconstruction engine (the core ADR-0050 Phase 6 IP):**

- **G4 Miter blends (61).** The largest single feature gap and the simpler corner case (exactly
  two edges). Build the miter reconstruction for non-planar shared/outer faces (and the
  "no outer face" degenerate). Sub-buckets: shared-face-not-planar (31), outer-face-not-planar
  (22), no-outer-face (8). This is the natural first corner-machinery slice.
- **G5 Corner blends (54).** Generalizes G4 to vertices where ≥3 fillets converge (n-way /
  trihedral), mixed radius at a corner, arc-end tangency, and endpoint-no-end-face. Depends on
  G4's machinery. This is OCCT's `PerformN Corner` / `IntersectionAtEnd` territory — the hard IP;
  route through `geometry-math-advisor` before implementing.

**Phase C — curved neighbours & the rest:**

- **G6 Curved-neighbour + fillet-into-fillet (51).** Fillet an edge adjacent to a curved face,
  and fillet-into-fillet (#1797, currently honest-rejected). Leans on the curved-boolean
  machinery. Sub-buckets: not-supported (35), fillet-into-fillet (16).
- **G7 Topology + misc (5+).** `simple/{F4,G5,G7,G9}`, `tolblend_simple/C8`, and the handful of
  one-offs surfaced by G3.
- **G8 Variable radius (108).** Two parts: (a) the oracle emits each `updatevol` law point's
  arc-length fraction (parameterization-invariant, per the corpus's centroid rule), (b) the
  runner maps it onto the feature's `RadiusPoint.T` after confirming that field's semantics.
  Then the buildevol grid measures real variable-radius parity.

## The "apex-edge" cases were the curved survivor-edge bug — FIXED (2026-07-12)

The 6 "apex" cases (`A9,B4,B8,C3,D2,D6`) were briefly parked in G5 as suspected revolution-pole
corner reconstruction. That diagnosis was **wrong**. Chasing the separate Q1 residual surfaced the
real, shared root cause and a single fix greened all seven (kernel test
`kernel/ops/fillet_apex_diagnosis_test.go`, now green; fix `kernel/ops/fillet_faces.go`
`survivorCurve` + `kernel/ops/fillet_survivor_curve_test.go`):

- **Root cause:** `transformLoop` rebuilt a filleted face's loop adding every non-corner
  ("survivor") vertex with a **nil curve**, dropping the geometry of any curved boundary edge that
  survived the fillet. Because both faces sharing such an edge are transformed, the shared **arc
  collapsed to a straight chord** (the edge catalog makes a `LineSegment` when neither side supplies
  a curve). A partial primitive's radial cut faces border the quadric along arc edges, so the sector
  deformed — the "apex" symptom (A9/B4 both 19098.9, off by 10–57%). It also inflated any planar face
  bordering a cylinder — the Q1 +3.4%.
- **The fix** carries an arc survivor edge's geometry through the rebuild, **oriented to the loop
  traversal** (a reversed use needs the reversed arc, or the two symmetric end caps come out
  different — one right, one bulged; that orientation subtlety is why the first attempt made Q1
  worse before the reversal was added). No-op for straight edges. Greened `A9,B4,B8,C3,D2,D6,Q1`;
  **+7 corpus PASS (20→27), 0 regressions** (stash-diff of the pass set). This is why M1 always
  passed: its shared arcs were oriented such that straightening barely moved the area.

## G1 1b/1c outcome (2026-07-12)

Implemented from plan `docs/superpowers/plans/2026-07-12-g1-area-parity-bugs.md`; kernel
regression `kernel/ops/fillet_box_corner_test.go`, triage `model/feature/occtparity/fillet_1c_triage_test.go`.

- **1b box shared-corner (`P8,V8`) — FIXED** (commit on `feat/occt-blend-parity-corpus`). Root
  cause was NOT a corner double-count (the plan's guess) but a corner-solve **orientation** bug:
  `fillet_miter.go:planeNormal` and `fillet.go:solveBlend` read raw geometric plane normals,
  ignoring `face.Reversed()`. On STEP-imported solids (inward plane normals) one miter arm's frame
  flipped — cylinder overshot, shared face under-trimmed → +3.4%. A native `brep.SolidBlock` box
  filleted correctly (145.126 vs OCCT 145.137); only the imported box failed. Fix routes both
  corner sites through the existing `outwardPlaneNormal` helper — the corner-path analogue of the
  earlier `edgePlanarFaces` fix (dbd28339). **Corpus: +6 PASS (13→19), 0 regressions**: greened
  `P8,V8` plus `A2,A6,K7,W1` (same root cause, were mis-filed under G3). Real user-facing bug:
  corner fillets on every imported STEP solid were wrong.
- **1c triage (`W2,H6,Y2,Q1`):**
  - `W2,H6` → **G6**: picked edge is a `geom.Arc3d` bordering a `geom.Cylinder` — curved-neighbour
    blends, not planar defects.
  - `Y2` → **two stacked bugs, both handled.** Symptom was `ops.BodyGeometryProperties`
    under-measuring the assembled body at 53337 (−12.63%) though the per-face tessellation sum is
    61147 (within 0.16% of OCCT 61050). **Bug A (FIXED):** conformance-repair's boundary-faithful
    CDT collapsed a face on a self-intersecting boundary (8475→675); it now bails on a non-simple
    loop and keeps the robust earclip mesh (`kernel/ops/conformance_repair.go`, guard `simpleLoop2D`;
    regression `fillet_conformance_notch_test.go`). Y2 now PASSES the gate (61147, +0.16%).
    **Bug B (TRACKED known-limitation):** the upstream cause is that `FilletEdges` builds a
    *self-intersecting neighbour-face loop* when a feature protrudes into the removed strip
    (`transformLoop` pulls back only the edge's own endpoints). A per-face 2D clip cannot fix it —
    the protrusion's vertices are shared with the feature's wall faces, so clipping one face cracks
    the body; the correct fix is a coordinated regularized 3D trim (OCCT `ChFi3d`), a substantial
    fillet-engine effort. Bounded impact: earclip is robust, so area/render are correct and only the
    B-rep topology is blemished. Repro skipped in `kernel/ops/fillet_neighbour_clip_test.go`;
    geometry-math-advisor design in `.superpowers/sdd/progress.md`.
  - `Q1` → **FIXED.** First mis-read as a planar run-out error; the real cause was the curved
    survivor-edge straightening (its end-cap face borders the solid's cylinder along an arc that was
    collapsed to a chord). The `survivorCurve` fix greens it — and, as the same bug, the whole apex
    sextet (see "apex-edge" note above). G1's area-parity bucket is now closed (only `W2,H6` remain,
    reclassified to G6).

## Cross-cutting notes

- **Grounding, per CLAUDE.md:** G1/G2 and all of Phase B/C route through
  `geometry-math-advisor`; the corner/miter architecture (G4/G5) also warrants a
  `software-architect-advisor` pass and an ADR-0050 Phase-6 sub-ADR. Use OCCT `ChFi3d_Builder_
  CnCrn.cxx` as the reference oracle (do like OCCT).
- **Never loosen the gate.** Reds are closed by fixing the engine, never by relaxing the 1%
  area assertion or skipping a case that OCCT itself does not skip.
- **Each package ships its cases green + a live MCP-bridge visual check** (CLAUDE.md Live tests)
  before its slice merges into the branch; the branch merges to develop only when the whole
  corpus is green.
