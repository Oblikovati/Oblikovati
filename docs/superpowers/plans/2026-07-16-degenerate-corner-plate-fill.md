<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Degenerate-corner variational plate fill (M6) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement
> this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Green OCCT `tests/blend/simple/N7` by porting OCCT's `GeomPlate` variational corner fill (a
Duchon order-3 polyharmonic thin-plate spline) into the RailLoop engine as a new `plateProvider` tier —
area emergent, **no tuned constant** — keeping B3 byte-identical and the M1–M4 corpus unchanged.

**Architecture:** Ports-&-adapters mirroring `coons4`. `kernel/geom` (imports only `math`) owns the
Duchon solver + BSpline finish; a thin topo-free `plateProvider` in `kernel/ops` adapts it into a
`CornerBlendPatch`. The plate is a NEW tier that COMPLEMENTS coons4 (order: analyticSphere → plate →
coons4 → tri3); a plate honest-reject falls through to coons4. Disambiguated from coons4 by an
extractor-stamped `RailLoop.Signature` marker (classification, not geometry) — NOT a purely geometric
`Fits`.

**Tech Stack:** Go (`oblikovati` GPL module); `kernel/geom` (reuse `gaussSolve`, `ApproximateSurfaceLS`,
`MatchSurface`); `kernel/ops` (RailLoop engine); DRAWEXE oracle; OCCT `Plate_Plate`/`GeomPlate_*` as the
reference implementation.

**Base:** `8e4d6207` (corpus 55). **Supersedes:** the tuned-constant E2 (`wallBridgeFullness=1.136`)
from `8e4d6207` — its RAILS are kept; its fill interior + E2 + the circular area gate are replaced.

## Global Constraints

- **NO PR until the whole corpus is green.** This milestone targets corpus 55→56 (N7) at P5; accumulate +
  commit per task.
- **SPDX GPL-2.0-only** header on every new `.go` (run `scripts/add-spdx-headers.py`).
- **Functions 4–20 lines; files < 500 lines; explicit types; early returns; ≤ 2 levels indent; no code
  duplication.** Error messages carry the offending value + expected shape.
- **Engine dependency rules:** `kernel/geom/plate_*.go` import ONLY `math` (no `ops`, no `topo`);
  `kernel/ops/corner_provider_plate.go` imports geom + math, **never `topo`** (providers are topo-free,
  ADR-0051). The `RailLoop.Signature` marker is the ONLY extractor→provider channel; `geom` never reads it.
- **Tolerances model-relative (ADR-0042):** `res.Weld()` scaled by model scale via
  `ResolutionForPoints`/`ResolutionForBody`; corner-local by `r` (=5), wall by `R` (=50). **No bare 1e-6.**
- **NO tuned scalar anywhere on the plate path.** The area 90.194 is a pure functional of {4 fixed rails,
  m=3 energy}; the area gate CHECKS the solve, it does not FIT it. A magic "fullness"/scale constant is a
  milestone failure (this is the whole point of M6).
