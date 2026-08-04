<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Canal-arm geometric far-runout (M6′ final slice) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement
> this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Implementers run on OPUS;
> reviewers on FABLE (user directive: bulletproof, no compromise).

**Goal:** Green OCCT `tests/blend/simple/N7` (corpus 55→**56**) by giving the three canal arm fillets
their GEOMETRIC far termini (smooth interior runouts, derived and DRAWEXE-verified in closed form) and
the corrected far-end host imprints, then assembling the whole body watertight (Σ=61222.9, 12 faces).

**Architecture:** The W4 blocker was that arm far ends were sought as host-LOOP CROSSINGS (a B3
coincidence); the truth (oracle-verified): **each arm ends where its terminating original face `F_far`
cuts the fillet band** — each host rail ends at rail∩F_far, the terminal boundary is armSurface∩F_far,
and when F_far ⊥ spine this is exactly `farCrossSectionArc` at `m_far = spine∩F_far`. NOT a
ball-tangency (the far ball is CUT by F_far, never tangent). All constructions are closed forms of
{r=5, R=50, ρ=45, d} — zero tuned constants. The derivation corrects the prior ledger claim: B3
far-runout splicing is verbatim ONLY for result_10; result_2/result_4 need new through-vertex extension
edges + (for result_2) a BSpline spiric-section bite.

**Tech Stack:** Go (`oblikovati` GPL, `kernel/ops` + minimal `kernel/geom`). Reuse `farCrossSectionArc`,
`armHostContactRail`/`curvedHostArc` (geometry verbatim, only outer TRIM changes), `canalArmHostRails`,
`canalHostBite`/`canalCloseFar`, `spliceCornerBite`, `assembleBody`. DRAWEXE oracle.

**Base:** the branch tip after W3b (`c127b125` + ledger docs). Derivation (authoritative math):
`.superpowers/sdd/canal-far-runout-derivation.md`. Architecture: `.superpowers/sdd/canal-armweld-architecture.md`.

## Global Constraints

- **NO PR until the whole corpus is green.** This slice greens N7 (55→56) at F3; accumulate + commit per task.
- **B3 + M1–M4 + every green corpus case stays BYTE-IDENTICAL.** The single-ball path is NOT edited;
  all changes live in the canal files (+ any new geom helpers). Gate every task on corpus-55-byte-identical
  (until F3 flips N7) + B3 golden/weld/volume.
- **Honest reject, never a forced green.** Any seam that won't close / wrong Σ / non-manifold edge → the
  do-no-harm floor with the exact gap. NEVER loosen a gate. NO tuned constants — every trim/sweep is a
  closed form of {r, R, ρ, d} from the derivation.
- **Shared-edge identity:** every shared edge is ONE curve object sampled the SAME way
  (`sampleCurve3Open` / same vertex sequence) on both faces. The terminal section curve is shared by the
  arm face (far side) AND the F_far host bite; its feet are the shared outer endpoints of the host rails.
- **Oracle-gated:** derived far geometry must match the DRAWEXE-measured far edges (F1); per-face areas
  gate against the measured role map (F3). The whole-body gate is `TestOCCTBlendSimple/N7` on the REAL
  STEP body (`occtparity/runcase.go`), never only a fixture.
- **Functions within golangci funlen (30 lines/20 statements); files < 500; SPDX GPL-2.0-only on new
  files; explicit types; early returns; ≤2 indent; errors carry offending value + expected shape;
  tolerances model-relative (`res.Weld·scale`) — no bare 1e-6 (scale-free angular guards must be
  justified as dimensionless). `math.NewPoint3` does NOT exist → `math.P3`.**
- **Corpus count (the `-v` is REQUIRED):**
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
- **DRAWEXE oracle:** `source test-utilities/occt-blend/oracle/drawenv.sh`; recipe: restore
  `CFI_f1234fim.rle` → `tscale s 0 0 0 10` → `explode s e` → `blend result s 5 s_5 5 s_4 5 s_10`.

## The oracle-verified geometry (from the derivation — the spec for F1/F2)

**Face role map (per-face area gates for F3):**

