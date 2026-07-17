<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Canal-aware arm-weld (M6′ C4) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement
> this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Weld the validated N7 rolling-ball **canal corner patch** into the whole body — greening OCCT
`tests/blend/simple/N7` (corpus 55→**56**) — via a sibling `canalWeldFaces` assembler, WITHOUT regressing
the byte-identical single-ball weld (B3 + every green corpus case).

**Architecture:** A sibling assembler `canalWeldFaces` (own file), dispatched as a pure prefix in
`assembleCurvedArmBody` on `loop.Canal != nil` (the non-concurrent-spine signal). The single-ball path
(`curvedWeldFaces`/`armRailBundle`) is NOT edited → B3 byte-identity by construction. The canal corner =
three independent single-ball sub-problems: each arm face is built at its own reflected center
(`reflectedArmCentres`) via a one-arm local `cornerWeld`, reusing the existing arm-rail/far-runout helpers
verbatim; only the arm's corner-side rail becomes the shared canal boundary curve. Watertightness is by
shared-curve identity (one boundary-isocurve source sampled by corner/arm/host alike).

**Tech Stack:** Go (`oblikovati` GPL module, `kernel/ops`). Reuse `reflectedArmCentres`, `solveArmSetback`,
`armHostContactRail`, `farCrossSectionArc`, `spliceCornerBite`, `bittenLoop`, `insertSplits`,
`curvedCornerFace`, `patchToFilletFace`, `canalBoundaryIsocurves`/`canalPatchLoops`, `assembleBody`.
DRAWEXE oracle. OCCT ChFi3d reference.

**Base:** `5c15bdb1` (corpus 55). Design: `.superpowers/sdd/canal-armweld-architecture.md` (ADR-C4-1..4).

## Global Constraints

- **NO PR until the whole corpus is green.** This milestone greens N7 (corpus 55→56) at W4;
  accumulate + commit per task.
- **B3 + M1–M4 + every green corpus case stays BYTE-IDENTICAL.** The single-ball path
  (`curvedWeldFaces`/`armRailBundle`) is NOT edited except the dispatch prefix; owner = the prefix. Gate:
  corpus count 55 unchanged until W4; B3 golden/weld/volume byte-identical; `git diff` on the single-ball
  path empty except the prefix + the `torusStation` axial guard (W0, itself B3-gated).
- **Honest reject, never a forced green** (standing rule: a passing test can have a bad premise). Any
  un-closable seam / wrong Σ / non-watertight body → the do-no-harm floor (`curvedArmUnweldedError`) with
  the exact gap — never a loosened gate, never a partial/mis-closed body. Do NOT fabricate fixtures.
- **Shared-edge identity:** every shared rail is ONE curve object sampled the SAME way
  (`sampleCurve3Open`) on both sides (corner/arm/host). No re-derived-coincidence rails (the coons4
  amid-rail 0.28 trap).
- **Dispatch signal is `loop.Canal != nil`** (set by `extractTangentDegenerateCorner` after `wallFeetSplit`
  — exactly when the single-ball model is geometrically invalid), NOT the resolved `BlendKindCanal`.
- **Per-arm centers are re-derived** via `reflectedArmCentres` (single source), NOT widened into the
  `CanalCorner` payload (ADR-C4-3).
- **`canalWeldFaces` and its helpers depend on the existing leaf helpers, never the reverse;** the
  single-ball path depends on none of the new code. `canalWeldFaces` is in `ops` (may touch `topo`).
- **Functions 4–20 lines (repo golangci funlen 30/20); files < 500** (hence the new file(s)); explicit
  types; early returns; ≤2 indent; no code duplication. Error messages carry the offending value + shape.
- **Tolerances model-relative (`res.Weld()·scale`, ADR-0042)** — no bare epsilons except existing
  scale-free angular constants. SPDX GPL-2.0-only on new files. `math.NewPoint3` → `math.P3`.
- **Corpus count (the `-v` is REQUIRED):**
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
  — prints `55` until W4, then `56`.
- **DRAWEXE oracle:** `source test-utilities/occt-blend/oracle/drawenv.sh`; N7 recipe `restore
  <CFI_f1234fim.rle> s; tscale s 0 0 0 10; explode s e; blend result s 5 s_5 5 s_4 5 s_10` → whole-body
  61222.9, `result_5` 90.194 (re-confirmed 2026-07-17).