- **Do-no-harm floor:** any solver non-convergence / guard failure ⇒ `plateProvider.Build` returns
  `false` ⇒ resolveBlend falls through to `coons4` (N7 is downside-protected; can only improve or fall
  back to today's coons4).
- **B3 byte-faithful + corpus byte-identical on untouched cases:** the octant (valence-3, unmarked) →
  analyticSphere wins → plate never runs. Every non-N7 case keeps `Signature==0` → `plateProvider.Fits`
  false → coons4 unchanged.
- **Corpus count (the `-v` is REQUIRED):**
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
  — must print `55` until P5, then `56`.
- **DRAWEXE oracle:** `source test-utilities/occt-blend/oracle/drawenv.sh`; run
  `printf 'source X.tcl\n' | "$DRAWEXE" -b`. Validate against `simple/N7.step` directly (result_5=90.194);
  the vendored `CFI_f1234fim.rle` is the WRONG shape (8 faces, area 8.04) — reconcile it at P5.
- **TDD, assert against the DRAWEXE oracle, not our own output.** Tests can have bad premises (a prior
  task shipped a fixture modeling impossible topology; a naive off-wall rail gave a deceptively good area
  because coons4 silently re-projects). Verify every fake against the real loop the provider receives.
- **`math.NewPoint3` does NOT exist → use `math.P3(x,y,z math.Scalar)`.** Verify every geom constructor
  (arc/cylinder/torus/BSpline surface) against the real code before use.

---

## File Structure

- `kernel/ops/corner_rail.go` (modify) — add `RailLoop.Signature` field + `RailSignature` enum.
- `kernel/ops/corner_extract_tangent.go` (modify) — stamp `RailSignatureTangentPlate` after
  `wallFeetSplit`; delete the `wallBridgeFullness=1.136` tuned E2 (keep the reflected-family rails, add
  the standard-Hermite/no-scalar E2).
- `kernel/ops/corner_resolve.go` (modify) — insert `plateProvider{}` into `blendTiers()` between
  analyticSphere and coons4.
- `kernel/ops/corner_blend.go` (modify) — add `BlendKindPlate` const.
- `kernel/ops/corner_provider_plate.go` (create) — `plateProvider` (Name/Fits/Build), reuse
  coons4/sphere certify helpers.
- `kernel/geom/plate_average_plane.go` (create) — the 2D average-plane domain Ω + constraint projection.
- `kernel/geom/plate_solve.go` (create) — `PlateSolve`: the Duchon E_m kernel + saddle assembly +
  `gaussSolve` + one-step refinement.
- `kernel/geom/plate_constraints.go` (create) — sample Sides → G0/G1 constraint rows.
- `kernel/geom/plate_fill.go` (create) — `PlateFill`: constraints → solve → grid-eval →
  `ApproximateSurfaceLS` (+ optional `MatchSurface`).
- Test files alongside each (`*_test.go`) + `model/feature/occt_blend_simple_test.go` (modify at P5).

Task order — **P0 → P1 → P2 → P3 → P4a → P4b → P5** — is load-bearing: the ops seam (P0) lands corpus-55
byte-identical first; the geom math (P1–P4a) builds bottom-up with unit tests independent of the engine;
P4b wires them; P5 is the only task that flips the corpus.

---

### Task P0: RailLoop.Signature marker + stamp + plate tier stub

**Files:**
- Modify: `kernel/ops/corner_rail.go` (add field + enum)
- Modify: `kernel/ops/corner_extract_tangent.go` (stamp after `wallFeetSplit`)
- Modify: `kernel/ops/corner_resolve.go` (`blendTiers` insert)
- Modify: `kernel/ops/corner_blend.go` (`BlendKindPlate` const)
- Create: `kernel/ops/corner_provider_plate.go` (STUB: `Fits` real, `Build` returns `false`)
- Test: `kernel/ops/corner_provider_plate_test.go`

**Interfaces:**
- Consumes: `RailLoop{Sides []Side, Provenance topo.Lineage}` (corner_rail.go); `railProvider` interface
  `Name()/Fits(RailLoop)/Build(RailLoop,Resolution)→(CornerBlendPatch,Certificate,bool)`; `blendTiers()
  []railProvider` (corner_resolve.go:8); `extractTangentDegenerateCorner`/`wallFeetSplit`
  (corner_extract_tangent.go).
- Produces: `RailLoop.Signature RailSignature`; `RailSignatureGeneral RailSignature = 0`,
  `RailSignatureTangentPlate`; `BlendKindPlate CornerBlendKind`; `plateProvider` with
  `Fits(l RailLoop) bool { return l.Signature == RailSignatureTangentPlate && l.Valence() == 4 }`.

- [ ] **Step 1 — Failing test:** `plateProvider.Fits` true iff Signature==TangentPlate && Valence()==4;
  false for a default (Signature==0) valence-4 loop and for a marked valence-3 loop. `Build` returns
  `false` (stub). Assert `blendTiers()` order = `[analyticSphere, plate, coons4, tri3]` by `Name()`.
  Assert a marked N7 loop still resolves to `BlendKindCoons4` (stub Build declines → falls through).
- [ ] **Step 2 — Run, verify it fails** (`plateProvider`/`Signature`/`BlendKindPlate` undefined).
- [ ] **Step 3 — Implement:** add the enum + field (default 0), the `BlendKindPlate` const, the stub
  provider, the `blendTiers` insert, and ONE stamp line in `extractTangentDegenerateCorner` (set
  `loop.Signature = RailSignatureTangentPlate` after `wallFeetSplit` confirms the degenerate valence-4;
  the octant valence-3 branch leaves it `RailSignatureGeneral`).
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus gate:** corpus prints `55`; B3 golden/weld/volume + N7 subtest byte-identical to
  base `8e4d6207` (stub always declines → coons4 still serves N7; nothing changed behaviourally).
- [ ] **Step 6 — Commit** (`git commit -m "feat(ops): RailLoop.Signature plate marker + plate tier stub (corpus-neutral)"`).

---

### Task P1: average-plane domain Ω + constraint projection

**Files:**
- Create: `kernel/geom/plate_average_plane.go`
- Test: `kernel/geom/plate_average_plane_test.go`

**Interfaces:**
- Consumes: `geom` value types (`math.Point3`, `geom.Curve3`, `geom.Surface`).
- Produces: `type PlateDomain struct { Origin math.Point3; U, V, N math.Vector3 }`;
  `AveragePlane(anchors []math.Point3) (PlateDomain, error)` (least-squares plane through the corner
  region, N = smallest-singular-vector, error if rank-deficient carrying the offending spread);
  `func (d PlateDomain) Project(p math.Point3) (u, v float64)` and `Lift(u, v, w float64) math.Point3`.

- [ ] **Step 1 — Failing test:** a known co-planar anchor set → N matches the plane normal (to
  `res.Weld`), round-trip `Lift(Project(p))` ≈ p for in-plane p; a near-degenerate (collinear) anchor
  set → error carrying the measured spread + expected minimum. **Math owner:** dispatch to
  geometry-math-advisor for the plane-fit conditioning (SVD vs normal-equations) before implementing.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the least-squares plane (centroid + covariance eigenvector) + project/lift.
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — `go build ./... && go vet ./kernel/geom && gofmt -l kernel/geom` clean; corpus 55.**
- [ ] **Step 6 — Commit** (`feat(geom): average-plane domain for the plate corner fill`).

---

### Task P2: Duchon order-3 polyharmonic solver (the TPS core)

**Files:**
- Create: `kernel/geom/plate_solve.go`
- Test: `kernel/geom/plate_solve_test.go`

**Interfaces:**
- Consumes: `gaussSolve` (kernel/geom/fitted_bspline.go:134 — Gauss-Jordan + partial pivot);
  `PlateDomain` (P1).
- Produces: `type PlateConstraint struct { U, V float64; Order [2]int; Value float64 }` (Order {0,0}=G0
  point, {1,0}/{0,1}=first-derivative rows); `PlateSolve(cs []PlateConstraint) (PlateCoeffs, error)` —
  builds the bordered saddle system `[K Pᵀ; P 0][λ;a]=[v;0]` (K_ij = E_m derivative of r⁴log(r²),
  quadratic reproduction basis {1,u,v,u²,uv,v²}), solves per RHS via `gaussSolve`, one Richardson
  refinement step, returns error on non-finite / non-converged residual (carrying the residual norm +
  tol); `func (c PlateCoeffs) Eval(u, v float64) float64` and `EvalGrad`.

- [ ] **Step 1 — Failing test (analytic reproduction):** constrain the solver with samples of a KNOWN
  order-3 polyharmonic function (e.g. a low-degree polynomial in the reproduction span, exactly
  reproduced) → `Eval` matches to `res.Weld`; a rank-deficient constraint set → error carrying the
  condition estimate. **Math owner:** geometry-math-advisor owns the E_m derivative table (the exact
  `r⁴log r` first/second derivatives), the reproduction basis, and the conditioning guard (ridge term
  vs pivoting) — dispatch it the derivation `.superpowers/sdd/n7-plate-solve-derivation.md` and have it
  confirm the derivative rows before coding.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the kernel, the bordered assembly, `gaussSolve` + one refinement step, the
  non-convergence reject. Keep each helper 4–20 lines (kernel eval, derivative row, assembly, solve,
  refine, eval as separate funcs); file < 500 (split into `plate_solve.go` + `plate_kernel.go` if needed).
- [ ] **Step 4 — Run tests; verify pass** (analytic reproduction + reject-on-degenerate).
- [ ] **Step 5 — build/vet/gofmt clean; corpus 55.**
- [ ] **Step 6 — Commit** (`feat(geom): Duchon order-3 polyharmonic plate solver`).

---

### Task P3: constraint discretization (Sides → PlateConstraint rows)

**Files:**
- Create: `kernel/geom/plate_constraints.go`
- Test: `kernel/geom/plate_constraints_test.go`

**Interfaces:**
- Consumes: `geom.Curve3`, `geom.Surface`, `PlateDomain` (P1), `PlateConstraint` (P2).
- Produces: `type PlateSide struct { Curve geom.Curve3; Adjacent geom.Surface; Order int }`;
  `DiscretizeSides(sides [4]PlateSide, d PlateDomain, samples int) ([]PlateConstraint, error)` — each
  side sampled → G0 rows at `Project(curve(t))`; for Order==1 (G1) sides, add first-derivative rows
  carrying the `Adjacent` surface's tangent-plane slope at the projected foot (the exact analytic host
  normal, NOT a chord). Error if a side projects to a near-degenerate strip (carrying the measured
  arc-length + expected minimum).