| Face | Role | Area |
|---|---|---|
| result_1 | wall (R=50 cyl, retrimmed) | 38033.8 |
| result_2 | x=80 plane (s_5 far imprint, spiric bite) | 1406.8 |
| result_3 | **s_5 torus ARM** | 212.306 |
| result_4 | z=80 plane (s_4 far imprint) | 810.723 |
| result_5 | canal corner patch | 90.194 |
| result_6 | **s_4 cylinder ARM** (= 5·arccos(−1/9)·65, verified) | 546.695 |
| result_7/8 | caps z=130 / z=0 | 7853.98 each |
| result_9 | z=10 floor | 517.428 |
| result_10 | y=30 plane (s_10 far imprint — NOT x=80) | 2094.63 |
| result_11 | x=50 plane (exact rectangle) | 1606.89 |
| result_12 | **s_10 cylinder ARM** | 195.464 |

**Far termini (all closed-form, DRAWEXE residuals <1e-12 except the spiric at 5.5e-7 = OCCT's own tol):**
- **s_4 arm** (F_far = z=80 plane ⊥ spine): m_far = spine∩{z=80} = (45, 5.279, 80); terminal =
  `farCrossSectionArc`: center (45, 50−√2000, 80), feet (50−50/9, 50−(10/9)√2000, 80) and
  (50, 50−√2000, 80), sweep arccos(−1/9).
- **s_10 arm** (F_far = y=30... the y=30 plane): terminal arc center (55, 30, 15), sweep π/2 — exact.
- **s_5 torus arm** (F_far = x=80 plane, NOT ⊥ spine): terminal = the SPIRIC section torus∩{x=80},
  closed form `P(v) = (80, 50−√((45+5cos v)²−900), 5+5sin v)` — a BSpline-sampled section curve, NOT an
  Arc3d. OCCT's own edge satisfies the torus implicit to only 1.6e-8; match pointwise ≤ ~5.5e-7.
- **Host-rail outer trims (closed form):** asin(1/9), asin(3/5), asin(2/3) at the respective rails —
  rail curves stay `armHostContactRail`/`curvedHostArc` geometry VERBATIM; only the outer TRIM changes
  from loop-crossing to rail∩F_far. **EXCEPT the s_5 CAP rail: its sweep must EXTEND past the far-vertex
  azimuth by `asin(d/ρ) − asin(d/R)` = 0.0862 rad** (the current [0→φ*] span is 3.88 units short).
- **Far-end host imprints:** result_10 (y=30): both bite feet ON the original loop → existing
  `spliceCornerBite` VERBATIM. result_4 (z=80) and result_2 (x=80): each has ONE foot OFF the original
  loop → a NEW **through-vertex extension edge** along host∩F_far — z=80: a co-circular arc sweeping
  asin(1/9) past (50,0,80); x=80: a collinear segment of length r past (80,10,10) — shared between the
  F_far face and the wall. `canalCloseFar` must anchor the wall's far path on the
  extension-AUGMENTED window loop. result_2's bite is the spiric BSpline section (and `cornerBiteArea`
  must SAMPLE it, not chord it).
- **All three far ends are clean SINGLE edges** (arc/arc/spiric) — no mini-corners (guard derived for
  future families: the far ball touches only the 2 hosts + F_far).

## File Structure

- `kernel/ops/fillet_curved_canal_far.go` (create) — far termini: `canalFarStation` (m_far = spine∩F_far),
  `canalTerminalSection` (arc / spiric per arm), the rail outer-trim closed forms, the s_5 cap-rail
  sweep extension.