## File Structure

- `kernel/ops/fillet_curved_weld.go` (modify — the ONLY single-ball edit) — the dispatch prefix in
  `assembleCurvedArmBody`.
- `kernel/ops/fillet_curved_corner_solve.go` (modify) — `torusStation` axial-offset decline (W0).
- `kernel/ops/fillet_curved_canal_weld.go` (create) — `canalWeldFaces` + `canalArmFace` + the tagged
  boundary-isocurve accessor.
- `kernel/ops/fillet_curved_canal_retrim.go` (create, if the retrim grows past a few funcs) —
  `retrimCanalHost` + `canalHostFaces` adapter.
- Test files alongside each (`*_test.go`) + `model/feature/occt_blend_simple_test.go` (W4, N7 whole-body).

Task order — **W0 → W1 → W2 → W3 → W4 → W5** — is load-bearing: W0 lands the solve guard (B3-gated);
W1 lands the dispatch + skeleton (corpus-neutral, N7 still floors); W2/W3 build the arm faces + host
retrims (unit-tested); W4 assembles → greens N7; W5 gates non-regression.

---

### Task W0: torusStation honest axial-offset decline (ADR-C4-4)

**Files:**
- Modify: `kernel/ops/fillet_curved_corner_solve.go` (`torusStation` ~:125)
- Test: `kernel/ops/fillet_curved_corner_solve_test.go`

**Interfaces:**
- Consumes: `torusStation(...)`'s existing signature (read it); `res.Weld()`, the model scale.
- Produces: an added guard `|(C − center)·axis| ≤ res.Weld·scale` (the center must lie ON the torus spine
  plane), declining honestly when off-plane.

- [ ] **Step 1 — Failing test:** a torus-arm station solved at a center 2r off the spine plane (N7's z=15
  vs the z=5 plane) → `torusStation` DECLINES (returns not-ok / error carrying the axial offset + the
  bound), instead of accepting it. A center ON the spine plane (axial offset 0) → accepts (unchanged).
- [ ] **Step 2 — Run, verify it fails** (currently accepts the off-plane center).
- [ ] **Step 3 — Implement** the axial-offset guard (a ≤6-line early return; model-relative tol).
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — B3 regression + corpus:** B3 (concurrent, zero axial offset) still passes; corpus prints
  `55` byte-identical to base `5c15bdb1`; B3 golden/weld/volume unchanged. `go build ./... && go vet
  ./kernel/... && gofmt -l kernel/ops && golangci-lint run` clean.
- [ ] **Step 6 — Commit** (`fix(ops): torusStation honest axial-offset decline (ADR-C4-4)`).

---

### Task W1: dispatch prefix + tagged boundary-isocurve accessor + canalWeldFaces skeleton

**Files:**
- Modify: `kernel/ops/fillet_curved_weld.go` (`assembleCurvedArmBody` prefix — the only single-ball edit)
- Create: `kernel/ops/fillet_curved_canal_weld.go` (`canalWeldFaces` skeleton + the tagged accessor)
- Test: `kernel/ops/fillet_curved_canal_weld_test.go`

**Interfaces:**
- Consumes: `assembleCurvedArmBody` (read its current shape); `extractCurvedCorner(w, arms, res) (RailLoop,
  bool)`; `RailLoop.Canal`; `resolveBlend`; `canalBoundaryIsocurves`/`canalPatchLoops`
  (corner_provider_canal.go); `patchToFilletFace`; `curvedCornerFace`; `curvedArmUnweldedError`/the floor.
- Produces:
  - The dispatch prefix: `if loop, ok := extractCurvedCorner(w, arms, res); ok && loop.Canal != nil {
    faces, reason := canalWeldFaces(body, arms, w, loop, res); return <assembleBody(faces) or floor(reason)> }`
    placed BEFORE the untouched single-ball `curvedWeldFaces` call.
  - `canalWeldFaces(body *topo.Body, arms []edgeFillet, w cornerWeld, loop RailLoop, res Resolution)
    ([]filletFace, string)` — SKELETON: resolve the patch once; build the corner face
    (`patchToFilletFace`/`curvedCornerFace`); return a decline reason for the still-missing arm faces
    (so N7 floors exactly as today). Empty reason = success (not yet).
  - `type canalBoundaries struct { endArcs [2]geom.Curve3; feet [2]geom.Curve3 }` +
    `canalBoundaryRoles(patch CornerBlendPatch) (canalBoundaries, error)` — tag the 4 boundary isocurves by
    role (v0/v1 end arcs @ C″/C; u0 foot-locus on wall; u1 foot-locus on s_10), the SINGLE source for the
    shared rails (ADR-C4-2). Reuse `canalBoundaryIsocurves`; identify roles by center/host geometry.