- [ ] **Step 1 — Failing test:** 4 synthetic sides (a unit square with known G1 slopes) → the assembled
  constraint set has the expected G0 count + G1 rows with the correct slopes (to `res.Weld`); a side
  whose Adjacent normal is ill-defined → error. **Math owner:** geometry-math-advisor confirms the G1
  row assembly (how the surface tangent maps into the ∂/∂u,∂/∂v rows in the average-plane chart).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the sampling + G0/G1 row assembly + degenerate-strip reject.
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — build/vet/gofmt clean; corpus 55.**
- [ ] **Step 6 — Commit** (`feat(geom): plate constraint discretization from rail sides`).

---

### Task P4a: plate → BSpline finish (`PlateFill`)

**Files:**
- Create: `kernel/geom/plate_fill.go`
- Test: `kernel/geom/plate_fill_test.go`

**Interfaces:**
- Consumes: `AveragePlane`/`PlateDomain` (P1), `PlateSolve`/`PlateCoeffs` (P2), `DiscretizeSides`/
  `PlateSide` (P3), `ApproximateSurfaceLS(points, us, vs, du, dv, nu, nv)`
  (kernel/geom/approximate_surface.go), `MatchSurface`.
- Produces: `PlateFill(sides [4]PlateSide, tol float64) (BSplineSurface, error)` — average-plane →
  discretize → solve per coordinate (X/Y/Z each a `PlateSolve` over the same K, 3 RHS) → grid-eval the
  lifted surface → `ApproximateSurfaceLS` (degmax ≤ 8, ~9 spans) → optional `MatchSurface` to nail G1 <
  tol; error ⇒ caller falls through.