- `kernel/geom` (only if a spiric-section sampler doesn't exist) — a small torus∩plane section curve
  helper (verify against existing SSI/section machinery first — reuse before adding).
- `kernel/ops/fillet_curved_canal_arms.go` (modify) — `canalArmHostRails` outer ends ← geometric far
  feet; the far side of the arm loop ← the shared terminal section.
- `kernel/ops/fillet_curved_canal_bite.go` (modify) — `canalCloseFar` on the extension-augmented loop;
  the two extension edges; the spiric bite sampling.
- `kernel/ops/fillet_curved_canal_weld.go` (modify) — final assembly wiring (F3).
- `model/feature/occt_blend_simple_test.go` (modify at F3) — N7 whole-body + per-face gates.
- Tests alongside each.

Task order — **F1 → F2 → F3 → F4** — is load-bearing: F1 makes the arm faces build on the REAL body
(termini + rails); F2 fixes the far-end host imprints (extensions + spiric bite); F3 assembles → greens
N7; F4 gates.

---

### Task F1: geometric far termini + terminal sections + rail outer trims

**Files:**
- Create: `kernel/ops/fillet_curved_canal_far.go` (+ `_test.go`)
- Modify: `kernel/ops/fillet_curved_canal_arms.go` (outer ends), `fillet_curved_canal_weld.go` (bundle)

**Interfaces:**
- Consumes: the arm offset spines (per-arm, from the reflected-center machinery), the terminating faces
  F_far (identified from the real body's topology at each edge's far vertex — derive the identification
  rule from the derivation doc: F_far = the non-host original face at the far vertex), `farCrossSectionArc`,
  `curvedHostArc`, `canalArmHostRails`, the W2/W3b bundle (`armRails{far; rails[2]; hosts[2]}`).
- Produces: `canalFarStation(arm, ...) (m_far, F_far, ok)`; `canalTerminalSection(arm, m_far, F_far, res)
  (geom.Curve3, ok)` — Arc3d for s_4/s_10 (via `farCrossSectionArc` when F_far ⊥ spine), the spiric
  BSpline section for s_5 (torus∩plane, sampled + fitted, pointwise ≤ the derivation's tol); rails' outer
  ends = the terminal section's host feet (the closed-form trims incl. the s_5 cap-rail +0.0862 rad
  extension); the bundle's `far` ← the shared terminal section.

- [ ] **Step 1 — Failing test (oracle-pinned):** on the REAL N7 body (drive `runcase`/the corpus fixture
  path far enough to obtain the arms + hosts), assert per arm: m_far, the terminal section's center/feet/
  sweep (s_4: center (45, 50−√2000, 80), sweep arccos(−1/9); s_10: (55,30,15), π/2; s_5: the spiric
  P(v) form sampled pointwise) match the derivation's closed forms to `res.Weld·scale` (spiric: its
  documented tol); the host rails' outer ends equal the terminal feet; the s_5 cap rail carries the
  +asin(d/ρ)−asin(d/R) extension. Assert the three ARM FACES now BUILD on the real body (the W4 blocker
  gone): each `canalArmFace` loop closes with the terminal section as its far side.
- [ ] **Step 2 — Run, verify it fails** (current outer ends are loop-crossings; z=130 vs 80).
- [ ] **Step 3 — Implement** (decompose: far-vertex→F_far identification, spine∩F_far station, per-type
  terminal section, spiric sampler (reuse geom section/SSI machinery if present), rail trim, cap-rail
  extension, bundle threading).
- [ ] **Step 4 — Run tests; verify pass** (all oracle residuals reported).
- [ ] **Step 5 — Corpus `55` byte-identical (N7 still floors on F2's imprints); B3 unchanged;
  build/vet/gofmt/golangci-lint clean.**
- [ ] **Step 6 — Commit** (`feat(ops): geometric canal-arm far termini — sections + rail trims (F1)`).

---

### Task F2: far-end host imprints — extension edges + spiric bite

**Files:**
- Modify: `kernel/ops/fillet_curved_canal_bite.go` (+ `_test.go`), `fillet_curved_canal_retrim.go`

**Interfaces:**
- Consumes: F1's terminal sections + feet; `spliceCornerBite`, `farArcsBiting`/`farRunoutFace`,
  `insertSplits`, `cornerBiteArea`, the extension-edge closed forms (z=80: co-circular arc sweep
  asin(1/9) past (50,0,80); x=80: collinear segment length r past (80,10,10)).
- Produces: result_10 (y=30) bite via the existing splice VERBATIM (both feet on-loop — assert);
  result_4/result_2 bites via the through-vertex extension edges (each shared between F_far and the
  wall — ONE curve object both faces sample); the spiric bite on result_2 SAMPLED (not chorded) incl.
  `cornerBiteArea`; `canalCloseFar` anchoring the wall's far path on the extension-augmented window loop.

- [ ] **Step 1 — Failing test:** on the real body: the y=30 bite splices verbatim; the z=80 and x=80
  bites close via their extension edges (each edge point-identical on its two faces); the wall's far
  path anchors on the augmented loop; the spiric bite's `cornerBiteArea` uses sampling (assert its area
  contribution vs a fine-sampled reference, NOT a chord approximation).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** (extension-edge builders from the closed forms; augmented-loop anchoring;
  spiric sampling in the bite; wire into `canalHostBite`/`canalCloseFar`).
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus `55` (N7 floors only on F3 final assembly — or greens early IF everything
  closes honestly); B3 unchanged; lint clean.**
- [ ] **Step 6 — Commit** (`feat(ops): canal far-end host imprints — extension edges + spiric bite (F2)`).

---

### Task F3: whole-body assembly → green N7 (corpus 55→56)

