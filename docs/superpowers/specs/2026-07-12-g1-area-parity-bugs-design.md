# G1 — Area-Parity Fillet Bugs — Design

**Package:** G1 of the OCCT blend greening roadmap
(`docs/superpowers/specs/2026-07-11-occt-blend-greening-roadmap.md`). Its gate is the
`occtparity` corpus: G1 is done when its cases are green **or** (for the interim-guarded
subset) fail *honestly* instead of emitting silently-wrong solids, and no G1 case regresses.

**Goal:** Eliminate the cases where our fillet builds a *valid closed solid* but its area
disagrees with OCCT — either by fixing the geometry, or (where the real fix is deferred to a
later package) by making the engine reject honestly so it never ships a wrong solid.

**Grounding:** the geometry here is corner/convexity reconstruction — route the hard parts
through the `geometry-math-advisor` skill and use OCCT `ChFi3d` as the oracle (CLAUDE.md).

## Investigation summary (verified, this reshaped the package)

The 12 flagged cases were assumed to be one "area bug" class. Evidence
(`model/feature/occtparity` throwaway probes) shows three distinct clusters. Two facts ruled
out the easy explanations first: **refining tessellation 1e-3 → 1e-5 does not move the error**
(so it is not a faceting/measurement artifact — our area is tessellation-based via
`ops.BodyGeometryProperties`, but the fillet geometry itself is wrong), and several cases
produce **byte-identical results for geometrically different inputs** (a systematic bug
fingerprint).

### Cluster 1a — apex-edge fillet on partial revolved primitives (6)
`simple/{A9,B4,B8,C3,D2,D6}`. Inputs are partial `pcylinder`/`pcone`/`psphere` sectors; the
picked edge (OCCT `s_9`) is the **revolution-axis apex edge** — a planar↔planar edge (both
neighbour faces are `geom.Plane`, the two radial cut faces) lying on the axis, whose two
endpoints are **high-valence vertices** (the sector centers, where both radials + the
top/bottom sector faces + the lateral quadric all converge). Verified failure signature:
- A9 = 90° cylinder (apex edge **convex**), B4 = 270° cylinder (apex edge **concave**) →
  **identical** result (area 19098.9, vol 122853.2, 6 faces) despite different base bodies
  (196344 vs 589040 vol). Same pattern for cone (B8/C3) and sphere (D2/D6).
- The fillet removes ~73000 vol³ for an r10 round that should barely change the body, and does
  not distinguish convex from concave — a corner-reconstruction / convexity defect at the
  high-valence axis vertices, **not** a curved-neighbour case (no curved face touches the
  picked edge).

### Cluster 1b — box shared-corner overshoot (2)
`simple/{P8,V8}`. `box 5 5 5`, fillet two convex edges meeting at a corner (r=1). Area +3.4%
(OCCT 145.137, ours 150.1). All faces planar; the excess points to a double-counted or
overlapping corner patch where the two fillet strips meet.

### Cluster 1c — restored real solids (4)
`simple/{W2,Y2,Q1,H6}`. Single-edge fillets on restored B-reps, off 5–13%. No common shape;
each needs its own diagnosis (candidate causes: a curved neighbour that slips a guard like 1a,
a self-intersecting strip, or a genuine radius/geometry error).

## Approach

Investigation-led, one cluster at a time, each ending at the gate (green or honest-fail).

### 1a — apex-edge / high-valence-corner reconstruction
1. **Root-cause** on A9 (convex) and B4 (concave): capture what the 6-face result actually is
   (which faces the corner reconstruction builds at the two axis vertices, and why it removes
   so much) and why convex/concave collapse to the same output. Compare against OCCT's result
   (the oracle STEP of `blend result s 10 s_9` — regenerate it once for A9/B4 to inspect the
   intended geometry).
2. **Decide, from the root cause:**
   - *If tractable within this package* (e.g. the corner reconstruction at an axis vertex uses
     the wrong neighbour faces / ignores the dihedral sign): fix it, so all 6 turn green. This
     is the preferred outcome; ground the corner math in `geometry-math-advisor`.
   - *If it is genuinely the general high-valence / axis-vertex corner problem* (ADR-0050
     Phase 6 territory): add an **interim guard** that detects the pathology and rejects
     honestly (health goes sick with a specific reason), so these stop emitting silent-wrong
     solids, and move the real fix to G5 (corner). Detection must be precise — key it on the
     verifiable pathology (an apex edge whose endpoints are high-valence vertices incident to a
     quadric lateral face), **not** on a result-plausibility heuristic.
3. **Gate:** the 6 cases are green, or fail with the honest reason and are logged to the G5
   backlog. Either way, no silent wrong solid.

### 1b — box shared-corner double-count
1. Reproduce P8 minimally (two r=1 fillets on adjacent edges of a `box 5 5 5`) as a kernel-
   level test; measure where the +3.4% area lives (corner patch vs strip overlap) by comparing
   against the analytic expected area of two quarter-round strips + one corner sphere-patch.
2. Fix the corner assembly (likely the shared-corner patch is added twice or the strips
   overlap). Ground the setback/corner-patch geometry in `geometry-math-advisor`.
3. **Gate:** P8 and V8 green; add a kernel regression asserting the analytic corner area.

### 1c — restored solids, per case
For each of W2, Y2, Q1, H6: reproduce, classify the actual defect, and either fix it here (if
small and self-contained) or reclassify it into the correct greening package (e.g. if it turns
out to be curved-neighbour or corner) — documented in the roadmap, never silently dropped.

## Global constraints

- **Never loosen the gate.** No relaxing the 1% area assertion, no skipping a case OCCT runs.
- **Drive the real feature path** for corpus validation (`AddFilletSets` + `Recompute`), but
  root-cause with minimal `kernel/ops` reproductions and add kernel-level regressions there
  (bug fixes get a regression test — CLAUDE.md).
- **No silent wrong solids.** The floor for every 1a case is an honest rejection; shipping a
  valid-but-wrong solid is the specific defect this package exists to remove.
- **SPDX GPL-2.0-only** on new `.go`; functions ≤20 lines; explicit types.
- **Live check** (CLAUDE.md Live tests): once 1a/1b are fixed, drive one representative case
  (a partial-cylinder apex fillet, and the box-corner) through the MCP bridge and confirm the
  rendered result visually before the slice merges.

## Verification

- Per cluster: the listed corpus cases move from `FAIL(area)`/silent-wrong to `PASS` (fixed)
  or an honest `FAIL(faulty)` with a specific reason (1a interim) — checked via
  `go test ./model/feature/ -run TestOCCTBlend` and the scoreboard.
- Kernel regressions: A9/B4 apex-fillet (asserting either OCCT area or the honest rejection),
  the box shared-corner analytic area (P8), and any 1c fix.
- No regression in the existing fillet suites (`kernel/ops` + `model/feature`
  `-run 'Fillet|Chamfer'`) or the rest of the corpus scoreboard.
