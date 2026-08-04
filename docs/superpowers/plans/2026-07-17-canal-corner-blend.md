<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Rolling-ball canal corner blend (M6′) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement
> this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Green OCCT `tests/blend/simple/N7` by building a rolling-ball **canal-surface corner provider**
in the RailLoop engine — spine = inner-offset host SSI, radius-r circular cross-sections, emitted as an
exact rational-quad × spline BSpline — faithful to OCCT ChFi3d, area emergent, **no tuned constant**.
Keep B3 byte-identical and M1–M4 unchanged. Corpus 55→**56**.

**Architecture:** A new `canalProvider` tier in the existing RailLoop/`railProvider` engine (`kernel/ops`),
backed by a topo-free canal-surface builder in `kernel/geom`. The corner's two roll hosts + radius ride a
new nilable `RailLoop.Canal` payload (mirroring the existing `ObstacleFeature` optional). The clean octant
carries `Canal=nil` → `analyticSphere` still wins → B3 byte-identical. The landed Duchon plate solver is
kept dormant (correct code, wrong model for this face).

**Tech Stack:** Go (`oblikovati` GPL module). Reuse `geom.OffsetSurface`, `geom.IntersectSurfacesAnalytic`
/ `geom.TraceSurfaceIntersection`, `geom.NewConicSectionCurve` / `NewRuledSectionBlend`,
`geom.NewBSplineSurface` / `ApproximateSurfaceLS`, `Resolution.Weld`. DRAWEXE oracle. OCCT ChFi3d as the
reference implementation.

**Base:** `02fd1932` (corpus 55). **Supersedes:** the plate model for `result_5` (the plate ops-stub +
`RailSignatureTangentPlate` marker are removed here; `kernel/geom/plate_*.go` stays dormant).

## Global Constraints

- **NO PR until the whole corpus is green.** This milestone targets corpus 55→56 (N7) at C4/C5;
  accumulate + commit per task.
- **NO tuned scalar anywhere on the canal path.** The area 90.194 emerges from the rolling-ball geometry
  (radius-r arcs on the offset-SSI spine). The area gate CHECKS the geometry; it does not FIT it. A magic
  constant is a milestone failure.
- **SPDX GPL-2.0-only** header on every new `.go` (run `scripts/add-spdx-headers.py`).
- **Functions 4–20 lines (repo golangci funlen = 30 lines/20 statements); files < 500; explicit types;
  early returns; ≤ 2 levels indent; no code duplication.** Error messages carry the offending value +
  expected shape.
- **Engine dependency rules:** `kernel/geom/canal_*.go` import ONLY `math` (no `ops`, no `topo`);
  `kernel/ops/corner_provider_canal.go` imports geom + math, **never `topo`** (providers topo-free,
  ADR-0051). The `RailLoop.Canal` payload is the ONLY extractor→provider channel; `geom` never reads it.
- **Tolerances model-relative (ADR-0042):** `res.Weld()` scaled by model scale; corner-local by `r` (=5),
  wall by `R` (=50), spine/feet on-host by `res.Weld·scale`. **No bare 1e-6.**
