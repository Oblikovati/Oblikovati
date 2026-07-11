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
| G1 | **Area-parity bugs** | 6 | P?/audit | Builds a valid solid, area disagrees with OCCT > 1%. **Was 12**; investigation moved the 6 apex-edge cases (`A9,B4,B8,C3,D2,D6`) to G5 (see note). G1 now = 1b box shared-corner (`P8,V8`) + 1c restored (`W2,Y2,Q1,H6`) |
| G2 | **Radius > max / degenerate** | 19 | P? | Max-radius comes back ~0 (`2.7e-15`) at certain scales/faces — a max-width computation bug, likely one fix unlocks many |
| G3 | **Generic invalid-solid (triage)** | 34 | — | Fillet ran but the assembled solid is invalid with no specific reason; diagnose → reassign to G4/G5/G6 or a real assembly bug |
| G4 | **Miter blends** (2 convex edges meet) | 61 | P6 | Two filleted edges meet and the corner/outer face is non-planar, or no outer face exists — miter reconstruction gap |
| G5 | **Corner blends** (vertex convergence) | 60 | P6 | n-way / trihedral vertices, mixed radius, arc-end-not-tangent, endpoint-no-end-face — corner reconstruction (IntersectionAtEnd, n-way). **+6 from G1**: the revolution-axis apex-edge fillet (`A9,B4,B8,C3,D2,D6`) is corner reconstruction at revolution poles (see note) |
| G6 | **Curved-neighbour + fillet-into-fillet** | 51 | P4/P6 | Filleting an edge adjacent to a cylinder/cone/sphere/bspline face, or running into an existing round (#1797) |
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

## G1→G5 apex-edge finding (2026-07-12)

The 6 apex cases were reclassified from G1 to G5 after investigation
(spec `docs/superpowers/specs/2026-07-12-g1-area-parity-bugs-design.md`, kernel test
`kernel/ops/fillet_apex_diagnosis_test.go`):

- The picked edge is the **revolution-axis apex** of a partial primitive (planar↔planar edge on
  the axis; both radial faces seam the quadric). Our fillet removes ~73000 vol³ and is identical
  for convex (A9 90°) and concave (B4 270°) — a corner-reconstruction defect at the revolution
  poles, error scaling with the sector (M1 -0.29%, A9 -10%, B4 -57%).
- **Not interim-guardable.** `simple/M1` fillets a *structurally identical* apex edge (a small
  fused partial cylinder) CORRECTLY (-0.29%), so the engine can do it and no clean structural
  predicate separates the good from the bad — a rejection guard keyed on apex-detection wrongly
  rejects M1. The zero-false-positive gate caught this before it shipped. G5 fixes the pole
  corner reconstruction (which also tightens M1); until then these are honest `FAIL(area)`.

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