- [ ] **Step 1 — Failing test:** feed the 4 real N7 rails (reflected-family E1/E3/E4 + the standard-
  Hermite on-wall E2 — construct them in the test from the derivation's validated vertices/mids) → the
  returned surface's G0 residual on each rail < tol, G1-to-Adjacent residual < tol, and **surface area
  (numerically integrated) matches 90.194 within `res.Weld·r²`** — assert against the DRAWEXE value,
  emergent (no scalar in `PlateFill`). If prescribe-E2 misses the area gate, STOP and report BLOCKED for
  the emerge-E2 escalation (SSI + same-parameter) — do NOT tune a scalar.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the finish (thin: it composes P1–P3 + reuse). ≤ 60 LOC.
- [ ] **Step 4 — Run tests; verify pass** (residuals + emergent area).
- [ ] **Step 5 — build/vet/gofmt clean; corpus 55.**
- [ ] **Step 6 — Commit** (`feat(geom): variational plate fill → BSpline (PlateFill)`).

---

### Task P4b: plateProvider Build + reflected-family rails (no tuned E2)

**Files:**
- Modify: `kernel/ops/corner_provider_plate.go` (replace the P0 stub Build)
- Modify: `kernel/ops/corner_extract_tangent.go` (delete `wallBridgeFullness=1.136`; E2 = standard
  chord-based Hermite, no scalar)
- Test: `kernel/ops/corner_provider_plate_test.go` (extend)

**Interfaces:**
- Consumes: `PlateFill`/`PlateSide` (P4a), `RailLoop`/`Side` (corner_rail.go), the coons4/sphere certify
  helpers (`certifyCoons4Patch` in corner_provider_coons4.go).
- Produces: `plateProvider.Build(l RailLoop, res Resolution) (CornerBlendPatch, Certificate, bool)` —
  map `l.Sides` → `[4]PlateSide` → `PlateFill` → certify (reuse) → `CornerBlendPatch{Surface, Loops,
  Kind: BlendKindPlate}`; honest-reject (return false) on any error ⇒ resolveBlend falls through to coons4.

- [ ] **Step 1 — Failing test:** N7's marked loop → `resolveBlend` returns `BlendKindPlate` with a valid
  Certificate + emergent area 90.194 within tol AND E2 on-wall witnessed independently
  (`|dist(wallAxis)−R| ≤ res.Weld·R`); a NON-marked valence-4 loop → still `BlendKindCoons4` (unchanged).
  Delete the `wallBridgeFullness` test that asserted the tuned area; replace E2 construction with the
  standard chord-based Hermite (no free scalar) and assert the area is EMERGENT.
- [ ] **Step 2 — Run, verify it fails** (stub Build declines; tuned E2 removed).
- [ ] **Step 3 — Implement** the real Build + the standard-Hermite E2; delete `wallBridgeFullness`.
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — B3 reduction:** clean octant (valence-3, unmarked) → `BlendKindSphere`; B3
  golden/weld/volume green. Corpus still `55` (N7's whole-body weld still declines upstream at wall
  termination — that is T-N7.3, out of this milestone; the corner FILL now certifies via plate).
  build/vet/gofmt/golangci-lint clean.
- [ ] **Step 6 — Commit** (`feat(ops): plateProvider Build + reflected-family rails (no tuned constant)`).

---

### Task P5: DRAWEXE gate + green N7 (corpus 55→56)

**Files:**
- Modify: `model/feature/occt_blend_simple_test.go` (N7 assertions)
- Modify: `model/feature/occtparity/corpus.json` (N7 already present; confirm expectedArea 61222.9)
- Modify/reconcile: `test-utilities/occt-blend/oracle/` N7 recipe (fix the mislabeled `CFI_f1234fim.rle`)

**Interfaces:**
- Consumes: the full N7 build path (`occtparity/runcase.go`: import `simple/N7.step` →
  `AddFilletSets` → `Recompute`); `plateProvider` (P4b); the DRAWEXE oracle.

- [ ] **Step 1 — Reconcile the fixture:** confirm via DRAWEXE on `simple/N7.step` that `result_5`
  area=90.194 and whole-body Σ=61222.9; fix the vendored `CFI_f1234fim.rle` recipe (wrong 8-face shape)
  so the manual oracle recipe matches the corpus fixture. Record the DRAWEXE output.
- [ ] **Step 2 — Failing (whole-body) test:** N7 whole-body Σ=61222.9 + per-face result_5=90.194
  (emergent) + all 12 faces + watertight + volume + E2-on-wall + G1-to-wall witnesses. Runs RED (T-N7.3
  wall termination + T-N7.4 far-runout still needed for the whole-body weld). **If T-N7.3/T-N7.4 are
  required to green the whole body, dispatch them per the corner-fill plan `2026-07-16-n7-tangent-
  degenerate-corner-fill.md` (they are unchanged by the plate work — they weld the plate patch to the
  wall/runout).**
- [ ] **Step 3 — Implement** the wall termination + far-runout (T-N7.3/T-N7.4) welding the plate patch,
  reusing the exact analytic E2 rail for the wall weld (NOT coons4's ~1e-4-off re-projection).
- [ ] **Step 4 — Run; verify N7 greens.** Corpus prints `56`.
- [ ] **Step 5 — Non-regression:** M1–M4 tripwire + whole corpus byte-identical except N7; B3
  golden/weld/volume unchanged. Full `go test ./...` + `golangci-lint run` + `gofmt -l` + markdownlint
  clean. Coverage > 80% on touched packages; duplication < 3%.
- [ ] **Step 6 — Commit** (`feat(ops): green OCCT blend/simple/N7 via variational plate corner fill (corpus 55→56)`).

---

## Verification

- **N7:** whole-body 61222.9 + result_5 90.194 **emergent** (functional of the fixed rails + energy, no
  free scalar); corpus 55→56; DRAWEXE-oracle-confirmed.
- **B3 byte-faithful:** octant valence-3 unmarked → analyticSphere wins → plate never runs; B3
  golden/weld/volume + corpus subtest unchanged.
- **Whole corpus byte-identical except N7:** the marker defaults zero; coons4/analyticSphere/tri3
  behaviour unchanged; every other extractor leaves Signature==0.
- **Honest gate:** NO tuned scalar on the plate path; the area gate checks, never fits; E2 on-wall +
  G1-to-wall witnessed independently of area.
- **Engine purity:** `geom/plate_*.go` import only `math`; `ops/corner_provider_plate.go` import geom+math,
  no `topo`; marker is the sole extractor→provider channel.
- **Before any PR (per CLAUDE.md):** full local suite + golangci-lint + markdownlint, coverage > 80% /
  duplication < 3%, SPDX check, cross-platform build; live MCP-bridge test placing/filleting N7 +
  screenshot; `Closes` the N7 tracking issue. **But NO PR until the whole corpus is green.**

## Escalations & risks (decided in-plan, never by tuning)

- **prescribe-E2 vs emerge-E2 (P4a Step 1 / P5):** prescribe-E2 (standard chord-based Hermite on-wall, no
  scalar) is primary — watertight by construction, area emergent. If its area misses `res.Weld·r²` of
  90.194, escalate to emerge-E2 (constrain only the 3 arm rails G1, leave the wall a plate contour, solve
  the fair interior, then intersect + same-parameter to imprint the exact wall edge — OCCT-literal, exact
  area). Decided at P4a/P5, NEVER by tuning a constant.
- **RBF conditioning (P2):** the dense `r⁴log r` system at ~40–120 constraints is the real numerical risk;
  geometry-math-advisor guards it (ridge vs pivoting); honest-reject on non-convergence → coons4 fallback.
- **T-N7.3/T-N7.4 (P5):** the wall-termination + far-runout welds are unchanged by the plate work; they
  weld the plate patch. Their wall weld must use the exact analytic E2 rail (a carried Minor: coons4
  rebuilds E2 ~1e-4 off-wall).
- **Carried Minors for the final whole-branch review** (from the corner-fill ledger): reject comments
  missing offending values; `curvedCornerConsistent` two-sided relaxation on `len(arms)==0`; the
  byte-identity golden + extraction test are a pair; hard-coded test constants.

## References
- `.superpowers/sdd/n7-plate-seam-architecture.md` — ADR-1/2/3, the seam contract, the 7-task shape.
- `.superpowers/sdd/n7-plate-solve-derivation.md` — the Duchon math, prescribe-vs-emerge E2, honest area.
- `.superpowers/sdd/n7-fill-rails-rederivation.md` — the reflected-family rails (kept, DRAWEXE-validated).
- `docs/superpowers/specs/2026-07-16-degenerate-corner-plate-fill-design.md` — the M6 spec.
- In-repo ADR-0051 (provider tiers), ADR-0042 (relative tolerances), ADR-0018 (public-API split, not
  triggered).
- Duchon (1977); Wahba, *Spline Models for Observational Data* (1990); OCCT `Plate_Plate` /
  `GeomPlate_BuildPlateSurface` / `GeomPlate_MakeApproxSurface` (reference implementation).