- **Do-no-harm floor:** any offset self-intersection / SSI non-convergence / point-spine / certify
  failure ⇒ `canalProvider.Build` returns `false` ⇒ resolveBlend falls through to `coons4` (N7 is
  downside-protected: can only improve or fall back to today's coons4).
- **B3 byte-faithful + whole corpus byte-identical on untouched cases:** the clean octant is valence-3
  with `Canal=nil` → analyticSphere wins → canal never runs. Every non-N7 case keeps `Canal=nil` →
  canal declines → coons4/sphere/tri3 unchanged. The plate-stub removal is corpus-neutral (it always
  declined).
- **Watertight:** `canalProvider.Build` MUST emit `Loops` on the RECEIVED rails
  (`railLoopToFilletLoops(loop)`), exactly as coons4 does; the canal patch boundary = the RailLoop rails.
- **Corpus count (the `-v` is REQUIRED):**
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
  — must print `55` until C4, then `56`.
- **DRAWEXE oracle:** `source test-utilities/occt-blend/oracle/drawenv.sh`; the faithful N7 recipe is
  `restore <CFI_f1234fim.rle> s; tscale s 0 0 0 10; explode s e; blend result s 5 s_5 5 s_4 5 s_10` →
  whole-body 61222.9, `result_5` = 90.194 (re-confirmed 2026-07-17; the vendored fixture is correct). The
  exact 3×10 pole net is at `.superpowers/sdd/result5-poles.txt` (the emission oracle).
- **TDD, assert against the DRAWEXE oracle, not our own output.** Tests can have bad premises (a prior
  effort shipped a fixture modeling impossible topology; a naive spine missed by +75%). Verify every fake
  against the real geometry.
- **`math.NewPoint3` does NOT exist → use `math.P3(x,y,z math.Scalar)`.** Verify every geom constructor
  (offset/SSI/arc/BSpline) against the real code before use.

## File Structure

- `kernel/ops/corner_rail.go` (modify) — add `Canal *CanalCorner` field + `CanalCorner` struct; remove
  the `RailSignature` enum + `Signature` field (plate marker).
- `kernel/ops/corner_blend.go` (modify) — add `BlendKindCanal`; remove `BlendKindPlate` (plate stub).
- `kernel/ops/corner_extract_tangent.go` (modify) — populate `loop.Canal` (both roll hosts + radius) in
  `extractTangentDegenerateCorner`; remove the `RailSignatureTangentPlate` stamp.
- `kernel/ops/corner_resolve.go` (modify) — `blendTiers() = [analyticSphere, canalProvider, coons4, tri3]`
  (canal replaces the plate stub).
- `kernel/ops/corner_provider_plate.go` + `_test.go` (delete) — the plate ops stub (Build-always-declines).
- `kernel/ops/corner_provider_canal.go` (create) + `_test.go` — `canalProvider` (Name/Fits/Build).
- `kernel/ops/fillet_curved_weld.go` (modify) — add `case BlendKindCanal` to the `curvedCornerFace`
  Kind-gate whitelist.
- `kernel/geom/canal_spine.go` (create) + `_test.go` — `canalSpine` (offset-SSI + endpoints + pinch guard).
- `kernel/geom/canal_arc.go` (create) + `_test.go` — `crossSectionArc` (exact rational-quadratic).
- `kernel/geom/canal_fill.go` (create) + `_test.go` — `CanalCornerFill` (compose spine + arc + loft).
- `kernel/geom/plate_*.go` — UNCHANGED (kept dormant).
- `model/feature/occt_blend_simple_test.go` (modify at C4/C5) — N7 assertions.

Task order — **C0 → C1 → C2 → C3 → C4 → C5** — is load-bearing: C0 lands the ops seam corpus-55
byte-identical (canal stub declines); C1/C2 build the geom math bottom-up with unit tests independent of
the engine; C3 wires them; C4 welds + flips the corpus; C5 gates non-regression.

---

### Task C0: RailLoop.Canal payload + extractor populate + plate-stub removal + canal stub

**Files:**
- Modify: `kernel/ops/corner_rail.go` (add `Canal` payload + `CanalCorner`; remove `RailSignature`/`Signature`)
- Modify: `kernel/ops/corner_blend.go` (add `BlendKindCanal`; remove `BlendKindPlate`)
- Modify: `kernel/ops/corner_extract_tangent.go` (populate `loop.Canal`; remove the marker stamp)
- Modify: `kernel/ops/corner_resolve.go` (`blendTiers` canal replaces plate)
- Delete: `kernel/ops/corner_provider_plate.go`, `kernel/ops/corner_provider_plate_test.go`
- Create: `kernel/ops/corner_provider_canal.go` (STUB: `Fits` real, `Build` returns `false`), `_test.go`

**Interfaces:**
- Consumes: `RailLoop{Sides []Side; Provenance topo.Lineage}` (corner_rail.go); `railProvider` interface;
  `blendTiers()` (corner_resolve.go:8); `extractTangentDegenerateCorner`/`wallFeetSplit`
  (corner_extract_tangent.go — it resolves both roll hosts and knows `w.radius`).
- Produces: `type CanalCorner struct { Rolls []geom.Surface; Radius float64 }`;
  `RailLoop.Canal *CanalCorner` (nil default); `BlendKindCanal CornerBlendKind`; `canalProvider` with
  `Fits(l RailLoop) bool { return l.Canal != nil && l.Valence() == 4 }`, `Build` returning
  `(CornerBlendPatch{}, Certificate{}, false)` (stub).

- [ ] **Step 1 — Failing test:** `canalProvider.Fits` true iff `Canal != nil && Valence()==4`; false for
  `Canal==nil` and for a canal-payload valence-3 loop. Assert `blendTiers()` order = `[analyticSphere,
  canal, coons4, tri3]` by `Name()`. Assert an N7-marked loop (Canal populated) still resolves to
  `BlendKindCoons4` (stub declines → falls through). Assert `RailSignature`/`BlendKindPlate` are gone
  (compile-level: the plate stub file + marker removed).
- [ ] **Step 2 — Run, verify it fails** (`CanalCorner`/`BlendKindCanal`/`canalProvider` undefined).
- [ ] **Step 3 — Implement:** add `CanalCorner` + `RailLoop.Canal` (nil default); add `BlendKindCanal`;
  populate `loop.Canal = &CanalCorner{Rolls: <both roll hosts>, Radius: w.radius}` in
  `extractTangentDegenerateCorner` on the DEGENERATE valence-4 branch (the octant valence-3 branch leaves
  `Canal=nil`); delete the plate stub file + test + the `RailSignature`/`Signature`/
  `RailSignatureTangentPlate`/`BlendKindPlate`; add the canal STUB provider; `blendTiers` insert canal in
  the plate's old slot.
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus gate:** corpus prints `55`; B3 golden/weld/volume + N7 subtest byte-identical to
  base `02fd1932` (canal stub declines → coons4 still serves N7; the plate stub always declined too, so
  its removal is neutral). `go build ./... && go vet ./kernel/... && gofmt -l kernel/ops && golangci-lint
  run` clean.
- [ ] **Step 6 — Commit** (`feat(ops): RailLoop.Canal payload + canal tier stub, retire plate stub (corpus-neutral)`).

---

### Task C1: offset-SSI spine

**Files:**
- Create: `kernel/geom/canal_spine.go`, `kernel/geom/canal_spine_test.go`

**Interfaces:**
- Consumes: `geom.OffsetSurface`, `geom.IntersectSurfacesAnalytic`, `geom.TraceSurfaceIntersection`,
  `geom.Surface`/`Curve3`, `Resolution.Weld` (read their REAL signatures first — grep `kernel/geom` for
  `func OffsetSurface`, `func IntersectSurfacesAnalytic`, `func TraceSurfaceIntersection`).
- Produces: `canalSpine(rolls []geom.Surface, radius float64, ends [2]math.Point3, res Resolution)
  (geom.Curve3, error)` — offset each of the two roll hosts inward by `radius` (cavity side), SSI the
  offsets (analytic dispatch → marched fallback), trimmed to the endpoint ball-centers `ends` (the
  reflected-family centers, N7: C″→C); error on offset self-intersection (`OffsetSurface.SelfIntersects`)
  / SSI non-convergence / point-spine (`length ≤ res.Weld·scale`), carrying the offending value.

- [ ] **Step 1 — Failing test:** N7's two roll hosts (wall cylinder R=50 axis at (50,50); the s_10 host)
  offset by r=5 → `canalSpine` returns a curve from C″=(55,5.27864,5) to C=(45,5.27864,15) (endpoints to
  `res.Weld·scale`), lying in y≈5.27864, with every point at distance r=5 from BOTH offset hosts (i.e.
  the ball at each spine point touches both hosts). Cross-check against the spike
  (`experiments/n7-blend-sweep`) values. A point-spine (equal offsets → concurrence) → error. **Math
  owner:** dispatch geometry-math-advisor to confirm the offset signs (into cavity: wall 50→45, s_10
  5→10) + the tangential-pinch Gauss-Newton guard before implementing.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the offset + SSI dispatch + endpoint trim + reject paths (each a 4–20-line
  helper: offset-both, ssi-dispatch, trim-to-ends, pinch-guard).
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — `go build ./... && go vet ./kernel/geom && gofmt -l kernel/geom && golangci-lint run
  ./kernel/geom/...` clean; corpus 55.**
- [ ] **Step 6 — Commit** (`feat(geom): rolling-ball canal spine via inner-offset host SSI`).

---

### Task C2: cross-section arc + homogeneous loft

**Files:**
- Create: `kernel/geom/canal_arc.go`, `kernel/geom/canal_arc_test.go`, `kernel/geom/canal_fill.go`
  (loft portion), split if `canal_fill.go` would exceed 500.

**Interfaces:**
- Consumes: `canalSpine` (C1); `geom.NewConicSectionCurve`/`NewRuledSectionBlend` (or the real rational-
  arc constructor — grep), `geom.NewBSplineSurface`/`ApproximateSurfaceLS`; `Resolution.Weld`.
- Produces: `crossSectionArc(m, fa, fb math.Point3, radius float64) (geom.Curve3, error)` — the exact
  radius-`radius` rational-quadratic arc `fa → shoulder → fb`, weight `cos(½·∠(fa,m,fb))`, in the plane of
  `{fa, fb, m}`, cavity side; error if `fa,fb,m` collinear (grazing) carrying the angle. Plus
  `loftCanal(spine geom.Curve3, hosts [2]geom.Surface, radius float64, res Resolution) (BSplineSurface,
  error)` — sample the spine (v = chord-length), at each v compute the two feet + `crossSectionArc`, and
  homogeneous-loft (Piegl&Tiller §10.3) the three arc control-curves `(w·x,w·y,w·z,w)` → deg-2-rational-u
  × spline-v; u = arc parameter (u=0→fa, u=1→fb).

- [ ] **Step 1 — Failing test:** (a) `crossSectionArc` on the N7 v=0 config (feet on E1, center C″) is a
  radius-5 quarter-circle matching E1 (sampled points on the arc to `res.Weld·r`; collinear feet → error).
  (b) `loftCanal` on the N7 spine → the surface's numerically-integrated **area = 90.194 within
  `res.Weld·r²`** (EMERGENT — no scalar), its 4 boundary isoparms equal the 4 N7 rails
  (`res.Weld·scale`), and its foot-loci lie ON the roll hosts (`|dist| ≤ res.Weld·scale`). Cross-check the
  emitted control net against `.superpowers/sdd/result5-poles.txt` (shape deviation < 0.1). **Math
  owner:** geometry-math-advisor confirms the loft (u/v = arc/chord-length, the homogeneous control-curve
  form) before implementing.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the arc + the loft (helpers: arc-plane, arc-weight, feet-at-v, control-curves,
  homogeneous-loft).
- [ ] **Step 4 — Run tests; verify pass** (emergent 90.194 + isoparm rails + feet-on-host).
- [ ] **Step 5 — build/vet/gofmt/lint clean; corpus 55.**
- [ ] **Step 6 — Commit** (`feat(geom): canal cross-section arc + homogeneous loft (emergent N7 area 90.194)`).

---

### Task C3: CanalCornerFill + canalProvider.Build + tier + weld Kind-gate

**Files:**
- Create/modify: `kernel/geom/canal_fill.go` (`CanalCornerFill` orchestration)
- Modify: `kernel/ops/corner_provider_canal.go` (replace the C0 stub Build)
- Modify: `kernel/ops/fillet_curved_weld.go` (`curvedCornerFace` Kind-gate: add `case BlendKindCanal`)
- Test: `kernel/geom/canal_fill_test.go`, `kernel/ops/corner_provider_canal_test.go` (extend)

**Interfaces:**
- Consumes: `canalSpine` (C1), `crossSectionArc`/`loftCanal` (C2), `RailLoop`/`CanalCorner` (C0), the
  coons4/sphere certify helpers (`certifyCoons4Patch`), `railLoopToFilletLoops`.
- Produces: `geom.CanalCornerFill(rolls []geom.Surface, radius float64, rails [4]geom.Curve3, res
  Resolution) (BSplineSurface, error)` (compose spine + loft; error ⇒ caller falls through);
  `canalProvider.Build(l RailLoop, res Resolution) (CornerBlendPatch, Certificate, bool)` → map
  `l.Canal.Rolls/Radius` + the 4 rails → `CanalCornerFill` → certify (reuse) →
  `CornerBlendPatch{Surface, Loops: railLoopToFilletLoops(l), Kind: BlendKindCanal}`; honest-reject
  (error) → return false.

- [ ] **Step 1 — Failing test:** N7's payload-carrying loop → `resolveBlend` returns `BlendKindCanal` with
  a valid Certificate + emergent area 90.194 within tol AND feet-on-host witnessed independently; a
  clean-octant loop (`Canal=nil`, valence-3) → `BlendKindSphere` (B3); a non-canal valence-4 loop →
  `BlendKindCoons4` (unchanged). Assert `curvedCornerFace` admits `BlendKindCanal` (a canal patch is NOT
  silently dropped by the Kind-gate whitelist).
- [ ] **Step 2 — Run, verify it fails** (stub Build declines; Kind-gate drops canal).
- [ ] **Step 3 — Implement** `CanalCornerFill`, the real `Build` (emit Loops on the received rails), the
  Kind-gate `case BlendKindCanal`.
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — B3 reduction:** clean octant → `BlendKindSphere`; B3 golden/weld/volume green. Corpus
  still `55` (N7 whole-body weld still declines upstream at wall termination — that is C4; the corner FILL
  now certifies via canal). build/vet/gofmt/golangci-lint clean.
- [ ] **Step 6 — Commit** (`feat(ops): canalProvider Build + tier + weld Kind-gate (N7 corner = BlendKindCanal 90.194)`).

---

### Task C4: weld T-N7.3/T-N7.4 + green N7 (corpus 55→56)

**Files:**
- Modify: `kernel/ops/fillet_curved_weld.go` (wall termination + far-runout weld of the canal patch)
- Modify: `model/feature/occt_blend_simple_test.go` (N7 whole-body assertions)
- Modify/confirm: `model/feature/occtparity/corpus.json` (N7 expectedArea 61222.9)

**Interfaces:**
- Consumes: the full N7 build path (`occtparity/runcase.go`: import `simple/N7.step` → `AddFilletSets` →
  `Recompute`); the canal patch from C3; the corner-fill plan's `bittenLoop`/far-runout machinery
  (`2026-07-16-n7-tangent-degenerate-corner-fill.md` T-N7.3/T-N7.4 — unchanged by the surface model; they
  weld whatever corner patch resolveBlend produced).

- [ ] **Step 1 — Failing (whole-body) test:** N7 whole-body Σ=61222.9 + per-face `result_5`=90.194
  (emergent) + all 12 faces + watertight + volume. Runs RED (wall termination + far-runout not yet welded
  for the canal patch).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the wall termination (bittenLoop) + far-runout welding the canal patch,
  reusing the corner-fill plan's machinery; the wall weld shares the canal patch's exact E2 isoparm edge
  (watertight — the canal emits Loops on the received rails, so the shared edge is exact, not a
  re-projection).
- [ ] **Step 4 — Run; verify N7 greens.** Corpus prints `56`.
- [ ] **Step 5 — Non-regression:** M1–M4 tripwire + whole corpus byte-identical except N7; B3
  golden/weld/volume unchanged. build/vet/gofmt/golangci-lint clean.
- [ ] **Step 6 — Commit** (`feat(ops): green OCCT blend/simple/N7 via rolling-ball canal corner (corpus 55→56)`).

---

### Task C5: DRAWEXE gate + non-regression + coverage

**Files:**
- Modify: `model/feature/occt_blend_simple_test.go` (per-face + witness assertions, if not already in C4)

**Interfaces:**
- Consumes: the greened N7 (C4); the DRAWEXE oracle.

- [ ] **Step 1 — DRAWEXE gate:** re-run the faithful N7 recipe (whole-body 61222.9, `result_5`=90.194) and
  confirm our build matches; record the output. Confirm the emitted canal control net shape-matches
  `result5-poles.txt` (< 0.1).
- [ ] **Step 2 — Non-regression sweep:** full `go test ./...` + `golangci-lint run` + `gofmt -l` +
  markdownlint clean; whole corpus byte-identical except N7; B3/M1–M4 unchanged.
- [ ] **Step 3 — Coverage/duplication:** coverage > 80% on `kernel/geom/canal_*` and
  `kernel/ops/corner_provider_canal`; duplication < 3%.
- [ ] **Step 4 — Commit** (`test(blend): DRAWEXE gate + non-regression for the canal corner (N7)`) — only
  if this task adds/changes test files beyond C4; otherwise fold into C4.

---

## Verification

- **N7:** whole-body 61222.9 + `result_5` 90.194 **emergent** (from the rolling-ball geometry, no free
  scalar); corpus 55→56; DRAWEXE-oracle-confirmed; control net shape-matches `result5-poles.txt`.
- **B3 byte-faithful:** clean octant → `Canal=nil` → analyticSphere → canal/plate never run; B3 golden/
  weld/volume + corpus subtest unchanged.
- **Whole corpus byte-identical except N7:** the payload is nil everywhere else; coons4/analyticSphere/
  tri3 unchanged; the plate ops-stub removal is corpus-neutral (it always declined).
- **Honest gate:** NO tuned scalar on the canal path; feet-on-host + shared-edge-isoparm rails witnessed
  independently of area (watertightness).
- **Engine purity:** `geom/canal_*.go` import only `math`; `ops/corner_provider_canal.go` import geom+math,
  no `topo`; the `Canal` payload is the sole extractor→provider channel.
- **Before any PR (per CLAUDE.md):** full local suite + golangci-lint + markdownlint, coverage > 80% /
  duplication < 3%, SPDX check, cross-platform build; live MCP-bridge test placing/filleting N7 +
  screenshot; `Closes` the N7 tracking issue. **But NO PR until the whole corpus is green.**

## Escalations & risks (decided in-plan, never by tuning)
- **Offset-SSI tangential ill-conditioning at the corner** (C1): detect `|n_a×n_b| < ε_tan`, seed the SSI
  at the reflected-family center endpoints, Newton→Gauss-Newton on ‖offset-residual‖² through the pinch;
  honest-reject on non-convergence → coons4. geometry-math-advisor owns the guard (C1 Step 1).
- **Offset self-intersection** when r > host min-curvature (`OffsetSurface.SelfIntersects`) → honest-reject.
- **Emergent-area miss** (C2 Step 1): if the loft area misses `res.Weld·r²` of 90.194, STOP and report —
  do NOT tune. Diagnose the spine or the arc plane against the spike's −0.025% construction (the spike
  proves it CAN be reconstructed from our data, so a miss is an implementation bug, not a model failure).