- [ ] **Step 1 — Failing test:** (a) `canalBoundaryRoles` on the N7 patch returns 2 end arcs (radius-5
  arcs @ C″/C) + 2 foot-loci (on wall R=50 / s_10 R=5), each tagged correctly (assert the end arcs' centers
  = C″/C to `res.Weld·scale`, the foot-loci on-host to `res.Weld·scale`). (b) The dispatch routes an N7
  loop (`Canal != nil`) into `canalWeldFaces` and a `Canal==nil` loop to the single-ball path (assert via a
  seam/spy or by the resulting floor reason). (c) Corpus 55 (N7 still floors — the skeleton declines on
  missing arm faces).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the prefix + skeleton + tagged accessor.
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus/regression:** corpus `55` byte-identical to `5c15bdb1` (canal skeleton floors N7 =
  today's behaviour; single-ball path untouched); B3 golden/weld/volume unchanged; build/vet/gofmt/lint
  clean.
- [ ] **Step 6 — Commit** (`feat(ops): canal-weld dispatch + tagged boundary-isocurve accessor (skeleton)`).

---

### Task W2: per-arm-center arm faces (`canalArmFace`)

**Files:**
- Modify: `kernel/ops/fillet_curved_canal_weld.go` (`canalArmFace` + wire into `canalWeldFaces`)
- Test: `kernel/ops/fillet_curved_canal_weld_test.go` (extend)

**Interfaces:**
- Consumes: `reflectedArmCentres(w, arms, scale, res)`; `solveArmSetback(arm, center, r, scale, res)`;
  `armHostContactRail(hostEdge, setback, station, weld, res)`; `farCrossSectionArc(arm, r, from0, from1)`;
  the `canalBoundaries` from W1; `cornerWeld` (read its fields to build a one-arm local weld); `filletFace`/
  `filletLoop` (assemble_curved.go). READ each helper's REAL signature before wiring.
- Produces: `canalArmFace(arm edgeFillet, center math.Point3, cornerRail geom.Curve3, w cornerWeld, res
  Resolution) (filletFace, bool)` — build the arm face at its reflected `center`: a one-arm local
  `cornerWeld{center, radius:w.radius, arms:[setback]}`; `h0 = armHostContactRail(arm.a, setback, t0, wᵢ)`,
  `h1 = armHostContactRail(arm.b, setback, t1, wᵢ)`, `far = farCrossSectionArc(...)`; loop = `[h0,
  cornerRail, reverse(h1), reverse(far)]` where `cornerRail` is this arm's shared canal boundary curve (an
  end arc for the two wall-sharing arms, the u=1 foot-locus for the mid arm). Return false → decline.