**Files:**
- Modify: `kernel/ops/fillet_curved_canal_weld.go`, `model/feature/occt_blend_simple_test.go`

**Interfaces:**
- Consumes: everything F1/F2 + W0–W3b; `assembleBody`; the W4-scoped reconciliations (shared-edge
  vertex-sequence identity: host bites sample shared curves into the SAME `ringSegSamples` sequence as
  arm/corner faces; the feet[1] adjacency manifold check).

- [ ] **Step 1 — Failing (whole-body) test:** `TestOCCTBlendSimple/N7` on the REAL body: Σ=61222.9
  (corpus tol), **12 faces**, per-face areas match the role-map table (arm faces 212.306 / 546.695 /
  195.464; corner 90.194; retrimmed hosts per table), WATERTIGHT (Valid + HolesContained + IsSolid;
  every edge exactly 2-incident), volume correct.
- [ ] **Step 2 — Run, verify it fails** (final assembly not wired).
- [ ] **Step 3 — Implement:** the final `canalWeldFaces` assembly (corner + 3 arms + all host faces →
  empty reason → `assembleBody`); the vertex-sequence reconciliation; the manifold check. Honest-reject
  with the exact edge/gap if any seam won't close.
- [ ] **Step 4 — Run; verify N7 greens.** Corpus prints `56`.
- [ ] **Step 5 — Non-regression:** whole corpus byte-identical except N7; B3 golden/weld/volume
  unchanged; full `go test ./kernel/... ./model/...` + lint clean.
- [ ] **Step 6 — Commit** (`feat(ops): green OCCT blend/simple/N7 — canal corner + geometric far runout (corpus 55→56)`).

---

### Task F4: DRAWEXE gate + non-regression sweep + coverage

- [ ] **Step 1 —** Re-run the DRAWEXE N7 recipe; confirm Σ + per-face vs our build; record output.
- [ ] **Step 2 —** Full `go test ./...` + `golangci-lint run` + `gofmt -l` + markdownlint; whole corpus
  byte-identical except N7.
- [ ] **Step 3 —** Coverage > 80% on the canal files; duplication < 3%.
- [ ] **Step 4 — Commit** (fold into F3 if no new files).

---

## Verification

- **N7:** Σ=61222.9, 12 faces, per-face role-map areas (incl. the three arm faces + corner 90.194),
  watertight, volume; corpus 55→56; DRAWEXE-confirmed.
- **B3 + whole corpus byte-identical except N7** (single-ball path untouched; canal path only fires on
  `Canal != nil`).
- **Zero tuned constants:** every trim/sweep/extension is a closed form of {r, R, ρ, d}; the area gates
  CHECK, never fit.
- **Shared-edge identity everywhere:** terminal sections, extension edges, foot-loci, rails — one curve
  object, same sampling, both faces.
- **Honest reject:** any non-closing seam → floor with the exact gap. Escalate genuinely new geometry to
  the advisors; never widen a tolerance.
- **Before any PR:** full suite + lint + coverage/duplication + SPDX + live MCP-bridge N7 fillet +
  screenshot; NO PR until the whole corpus is green.

## Risks & escalations
- **F_far identification on the real topology** (F1): the rule is "the non-host original face at the
  edge's far vertex"; if a body presents an ambiguous far vertex → honest-decline, escalate.
- **The spiric section fit** (F1/F2): pointwise tol is the derivation's documented bound (OCCT itself
  is only 1.6e-8 on the torus implicit); do not chase tighter than the oracle.
- **Mini-corner families** (future): the derivation's guard (far ball touches >2 hosts + F_far) →
  honest-decline for now.
- **Carried Minors** (whole-branch review): prior ledger entries + the W3b anchor-tol nit
  (`res.Weld·radius` vs `·scale`).

## References
- `.superpowers/sdd/canal-far-runout-derivation.md` (THE math: termini, sections, trims, extensions,
  spiric, identity contract, tolerances).
- `.superpowers/sdd/canal-armweld-architecture.md` (+ W3 addendum), `.superpowers/sdd/armweld-w4-report.md`
  (the blocker + the reverted prototype recipe).
- `docs/superpowers/2026-07-17-canal-corner-m6prime-status-and-armweld-blueprint.md`;
  `docs/superpowers/plans/2026-07-17-canal-aware-arm-weld.md` (W0–W3b, landed).
- OCCT ChFi3d (reference); Patrikalakis & Maekawa (sections/SSI).