- **Marched-spine cases** beyond N7 (general cyl∩cyl, torus, elliptical-cyl) use the
  `TraceSurfaceIntersection` fallback but are exercised as later corpus cases need them; N7 is closed-form.
- **plate_*.go dormancy:** kept (unit-tested, correct for variational corners); not deleted.
- **Carried Minors for the final whole-branch review** (from the plate ledger): reject comments missing
  offending values; `curvedCornerConsistent` two-sided relaxation on `len(arms)==0`; hard-coded test
  constants; the corner-fill plan's T-N7.3 wall-weld must use the canal's exact E2 isoparm (now guaranteed
  by same-rails emission).

## References
- `docs/superpowers/specs/2026-07-17-canal-corner-blend-design.md` — the M6′ spec.
- `.superpowers/sdd/canal-corner-math.md` — spine-SSI, cross-section, u/v parametrization, loft, B3, pitfalls.
- `.superpowers/sdd/canal-corner-seam-architecture.md` — the RailLoop.Canal payload, tier, plate disposition, weld.
- `.superpowers/sdd/blend-sweep-spike-report.md` — the −0.025% reconstruction + host-offset spine.
- `.superpowers/sdd/n7-fill-rails-rederivation.md` — rails + reflected-family centers C/C′/C″.
- `.superpowers/sdd/result5-poles.txt` — OCCT's exact 3×10 net (the emission oracle).
- `docs/superpowers/plans/2026-07-16-n7-tangent-degenerate-corner-fill.md` — T-N7.3/T-N7.4 weld machinery (reused by C4).
- In-repo ADR-0051 (provider tiers), ADR-0042 (relative tolerances), ADR-0018 (public-API split — not triggered).
- Patrikalakis & Maekawa (SSI/canal/offsets); Pottmann & Peternell (canal surfaces); Piegl & Tiller §10.3
  (rational loft); OCCT ChFi3d (reference implementation).