- [ ] **Step 1 — Failing test:** for N7's 3 arms, `canalArmFace` builds 3 valid arm faces: each has a
  closed loop (`filletLoop` closes, first pt = last pt to `res.Weld·scale`), the two host rails lie on
  their hosts (`res.Weld·scale`), the corner-side rail IS the shared canal boundary curve (same curve
  object / point-identical to the corner face's corresponding boundary — the watertight self-check), and
  the far arc is the arm's terminal cross-section. The torus arm (s_5) — which declined under the single
  ball — now builds at its OWN center (z=5) without the 2r gap. Assert each arm face's corner rail is
  point-identical to `canalBoundaries` (shared identity).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** `canalArmFace` (helpers: local-weld, arm-host-rails, corner-rail-select,
  assemble-loop) + wire the 3 arm faces into `canalWeldFaces` (still floors on the missing host retrims).
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus `55` (N7 still floors on host retrims); B3 unchanged; build/vet/gofmt/lint clean.**
- [ ] **Step 6 — Commit** (`feat(ops): per-arm-center canal arm faces (3 single-ball sub-problems)`).

---

### Task W3: `retrimCanalHost` + `canalHostFaces` adapter (host retrims to foot-loci)

**Files:**
- Create: `kernel/ops/fillet_curved_canal_retrim.go` (`retrimCanalHost`, `canalHostFaces`)
- Modify: `kernel/ops/fillet_curved_canal_weld.go` (wire host faces into `canalWeldFaces`)
- Test: `kernel/ops/fillet_curved_canal_retrim_test.go`

**Interfaces:**
- Consumes: `bittenLoop(host, ...)`, `insertSplits`, `spliceCornerBite` (fillet_curved_retrim.go /
  fillet_curved_farrunout.go — read real signatures); `farArcsBiting`/`farRunoutFace` (verbatim for
  far-runout hosts); the `canalBoundaries` foot-loci; the host faces from `body`.
- Produces:
  - `retrimCanalHost(host *topo.Face, footLocus geom.Curve3, w cornerWeld, res Resolution) (filletFace,
    bool)` — re-clip the host so its bitten loop follows the shared `footLocus`: `bittenLoop` picks the
    loop the corner opens; `insertSplits` + `spliceCornerBite` substitute the shared foot-locus `endSeg`
    (the SAME curve the arm/corner sampled). Decline on failure.
  - `canalHostFaces(body, arms, w, boundaries, res) ([]filletFace, string)` — route the two corner roll
    hosts (wall, s_10) + the two planes to `retrimCanalHost` (keyed on the foot-loci); route every
    far-runout bitten host to the existing `farArcsBiting`/`farRunoutFace` path VERBATIM.
- **Watertightness risk (flag, per the architect):** the foot-locus endpoints (wall/plane feet) must lie
  on the host's original bitten loop for the splice to anchor within `res.Weld·scale`. If the exact
  foot-locus↔loop intersection does NOT land within tol, STOP and escalate the intersection to
  geometry-math-advisor — do NOT loosen the gate.

- [ ] **Step 1 — Failing test:** `retrimCanalHost` on N7's wall host + the wall foot-locus → a valid
  retrimmed host face whose bitten loop FOLLOWS the foot-locus (the spliced edge is point-identical to the
  shared foot-locus curve); same for the s_10 host + its foot-locus. `canalHostFaces` routes the far-runout
  hosts through the existing verbatim path (assert those faces are byte-identical to what
  `farArcsBiting`/`farRunoutFace` produce). If the foot-locus↔loop anchor fails, the test surfaces it (the
  escalation trigger).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** `retrimCanalHost` + `canalHostFaces`; wire into `canalWeldFaces`.
- [ ] **Step 4 — Run tests; verify pass** (or, if the anchor risk fires, STOP + escalate per the flag).
- [ ] **Step 5 — Corpus `55` (N7 may now assemble — if so it jumps to W4's gate; if it still floors on the
  final weld, that's W4); B3 unchanged; build/vet/gofmt/lint clean.**
- [ ] **Step 6 — Commit** (`feat(ops): canal host retrims to foot-loci + far-runout reuse adapter`).

---

### Task W4: assemble whole body + green N7 (corpus 55→56)

**Files:**
- Modify: `kernel/ops/fillet_curved_canal_weld.go` (`canalWeldFaces` final assembly)
- Modify: `model/feature/occt_blend_simple_test.go` (N7 whole-body assertions)
- Modify/confirm: `model/feature/occtparity/corpus.json` (N7 expectedArea 61222.9)

**Interfaces:**
- Consumes: the corner face (W1) + 3 arm faces (W2) + retrimmed/far hosts (W3); `assembleBody`; the N7
  build path (`occtparity/runcase.go`).

- [ ] **Step 1 — Failing (whole-body) test:** `TestOCCTBlendSimple/N7`: whole-body Σ area = **61222.9**
  (corpus oracle tol), `result_5` per-face = 90.194 (emergent), all 12 faces present, watertight
  (`Validate.Valid && HolesContained && IsSolid`; every edge 2-incident), volume matches. Runs RED (final
  assembly not wired).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the final `canalWeldFaces` assembly: gather corner + 3 arm faces + host faces
  → return them (empty reason) → `assembleBody` welds by shared points (watertight by shared-curve
  identity). Honest-reject (floor with the exact gap) if any seam won't close or Σ/volume is wrong — do NOT
  loosen the gate.
- [ ] **Step 4 — Run; verify N7 greens.** Corpus prints `56`.
- [ ] **Step 5 — Non-regression:** B3 + M1–M4 tripwire + whole corpus byte-identical except N7; B3
  golden/weld/volume unchanged. build/vet/gofmt/golangci-lint clean.
- [ ] **Step 6 — Commit** (`feat(ops): green OCCT blend/simple/N7 via canal-aware arm-weld (corpus 55→56)`).

---

### Task W5: DRAWEXE gate + non-regression + coverage

**Files:**
- Modify: `model/feature/occt_blend_simple_test.go` (per-face + watertight witnesses, if not in W4)

- [ ] **Step 1 — DRAWEXE gate:** re-run the faithful N7 recipe (whole-body 61222.9, `result_5`=90.194);
  confirm our build matches; record the output.
- [ ] **Step 2 — Non-regression sweep:** full `go test ./...` + `golangci-lint run` + `gofmt -l` +
  markdownlint clean; whole corpus byte-identical except N7; B3/M1–M4 unchanged.
- [ ] **Step 3 — Coverage/duplication:** coverage > 80% on the new `fillet_curved_canal_*`; duplication
  < 3%.
- [ ] **Step 4 — Commit** (`test(blend): DRAWEXE gate + non-regression for the canal-aware arm-weld`) —
  only if this task adds test files beyond W4; else fold into W4.

---

## Verification

- **N7:** whole-body 61222.9 + `result_5` 90.194 **emergent**; corpus 55→56; DRAWEXE-confirmed; watertight
  (Valid + HolesContained + IsSolid) + volume.
- **B3 byte-faithful:** the single-ball path is not edited (only the dispatch prefix + the B3-gated
  `torusStation` guard); clean octant → single-ball weld → B3 golden/weld/volume + corpus subtest
  byte-identical.
- **Whole corpus byte-identical except N7:** `Canal == nil` everywhere else → the dispatch falls through to
  the untouched single-ball path.
- **Shared-edge identity:** every shared rail is one boundary-isocurve object sampled the same way on both
  sides (corner/arm/host); witnessed by the point-identity self-checks (W2/W3).
- **Honest reject:** any un-closable seam / wrong Σ → the do-no-harm floor with the exact gap; NO loosened
  gate, NO forced green.
- **Before any PR (per CLAUDE.md):** full local suite + golangci-lint + markdownlint, coverage > 80% /
  duplication < 3%, SPDX check, cross-platform build; live MCP-bridge test filleting N7 + screenshot;
  `Closes` the N7 tracking issue. **But NO PR until the whole corpus is green.**

## Escalations & risks (decided in-plan, never by loosening a gate)
- **Foot-locus↔host-loop anchor** (W3): if the foot-locus endpoints don't land on the host's bitten loop
  within `res.Weld·scale`, escalate the intersection to geometry-math-advisor — do NOT loosen the splice
  tolerance.
- **Whole-body watertightness** (W4): if a seam won't close despite shared-curve identity, the diagnosis
  (which edge, which gap) is the next slice's input — honest-reject, don't force.
- **`torusStation` shared-function change** (W0): gated by B3 regression (concurrent center → zero axial
  offset → passes) + corpus byte-identity before landing.
- **Carried Minors for the final whole-branch review** (from the canal ledger): the `1e-6 // tol:angular`
  constants (deliberate, scale-free); the 3×163 canal control net (knot-removal is a later optimization,
  not correctness); reject comments carrying offending values.

## References
- `.superpowers/sdd/canal-armweld-architecture.md` — ADR-C4-1..4, the seam signatures, build order,
  invariants.
- `docs/superpowers/2026-07-17-canal-corner-m6prime-status-and-armweld-blueprint.md` — the measured
  topology (§2), the C4 blocker (§3), the blueprint (§4).
- `.superpowers/sdd/canal-c4-report.md` — the precise decline diagnosis.
- `docs/superpowers/specs/2026-07-17-canal-corner-blend-design.md` + `.../plans/2026-07-17-canal-corner-
  blend.md` — the corner-surface milestone (C0–C3, landed).
- In-repo ADR-0051 (provider tiers / topo-free), ADR-0042 (relative tolerances); strangler-fig dispatch
  (mirrors ADR-2 Step 1). OCCT ChFi3d (reference implementation).
