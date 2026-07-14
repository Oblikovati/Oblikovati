<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Corner Extractor Wave — Tracer Bullet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove the generalized corner-blend seam end-to-end by routing junctions through one `ExtractRailLoop → resolveBlend` facade — correcting the latent obstacle ribbon-sign fold, holding the green planar-trihedral + obstacle cases byte-for-byte, then greening the five runout cases (S1/S4/T1/T7/T9).

**Architecture:** Strangler pattern behind the existing `assembleFilletBody` do-no-harm fallback. Milestone 1 corrects the obstacle ribbon sign (F2), builds the runtime dual-path probe that gates it, de-dups the certify helpers (F3), then wires `extractTrihedral`/`extractObstacle` → `resolveBlend` proving byte-for-byte. Milestone 2 adds `extractRunout` (a single adapter over the already-shipped `detectRunouts`/`solveImprint`) → 4-sided `RailLoop` → `coons4`, gated per family by the DRAWEXE area oracle.

**Tech Stack:** Go (package `ops` under `kernel/ops/`), `oblikovati.org/kernel/geom` (NURBS/analytic surfaces, `FillSurface`/`MatchSurface`), `oblikovati.org/math`, `oblikovati.org/kernel/topo`. Oracle: OCCT DRAWEXE 8.0.0. Corpus harness: `model/feature` `TestOCCTBlendSimple`.

## Global Constraints

- **NO PR until the whole `tests/blend` corpus is green.** Accumulate + commit per task on branch `feat/occt-blend-parity-corpus`. Never regress the green corpus.
- SPDX `GPL-2.0-only` header (first line) on every new `.go`; run `scripts/add-spdx-headers.py`.
- Functions **4–20 lines**; files **< 500 lines**; explicit types (no `any`); early returns; **≤ 2 levels of indent**; no code duplication; error messages carry the offending value + expected shape.
- **TDD**, named fakes (never inline stubs). Every new function gets a test; every fix gets a regression test.
- **Tolerances are model-relative** — `ResolutionForPoints(pts).Weld()` / `.Size()`, **never a bare `1e-6`**.
- **Providers/extractors must not import `topo` for geometry** — they depend on `geom`+`math` only. (`RailLoop.Provenance topo.Lineage` is the sole `topo` touch, carried opaquely.)
- **Corpus-neutrality gate** (run after every task that could touch a live path):
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` → MUST print **≥ 50**, byte-identical on untouched cases.
- **Per-family oracle:** DRAWEXE at `occt-build/lin64/gcc/bin/DRAWEXE`, run via `printf 'source X.tcl\n' | DRAWEXE -b` (never `-b file`, never line-by-line).
- Before any PR (end of the whole corpus, out of scope here): live MCP-bridge test + screenshot.

## Scope Check

This plan covers **Milestone 1 (F2/F3 + strangler seam)** and **Milestone 2 (runout tracer)** of `docs/superpowers/specs/2026-07-14-corner-extractor-wave-design.md`. **Milestone 3 (curved miter + N-way, ~69 cases) is DEFERRED to a follow-up plan** authored once this tracer proves the pipeline — it is a distinct, larger vertical whose extractor (`extractMiter`) and setback generalization warrant their own oracle-gated tasks.

The **foundation wave is DONE and OUT OF SCOPE**: `corner_rail.go` (`Side`/`RailLoop`), `analyticSphereProvider`/`coons4Provider`/`tri3Provider`, `resolveBlend`/`blendTiers`/`resolveBlendWith` are all shipped (base `7cad7cb2` → HEAD `2079f571`). Do not re-plan them.

**M1→M2 binding:** Milestone 2's `extractRunout` consumes the strangler facade and the `patchToFilletFace` adapter defined in Milestone 1, Task 6. Its Interfaces block names the exact signatures.

---

## File Structure

**New files:**
- `kernel/ops/corner_ribbon_probe.go` — the runtime dual-path probe (F2 gate instrument): oriented-normal seam agreement + `NoFold`, model-scaled degeneracy abstention. `corner_ribbon_probe_test.go`.
- `kernel/ops/corner_extract_trihedral.go` — `extractTrihedral` (`*cornerBlend` → 3-sided `RailLoop`) + the sphere-patch strangler entry. `corner_extract_trihedral_test.go`.
- `kernel/ops/corner_extract_obstacle.go` — `extractObstacle` (`*ObstacleFeature` → 4-sided `RailLoop`) + the obstacle strangler entry. `corner_extract_obstacle_test.go`.
- `kernel/ops/corner_extract_runout.go` — `extractRunout` (`runoutImprint` + `imprintCut` → 4-sided `RailLoop`). `corner_extract_runout_test.go`.
- `kernel/ops/corner_patch_adapter.go` — `patchToFilletFace` (`CornerBlendPatch` → `filletFace`) shared by the three strangler entries. `corner_patch_adapter_test.go`.

**Modified files:**
- `kernel/ops/corner_blend_obstacle.go:142-144` — flip the wall + two wings to the outward `awayRef` reference (F2 sign correction).
- `kernel/ops/corner_blend_obstacle_certify.go` — `obstacleNoFold` delegates to a shared `noFoldOverColumns` (F3).
- `kernel/ops/corner_provider_tri3.go` — `tri3NoFold` delegates to `noFoldOverColumns`; `tri3RibLen`/`coons4RibLen` collapse into `loopRibLen` (F3).
- `kernel/ops/corner_provider_coons4.go` — `coons4RibLen` → `loopRibLen` (F3).
- `kernel/ops/fillet_faces.go` — `spherePatchFace` routes its sphere recognition through `extractTrihedral` (strangler, byte-for-byte loop preserved).
- `kernel/ops/fillet.go` — the obstacle/junction dispatch calls the strangler entry behind the do-no-harm fallback.

---

## MILESTONE 1 — F2 sign correction, F3 de-dup & the Strangler Migration

### Task 1: Runtime dual-path probe instrument (F2 gate)

**Files:**
- Create: `kernel/ops/corner_ribbon_probe.go`
- Test: `kernel/ops/corner_ribbon_probe_test.go`

**Interfaces:**
- Consumes: `geom.BSplineSurface`, `geom.CoonsFill(c0,c1,d0,d1) (BSplineSurface, error)`, `geom.FillSide{Adjacent BSplineSurface; AdjEdge Boundary; Order int}`, `fillEdge`/`edgeVMin…edgeUMax` + `coons4Edges() [4]fillEdge`, `inwardCrossV(s, atMax) math.Vector3` / `inwardCrossU(s, atMax) math.Vector3` (corner_blend_obstacle.go), `obstacleNoFold(s, scale) bool`, `Resolution.Weld()`.
- Produces: `ribbonSeamNonFolding(fill geom.BSplineSurface, rails [4]geom.BSplineCurve, sides [4]geom.FillSide, scale Resolution) bool`.

This is the instrument that catches the exact defect `creaseAngle` masks — a ribbon whose tangent plane is right but whose *oriented* normal is flipped (§5 of the spec).

**⚠ CORRECTED after Task-1 first attempt (a provably-blind method was caught):** the advisor's report and the earlier draft of this task compared the fill normal to the ribbon normal at the seam (`n̂_fill·n̂_rib`). That is **tautological**: for a VMin↔VMin Order-1 match the operator forces `F_v(boundary) = −dir` *exactly*, so `nf = −nr` **identically for both orientations** — the test is blind to the fold (empirically verified: identical dot on the correct and the inverted fixture). The **real discriminator** is the advisor's check #2: compare the MATCHED fill's into-patch cross-derivative against the **base Coons** interior cross-derivative at the same edge. The base depends only on rail *positions* (ribbon-independent), so matched-vs-base flips sign with the ribbon orientation. It is boundary-local and exact — it catches even T6's shallow fold with no 24×24 sampling luck. For each G1 side (`Order > 0`), assert `matched_cross · base_cross > 0`, then that the whole patch passes `NoFold`.

- [ ] **Step 1: Write the failing test**

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// TestRibbonSeamNonFoldingAcceptsOutwardCoons4 proves the probe passes the shipped, correct
// coons4 patch (outward ribbons) built from the quarter-cylinder fixture.
func TestRibbonSeamNonFoldingAcceptsOutwardCoons4(t *testing.T) {
	loop := quarterCylLoop(t, 4)
	fill, rails, sides, ok := coons4Fill(loop)
	if !ok {
		t.Fatal("coons4Fill declined the quarter-cyl fixture")
	}
	if !ribbonSeamNonFolding(fill, rails, sides, blendScale()) {
		t.Fatal("probe rejected the correct outward-ribbon coons4 patch")
	}
}

// TestRibbonSeamNonFoldingRejectsInwardRibbon proves the probe FLAGS a deliberately inward-signed
// ribbon (the latent obstacle defect) — the sign check creaseAngle cannot see.
func TestRibbonSeamNonFoldingRejectsInwardRibbon(t *testing.T) {
	loop := quarterCylLoop(t, 4)
	c0, c1, d0, d1, ok := loopRails(loop)
	if !ok {
		t.Fatal("loopRails declined the fixture")
	}
	c0, c1, d0, d1, _ = refineForG1(c0, c1, d0, d1)
	rails := [4]geom.BSplineCurve{c0, c1, d0, d1}
	base, err := geom.CoonsFill(rails[0], rails[1], rails[2], rails[3])
	if err != nil {
		t.Fatalf("CoonsFill: %v", err)
	}
	// INWARD ribbons: the negation of the shipped outward awayRef (the bug we are guarding against).
	length := loopRibLen(loop)
	inward := invertedCoons4Sides(loop, rails, base, length)
	fill, err := geom.FillSurface(rails[0], rails[1], rails[2], rails[3], inward)
	if err != nil {
		t.Fatalf("FillSurface: %v", err)
	}
	fill, _ = pinFillBoundary(fill, rails[0], rails[1], rails[2], rails[3])
	if ribbonSeamNonFolding(fill, rails, inward, blendScale()) {
		t.Fatal("probe accepted an inward-signed (folded) ribbon patch")
	}
}

// invertedCoons4Sides mirrors coons4Sides but anchors on the INWARD cross-derivative (no Scale(-1)),
// producing the folded patch under test. Named test helper (not an inline stub).
func invertedCoons4Sides(loop RailLoop, rails [4]geom.BSplineCurve, base geom.BSplineSurface, length float64) [4]geom.FillSide {
	fs0, _ := ribbonSide(rails[0], loop.Sides[0], inwardCrossV(base, false), length)
	fs1, _ := ribbonSide(rails[1], loop.Sides[2], inwardCrossV(base, true), length)
	fs2, _ := ribbonSide(rails[2], loop.Sides[3], inwardCrossU(base, false), length)
	fs3, _ := ribbonSide(rails[3], loop.Sides[1], inwardCrossU(base, true), length)
	return [4]geom.FillSide{fs0, fs1, fs2, fs3}
}
```

`length := coons4RibLen(loop)` in the RejectsInward test compiles today; add `// TODO(Task 3): loopRibLen` and switch when Task 3 lands.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestRibbonSeamNonFolding -v`
Expected: FAIL — `undefined: ribbonSeamNonFolding` / `undefined: invertedCoons4Sides`.

- [ ] **Step 3: Write the implementation**

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// ribbonSeamNonFolding is the F2 runtime probe. For every G1 side it asserts the MATCHED fill's
// into-patch cross-derivative still agrees with the BASE Coons interior cross-derivative there — the
// sign-sensitive test creaseAngle omits — then requires the whole patch to pass the anti-fold sweep.
// It reconstructs the base from the rails (identical to the base used during the match). G0 /
// ribbon-less sides are skipped.
//
// WHY NOT compare the fill normal to the ribbon normal: for a VMin↔VMin Order-1 match the operator
// forces F_v(boundary) = −dir EXACTLY, so the fill normal nf = −nr IDENTICALLY for BOTH orientations —
// a boundary nf·nr test is tautological, blind to the fold (proven + empirically confirmed in the F2
// wave). The base's interior cross-derivative depends only on rail POSITIONS (ribbon-independent), so
// matched-vs-base DOES flip sign with the ribbon orientation and is the real discriminator.
func ribbonSeamNonFolding(fill geom.BSplineSurface, rails [4]geom.BSplineCurve, sides [4]geom.FillSide, scale Resolution) bool {
	base, err := geom.CoonsFill(rails[0], rails[1], rails[2], rails[3])
	if err != nil {
		return false
	}
	edges := coons4Edges()
	for i, e := range edges {
		if sides[i].Order <= 0 {
			continue
		}
		if !matchedCrossPointsInward(fill, base, e, scale) {
			return false
		}
	}
	return obstacleNoFold(fill, scale)
}

// matchedCrossPointsInward compares the matched fill's into-patch cross-derivative at edge e's
// midpoint against the base Coons interior cross-derivative there. Correct (outward) ribbon: the
// matched cross-derivative lands back inside the patch, agreeing with base (dot > 0). Inward/folded
// ribbon: it flips (dot < 0). Boundary-local and exact — catches even a shallow fold (no 24×24
// sampling luck). Abstains (true) when either derivative is degenerate below the model-scaled weld floor.
func matchedCrossPointsInward(fill, base geom.BSplineSurface, e fillEdge, scale Resolution) bool {
	cf, cb := inwardCrossAt(fill, e), inwardCrossAt(base, e)
	if cf.Length() < scale.Weld() || cb.Length() < scale.Weld() {
		return true
	}
	return cf.Dot(cb) > 0
}

// inwardCrossAt returns the into-patch cross-derivative at fill edge e's midpoint, reusing the
// obstacle certify sign convention (+∂/∂v at VMin, −∂/∂v at VMax, +∂/∂u at UMin, −∂/∂u at UMax) so it
// matches the awayRef anchor coons4Sides/obstacleSides build against.
func inwardCrossAt(s geom.BSplineSurface, e fillEdge) math.Vector3 {
	switch e {
	case edgeVMin:
		return inwardCrossV(s, false)
	case edgeVMax:
		return inwardCrossV(s, true)
	case edgeUMin:
		return inwardCrossU(s, false)
	default: // edgeUMax
		return inwardCrossU(s, true)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestRibbonSeamNonFolding -v`
Expected: PASS (both subtests).

- [ ] **Step 5: Corpus-neutral gate + build**

Run: `go build ./kernel/... && go vet ./kernel/ops/ && gofmt -l kernel/ops/corner_ribbon_probe.go kernel/ops/corner_ribbon_probe_test.go`
Expected: empty output.
Run: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
Expected: `50` (probe is dead code from the corpus's view — nothing wired yet).

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_ribbon_probe.go kernel/ops/corner_ribbon_probe_test.go
git commit -m "feat(blend): F2 runtime dual-path probe (oriented-normal seam agreement)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Correct the obstacle ribbon sign (the latent-fold fix)

**Files:**
- Modify: `kernel/ops/corner_blend_obstacle.go:142-144`
- Test: `kernel/ops/corner_blend_obstacle_test.go` (add cases)

**Interfaces:**
- Consumes: `obstaclePatchNeighbours(of) (obstaclePatchGeom, bool)`, `obstacleSides(of, wingL, wingR, wall) [4]geom.FillSide`, `geom.FillSurface`, `pinFillBoundary`, `ribbonSeamNonFolding` (Task 1), `newT6Obstacle(t)`/`newFoldingObstacle(t)` fixtures, `blendScale()`.
- Produces: the sign-corrected obstacle baseline (behavioral; no new exported symbol).

The F2 derivation (`scratchpad/f2-reconciliation-report.md`) proved the obstacle's inward anchor (`inwardCrossV(base,false)`, `inwardCrossU(base,false)`, `inwardCrossU(base,true)`) is the **wrong** sign — masked by antipodal-blind `creaseAngle` + 24×24 sampling luck. Flip the three anchors to the **outward** `awayRef` reference (`.Scale(-1)`), matching `coons4Sides`.

- [ ] **Step 1: Write the failing test**

```go
// TestObstacleT6RibbonNonFolding proves the sign-corrected obstacle patch passes the F2 probe.
// Before the fix this FAILS on the wall seam — that failure is the regression witness the report
// predicts (f2-reconciliation-report.md §C, "before the flip this assertion is expected to FAIL").
func TestObstacleT6RibbonNonFolding(t *testing.T) {
	of := newT6Obstacle(t)
	g, ok := obstaclePatchNeighbours(of)
	if !ok {
		t.Fatal("obstaclePatchNeighbours declined T6")
	}
	sides := obstacleSides(of, g.wingL, g.wingR, g.wall)
	rails := [4]geom.BSplineCurve{g.c0, g.c1, g.d0, g.d1}
	fill, err := geom.FillSurface(g.c0, g.c1, g.d0, g.d1, sides)
	if err != nil {
		t.Fatalf("FillSurface: %v", err)
	}
	fill, err = pinFillBoundary(fill, g.c0, g.c1, g.d0, g.d1)
	if err != nil {
		t.Fatalf("pinFillBoundary: %v", err)
	}
	if !ribbonSeamNonFolding(fill, rails, sides, blendScale()) {
		t.Fatal("sign-corrected obstacle T6 patch still folds")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestObstacleT6RibbonNonFolding -v`
Expected: FAIL — the probe flags the inward-signed wall seam (the masked fold, now resolved).

- [ ] **Step 3: Apply the one-flip correction**

In `corner_blend_obstacle.go`, `obstaclePatchNeighbours` (lines 142-144), negate each inward anchor to the outward reference:

```go
	wall, e0 := extrudeRibbon(c0, orientInward(of.WallInto.Scale(length), inwardCrossV(base, false).Scale(-1)))
	wingL, e1 := extrudeRibbon(d0, orientInward(wingDir(of, of.Nodes[0], length), inwardCrossU(base, false).Scale(-1)))
	wingR, e2 := extrudeRibbon(d1, orientInward(wingDir(of, of.Nodes[1], length), inwardCrossU(base, true).Scale(-1)))
```

Update the `orientInward` doc comment (corner_blend_obstacle.go:151-157) to state the anchor is now the **outward** cross-derivative (matching `coons4Sides`), and reference this task + `f2-reconciliation-report.md` as the provenance for the flip.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestObstacle -v`
Expected: PASS (the new probe test + all existing obstacle tests — `newFoldingObstacle`-based tests included).

- [ ] **Step 5: Corpus gate (the re-baseline)**

Run: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
Expected: `50`. The flip is a *correctness* fix; the corpus compares area to the OCCT external oracle, so a more-correct obstacle stays green. If any obstacle case drops, STOP — the flip is not the whole story on that case; investigate before proceeding (do not mask with the fallback).

This run establishes the **corrected obstacle baseline** the strangler (Task 5/6) diffs against.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_blend_obstacle.go kernel/ops/corner_blend_obstacle_test.go
git commit -m "fix(blend): correct obstacle ribbon sign inward->outward (F2 latent fold)

The obstacle wall+wings anchored on the inward cross-derivative -- the wrong sign,
masked by antipodal-blind creaseAngle + 24x24 sampling luck (f2-reconciliation-report.md).
Flip to the outward awayRef reference matching coons4Sides; the new probe on T6 is the
regression witness. Corpus stays green (area vs OCCT oracle). Re-baselines obstacle.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: F3 — de-duplicate the certify helpers

**Files:**
- Modify: `kernel/ops/corner_blend_obstacle_certify.go` (`obstacleNoFold`), `kernel/ops/corner_provider_tri3.go` (`tri3NoFold`, `tri3RibLen`), `kernel/ops/corner_provider_coons4.go` (`coons4RibLen`)
- Test: `kernel/ops/corner_blend_obstacle_certify_test.go` (add), reuse existing tri3/coons4 tests

**Interfaces:**
- Consumes: `columnFolds(s, u, v0, v1, scale) bool` (already shared), `loopCorners`, `RailLoop.Sides`, `ResolutionForPoints`, `ribbonSpanFactor`, `obstacleFoldSamples`, `poleExcl`.
- Produces: `noFoldOverColumns(s geom.BSplineSurface, vLo, vHi float64, scale Resolution) bool`, `loopRibLen(loop RailLoop) float64`.

`obstacleNoFold` and `tri3NoFold` share the identical u-column sweep, differing only in the v-upper bound (tri3 caps at `v1 − poleExcl·span` to exclude the pole). `coons4RibLen`/`tri3RibLen` are the identical model-relative span×factor over the loop corners. Extract one helper each; no behavior change.

- [ ] **Step 1: Write the failing test**

```go
// TestNoFoldOverColumnsMatchesObstacle proves the extracted sweep reproduces obstacleNoFold's
// verdict on the folding + non-folding fixtures (behavior-preserving refactor).
func TestNoFoldOverColumnsMatchesObstacle(t *testing.T) {
	of := newT6Obstacle(t)
	g, _ := obstaclePatchNeighbours(of)
	sides := obstacleSides(of, g.wingL, g.wingR, g.wall)
	fill, _ := geom.FillSurface(g.c0, g.c1, g.d0, g.d1, sides)
	fill, _ = pinFillBoundary(fill, g.c0, g.c1, g.d0, g.d1)
	v0, v1 := fill.VDomain()
	if noFoldOverColumns(fill, v0, v1, blendScale()) != obstacleNoFold(fill, blendScale()) {
		t.Fatal("noFoldOverColumns disagrees with obstacleNoFold on T6")
	}
}

// TestLoopRibLenMatchesValence proves the unified rib length equals the old per-valence helpers.
func TestLoopRibLenMatchesValence(t *testing.T) {
	q := quarterCylLoop(t, 4)
	if loopRibLen(q) != coons4RibLenLegacyForTest(q) {
		t.Fatalf("loopRibLen(valence4) = %g, want legacy value", loopRibLen(q))
	}
}

// coops the pre-refactor coons4RibLen formula so the test pins equality (named helper, deleted with
// the test once the refactor is proven).
func coons4RibLenLegacyForTest(loop RailLoop) float64 {
	a, b, c, d := loopCorners(loop)
	return ResolutionForPoints([]math.Point3{a, b, c, d}).Size() * ribbonSpanFactor
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run 'TestNoFoldOverColumns|TestLoopRibLen' -v`
Expected: FAIL — `undefined: noFoldOverColumns`, `undefined: loopRibLen`.

- [ ] **Step 3: Extract the shared helpers**

In `corner_blend_obstacle_certify.go`, add and re-point `obstacleNoFold`:

```go
// noFoldOverColumns is the shared anti-fold column sweep: it scans obstacleFoldSamples u-columns over
// [vLo,vHi] and reports true iff no column folds (columnFolds). obstacleNoFold uses the full v-range;
// tri3NoFold caps vHi below the pole. (F3 de-dup of the obstacle/tri3 sweeps.)
func noFoldOverColumns(s geom.BSplineSurface, vLo, vHi float64, scale Resolution) bool {
	u0, u1 := s.UDomain()
	for i := 0; i <= obstacleFoldSamples; i++ {
		u := u0 + float64(i)/float64(obstacleFoldSamples)*(u1-u0)
		if columnFolds(s, u, vLo, vHi, scale) {
			return false
		}
	}
	return true
}

func obstacleNoFold(s geom.BSplineSurface, scale Resolution) bool {
	v0, v1 := s.VDomain()
	return noFoldOverColumns(s, v0, v1, scale)
}
```

(Match the existing `obstacleNoFold` u-sampling loop exactly — read lines 58-75 and preserve the sample count/bounds; the code above assumes the `0..obstacleFoldSamples` inclusive stride the file already uses. If the existing loop differs, keep ITS stride and move only the body.)

In `corner_provider_tri3.go`, re-point `tri3NoFold`:

```go
func tri3NoFold(fill geom.BSplineSurface, scale Resolution) bool {
	v0, v1 := fill.VDomain()
	return noFoldOverColumns(fill, v0, v1-poleExcl*(v1-v0), scale)
}
```

Add the unified rib length in `corner_provider_coons4.go` (next to `coons4RibLen`), then delete `coons4RibLen` and `tri3RibLen`, re-pointing their call sites:

```go
// loopRibLen is the model-relative ribbon length for a loop of ANY valence: a small fraction of the
// loop's bounding span over ALL side-start corners (ADR-0042). Replaces coons4RibLen/tri3RibLen.
func loopRibLen(loop RailLoop) float64 {
	pts := make([]math.Point3, loop.Valence())
	for i, s := range loop.Sides {
		pts[i] = curveStart(s.Curve)
	}
	return ResolutionForPoints(pts).Size() * ribbonSpanFactor
}
```

Re-point: `coons4Sides` uses `length := loopRibLen(loop)` (was `coons4RibLen`); the tri3 ribbon builder uses `loopRibLen` (was `tri3RibLen`). Grep `coons4RibLen`/`tri3RibLen` and replace all.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./kernel/ops/ -run 'TestNoFoldOverColumns|TestLoopRibLen|Tri3|Coons4|Obstacle|TestResolveBlend' -v`
Expected: PASS (all).

- [ ] **Step 5: Full package + corpus gate**

Run: `go test ./kernel/ops/ && go build ./kernel/... && go vet ./kernel/ops/`
Run: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
Expected: `50`.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_blend_obstacle_certify.go kernel/ops/corner_provider_tri3.go kernel/ops/corner_provider_coons4.go kernel/ops/corner_blend_obstacle_certify_test.go
git commit -m "refactor(blend): F3 de-dup certify helpers (noFoldOverColumns, loopRibLen)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: `patchToFilletFace` adapter

**Files:**
- Create: `kernel/ops/corner_patch_adapter.go`
- Test: `kernel/ops/corner_patch_adapter_test.go`

**Interfaces:**
- Consumes: `CornerBlendPatch{Surface geom.Surface; Loops []filletLoop; Kind CornerBlendKind}`, `filletFace{surface geom.Surface; loops []filletLoop; parent topo.Lineage}`, `topo.Lineage`.
- Produces: `patchToFilletFace(patch CornerBlendPatch, parent topo.Lineage) filletFace`.

The strangler entries (Tasks 5, 6; Milestone 2) all turn a `resolveBlend` patch into the `filletFace` the hardened assembly consumes. One adapter, so the three paths never drift.

- [ ] **Step 1: Write the failing test**

```go
func TestPatchToFilletFaceCarriesSurfaceLoopsParent(t *testing.T) {
	sph, _ := geom.NewSphere(math.Point3{X: 1, Y: 2, Z: 3}, 5)
	loops := []filletLoop{{pts: []math.Point3{{X: 0}, {X: 1}, {X: 1, Y: 1}}}}
	patch := CornerBlendPatch{Surface: sph, Loops: loops, Kind: BlendKindSphere}
	lin := topo.Lineage{}
	f := patchToFilletFace(patch, lin)
	if f.surface != geom.Surface(sph) {
		t.Fatal("surface not carried")
	}
	if len(f.loops) != 1 || len(f.loops[0].pts) != 3 {
		t.Fatalf("loops not carried: %+v", f.loops)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestPatchToFilletFace -v`
Expected: FAIL — `undefined: patchToFilletFace`.

- [ ] **Step 3: Write the implementation**

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import "oblikovati.org/kernel/topo"

// patchToFilletFace adapts a resolveBlend result into the filletFace the hardened assembly consumes
// (assembleBody). The provider owns the surface + boundary loops; the extractor supplies the lineage
// (ADR-0043) since providers are topo-free. One adapter for every strangler entry so they never drift.
func patchToFilletFace(patch CornerBlendPatch, parent topo.Lineage) filletFace {
	return filletFace{surface: patch.Surface, loops: patch.Loops, parent: parent}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestPatchToFilletFace -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/corner_patch_adapter.go kernel/ops/corner_patch_adapter_test.go
git commit -m "feat(blend): patchToFilletFace adapter (CornerBlendPatch -> filletFace)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: `extractObstacle` — ObstacleFeature → 4-sided RailLoop

**Files:**
- Create: `kernel/ops/corner_extract_obstacle.go`
- Test: `kernel/ops/corner_extract_obstacle_test.go`

**Interfaces:**
- Consumes: `ObstacleFeature`, `obstacleRails(of) (c0,c1,d0,d1 geom.BSplineCurve, ok bool)`, `Side{Curve; Adjacent; Cont}`, `RailLoop{Sides; Provenance}`, `G0`/`G1`, the wall/wing/rim `geom.Surface` neighbours from `obstaclePatchNeighbours`.
- Produces: `extractObstacle(of *ObstacleFeature) (RailLoop, bool)`.

Build the 4-sided loop in the coons4 Side order (`s0=wall/c0`, `s1=wingR/d1`, `s2=rim/c1`, `s3=wingL/d0`) so `resolveBlend` → `coons4Provider` reproduces the corrected obstacle patch. Adjacent surfaces are the ribbon-generating neighbours; rim is G0, wall+wings are G1.

- [ ] **Step 1: Write the failing test**

```go
func TestExtractObstacleIsClosedValence4(t *testing.T) {
	of := newT6Obstacle(t)
	loop, ok := extractObstacle(of)
	if !ok {
		t.Fatal("extractObstacle declined T6")
	}
	if loop.Valence() != 4 {
		t.Fatalf("valence = %d, want 4", loop.Valence())
	}
	if !loop.Closed(blendScale().Weld()) {
		t.Fatal("loop not closed")
	}
}

// TestExtractObstacleResolvesToCoons4 proves the extracted loop fills via the general coons4 tier and
// passes the F2 probe (the corrected, non-folding sign).
func TestExtractObstacleResolvesToCoons4(t *testing.T) {
	of := newT6Obstacle(t)
	loop, _ := extractObstacle(of)
	fill, rails, sides, ok := coons4Fill(loop)
	if !ok {
		t.Fatal("coons4Fill declined the extracted obstacle loop")
	}
	if !ribbonSeamNonFolding(fill, rails, sides, blendScale()) {
		t.Fatal("extracted obstacle loop folds under coons4")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestExtractObstacle -v`
Expected: FAIL — `undefined: extractObstacle`.

- [ ] **Step 3: Write the implementation**

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// extractObstacle turns a mid-span ObstacleFeature into the 4-sided RailLoop the general coons4 tier
// fills. The Side order mirrors the coons4 mapping (s0=wall, s1=wingR, s2=rim, s3=wingL) so resolveBlend
// reproduces the (sign-corrected) obstacle patch. The wall and both wings are G1 to their neighbour
// surfaces; the base rim is a G0 crease. ok=false on a bad rail → the caller honest-rejects (ADR-3).
func extractObstacle(of *ObstacleFeature) (RailLoop, bool) {
	c0, c1, d0, d1, ok := obstacleRails(of)
	if !ok {
		return RailLoop{}, false
	}
	wall, wingL, wingR, rim, ok := obstacleAdjacents(of)
	if !ok {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: c0, Adjacent: wall, Cont: G1},  // s0 wall  → c0/VMin
		{Curve: d1, Adjacent: wingR, Cont: G1}, // s1 wingR → d1/UMax
		{Curve: c1, Adjacent: rim, Cont: G0},   // s2 rim   → c1/VMax
		{Curve: d0, Adjacent: wingL, Cont: G1}, // s3 wingL → d0/UMin
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}
```

`obstacleAdjacents(of)` must return the four analytic neighbour surfaces (`geom.Surface`) the ribbons are built from: the wall plane, the two wing cylinders, and the rim's host plane. Derive them from the `ObstacleFeature` fields (`HostPlane` for the rim's G0 side; the wall plane from `WallLine`+`WallInto`; the wing cylinders from `WingStart`/`WingEnd`+`BlendAxis`+`Radius`). Implement `obstacleAdjacents` as a small helper (4-20 lines) reading those fields; return `ok=false` if a wing cannot be reconstructed. Read `corner_blend_obstacle.go` + `corner_blend_obstacle_rails.go` for how `obstacleSides`/`obstaclePatchNeighbours` already reconstruct the wall plane and wing cylinders, and reuse that construction rather than duplicating it. For a rim whose Adjacent is only used for G0 (no ribbon), `HostPlane` suffices.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestExtractObstacle -v`
Expected: PASS.

- [ ] **Step 5: SPDX + build + fmt**

Run: `go build ./kernel/... && go vet ./kernel/ops/ && gofmt -l kernel/ops/corner_extract_obstacle.go`
Expected: empty. Confirm the SPDX header is present (`scripts/add-spdx-headers.py`).

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_extract_obstacle.go kernel/ops/corner_extract_obstacle_test.go
git commit -m "feat(blend): extractObstacle (ObstacleFeature -> 4-sided RailLoop)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: `extractTrihedral` (planar) + sphere strangler wiring (golden diff)

**Files:**
- Create: `kernel/ops/corner_extract_trihedral.go`
- Modify: `kernel/ops/fillet_faces.go` (`spherePatchFace`)
- Test: `kernel/ops/corner_extract_trihedral_test.go`

**Interfaces:**
- Consumes: `cornerBlend{vertex; center; sphere geom.Sphere; tan; arcs []blendArc}`, `blendArc{ta,tb,mid; chords}`, `geom.Arc3d`, `geom.NewArcFrom3Points`-equivalent (see below), `resolveBlend(loop, scale) (CornerBlendPatch, bool)`, `chainArcs(arcs) filletLoop`, `spherePatchFlipped`/`reverseArcLoop`, `patchToFilletFace`.
- Produces: `extractTrihedral(cb *cornerBlend) (RailLoop, bool)`, and a `spherePatchFace` that routes recognition through the extractor while preserving the byte-identical loop.

The strangler's first arity: route the planar-trihedral sphere through `extractTrihedral → resolveBlend → analyticSphereProvider`, proving the RailLoop path recognizes the **same** sphere. **Byte-for-byte is preserved by keeping the boundary loop from `chainArcs(cb.arcs)` unchanged** (§2/§3 of the spec: the trim/arcs are the byte-for-byte risk, so the planar path reuses them — it does NOT recompute setbacks). The extractor validates the surface; the loop is untouched.

- [ ] **Step 1: Write the failing test**

```go
// TestExtractTrihedralRecognizesSphere proves the extracted 3-arc loop is claimed by the exact sphere
// tier and yields the SAME sphere as the cornerBlend (center + radius within weld).
func TestExtractTrihedralRecognizesSphere(t *testing.T) {
	cb := sphereCornerBlendFixture(t, 4) // named fixture: a 3-arc planar-trihedral cornerBlend, r=4
	loop, ok := extractTrihedral(cb)
	if !ok {
		t.Fatal("extractTrihedral declined the planar trihedral")
	}
	if loop.Valence() != 3 {
		t.Fatalf("valence = %d, want 3", loop.Valence())
	}
	patch, ok := resolveBlend(loop, blendScale())
	if !ok || patch.Kind != BlendKindSphere {
		t.Fatalf("resolveBlend kind = %q ok=%v, want analytic-sphere", patch.Kind, ok)
	}
	got := patch.Surface.(geom.Sphere)
	if got.Center.DistanceTo(cb.sphere.Center) > blendScale().Weld() || abs(got.Radius-cb.sphere.Radius) > blendScale().Weld() {
		t.Fatalf("recognized sphere %+v != cornerBlend sphere %+v", got, cb.sphere)
	}
}
```

Provide `sphereCornerBlendFixture(t, r)` as a named fixture building a `*cornerBlend` with a valid `geom.Sphere` and three `blendArc`s whose endpoints are three mutually-tangent points on the sphere (mirror the geometry `sphereTriLoop` already encodes, but as a `cornerBlend`). Add `abs(float64) float64` if not already in the test package.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestExtractTrihedral -v`
Expected: FAIL — `undefined: extractTrihedral`.

- [ ] **Step 3: Write `extractTrihedral`**

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// extractTrihedral turns a solved planar-trihedral cornerBlend into the 3-sided RailLoop the exact
// sphere tier recognizes (3 concentric equal-radius arcs on the corner sphere). Each Side's Curve is
// the boundary arc between successive arm tangent points, lying on cb.sphere; Adjacent is the sphere
// itself (recognition is rails-only, so Adjacent is not load-bearing here) and Cont is G1. This is the
// PLANAR strangler path: it reuses the existing arc geometry (no setback recomputation) so the
// downstream face stays byte-for-byte. ok=false if an arc cannot be reconstructed → honest-reject.
func extractTrihedral(cb *cornerBlend) (RailLoop, bool) {
	if len(cb.arcs) != 3 {
		return RailLoop{}, false
	}
	sides := make([]Side, 3)
	for i, a := range cb.arcs {
		arc, ok := sphereBoundaryArc(cb.sphere, a)
		if !ok {
			return RailLoop{}, false
		}
		sides[i] = Side{Curve: arc, Adjacent: cb.sphere, Cont: G1}
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// sphereBoundaryArc reconstructs the exact geom.Arc3d for one blendArc on cb.sphere: a great-or-small
// circle arc through ta→mid→tb whose center is the projection of the sphere center onto the arc plane
// and whose radius/normal come from the three points. Reuse geom's 3-point arc constructor.
func sphereBoundaryArc(sph geom.Sphere, a blendArc) (geom.Arc3d, bool) {
	return geom.NewArc3dThrough(a.ta, a.mid, a.tb) // see note: use the actual 3-point arc constructor
}
```

`geom.NewArc3dThrough` is a placeholder for the **actual** 3-point circular-arc constructor in `kernel/geom` — grep for it (`grep -rn "func NewArc3d\|ThreePoint\|Through" kernel/geom/`) and call the real one; if none exists, construct the `geom.Arc3d` directly from its fields (`Center`, `Normal`, `RefDir`, `Radius`, `StartAngle`, `SweepAngle`) computed from `ta`/`mid`/`tb` (circumcircle of three points), and add a small `arc3dThrough(ta, mid, tb) (geom.Arc3d, bool)` helper in `geom` with its own unit test. Return `ok=false` on collinear points.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestExtractTrihedral -v`
Expected: PASS.

- [ ] **Step 5: Wire the sphere strangler (byte-for-byte)**

Modify `spherePatchFace` (fillet_faces.go:441) to route recognition through the extractor while keeping the loop byte-identical:

```go
// spherePatchFace builds the corner sphere patch. The surface is now VALIDATED through the generalized
// engine (extractTrihedral -> resolveBlend recognizes the exact sphere), proving the RailLoop path
// agrees; the boundary LOOP is still chainArcs(cb.arcs) so the assembled face is byte-for-byte with the
// pre-strangler output (the trim/arcs are the byte-for-byte risk -- see the extractor-wave spec §2/§3).
func spherePatchFace(cb *cornerBlend) filletFace {
	surface := sphereSurfaceViaRail(cb) // extractor-recognized sphere, == cb.sphere by construction
	loop := chainArcs(cb.arcs)
	if spherePatchFlipped(cb, loop) {
		loop = reverseArcLoop(loop)
	}
	return filletFace{surface: surface, loops: []filletLoop{loop}}
}

// sphereSurfaceViaRail routes the corner sphere through the RailLoop engine and returns the recognized
// analytic sphere; it falls back to cb.sphere if extraction/recognition declines (do-no-harm), so a
// mis-extraction can never change the byte-for-byte output.
func sphereSurfaceViaRail(cb *cornerBlend) geom.Surface {
	loop, ok := extractTrihedral(cb)
	if !ok {
		return cb.sphere
	}
	patch, ok := resolveBlend(loop, sphereScale(cb))
	if !ok || patch.Kind != BlendKindSphere {
		return cb.sphere
	}
	return patch.Surface
}
```

`sphereScale(cb)` = `ResolutionForPoints` over the arc tangent points (or reuse the existing scale the fillet pipeline threads — grep how `spherePatchFace`'s callers obtain a `Resolution`; if one is in scope, pass it through instead of recomputing). Keep `spherePatchFace` ≤ 20 lines by splitting as shown.

- [ ] **Step 6: The golden-diff gate**

Run: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
Expected: `50`, **byte-identical on the planar-trihedral cases**. Confirm byte-identity explicitly: before this task, capture the per-case output hashes (`go test ./model/feature -run TestOCCTBlendSimple` writes result solids under the corpus fixture dir; diff the emitted `.brep`/area dumps against the pre-task run). Any planar-trihedral case whose output changed ⇒ the extractor's sphere ≠ `cb.sphere` ⇒ investigate before committing (the fallback should have prevented this; a change means recognition returned a *different* valid sphere).

- [ ] **Step 7: Full package + build + commit**

Run: `go test ./kernel/ops/ && go build ./kernel/... && go vet ./kernel/ops/`

```bash
git add kernel/ops/corner_extract_trihedral.go kernel/ops/corner_extract_trihedral_test.go kernel/ops/fillet_faces.go
git commit -m "feat(blend): extractTrihedral + sphere strangler (recognition via RailLoop, byte-for-byte loop)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## MILESTONE 2 — Runout extractor integration (the Tracer Bullet)

Greens the five FAIL(area) runout cases by feeding the already-shipped runout detection into the general 4-sided fill. `solveImprint` already tiers the five families internally (S1 cyl / S4 cone / T1 torus / T7 ellip-cyl / T9 bspline), so `extractRunout` is **one adapter**, gated **five times** by the DRAWEXE area oracle.

### Task 7: `extractRunout` — runout crossing → 4-sided RailLoop

**Files:**
- Create: `kernel/ops/corner_extract_runout.go`
- Test: `kernel/ops/corner_extract_runout_test.go`

**Interfaces:**
- Consumes: `detectRunouts(ef edgeFillet, res Resolution) []runoutImprint`, `runoutImprint{host; hostIsA; plane geom.Plane; footprintEdge; nodes [2]crossing; boundary; side; flat; back}`, `solveImprint(im runoutImprint, res Resolution) (imprintCut, bool)`, `imprintCut{pMinus, pPlus math.Point3; arc geom.Curve3}`, `edgeFillet{a,b; cyl geom.Cylinder; c0,c1 corner; edge}`, `Side`/`RailLoop`, `G0`/`G1`.
- Produces: `extractRunout(ef edgeFillet, im runoutImprint, cut imprintCut, res Resolution) (RailLoop, bool)`.

The 4-sided runout loop: **s0** = the receded fillet-boundary segment between the crossing nodes P−→P+ (on the host plane, the `boundary`/`side` from `runoutImprint`), G1 to the host plane; **s1** = the fillet arm cross-section arc at P+ (from `ef.cyl` at the P+ generator), G1 to `ef.cyl`; **s2** = the footprint arc `cut.arc` from P+→P− (the curved feature's base curve, G0 crease); **s3** = the fillet arm cross-section arc at P− , G1 to `ef.cyl`. This replaces the un-trimmed fillet strip in the crossing span with a coons4 patch bounded by the exact footprint.

- [ ] **Step 1: Write the failing test**

```go
// TestExtractRunoutIsClosedValence4 drives the S1-shaped synthetic fixture: a straight plane∧plane
// fillet cylinder whose runout crosses a coplanar cylindrical boss footprint.
func TestExtractRunoutIsClosedValence4(t *testing.T) {
	ef, im, cut, res := s1RunoutFixture(t)
	loop, ok := extractRunout(ef, im, cut, res)
	if !ok {
		t.Fatal("extractRunout declined the S1 fixture")
	}
	if loop.Valence() != 4 {
		t.Fatalf("valence = %d, want 4", loop.Valence())
	}
	if !loop.Closed(res.Weld()) {
		t.Fatal("runout loop not closed")
	}
}

// TestExtractRunoutFillsAndDoesNotFold proves the loop fills via coons4 and passes the F2 probe.
func TestExtractRunoutFillsAndDoesNotFold(t *testing.T) {
	ef, im, cut, res := s1RunoutFixture(t)
	loop, _ := extractRunout(ef, im, cut, res)
	fill, rails, sides, ok := coons4Fill(loop)
	if !ok {
		t.Fatal("coons4Fill declined the runout loop")
	}
	if !ribbonSeamNonFolding(fill, rails, sides, res) {
		t.Fatal("runout loop folds under coons4")
	}
}
```

`s1RunoutFixture(t)` is a named fixture returning a hand-built `edgeFillet` (a straight cylinder over a plane∧plane edge), the `runoutImprint` a `detectRunouts` call yields on it, its `solveImprint` `imprintCut`, and the model `Resolution`. Build it from a synthetic slab + coplanar cylindrical boss (mirror the "synthetic slab+coplanar-boss" fixture named in the spec's verification section). Prefer constructing `ef`/`im` via the real `computeEdgeFillet` + `detectRunouts` on a small `topo.Body` so the fixture exercises the true detection path, not a fabricated struct.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestExtractRunout -v`
Expected: FAIL — `undefined: extractRunout`.

- [ ] **Step 3: Write the implementation**

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// extractRunout turns one runout crossing into the 4-sided RailLoop the general coons4 tier fills:
// the receded host-plane boundary segment (G1 to the plane), the two fillet-arm cross-section arcs at
// the crossing nodes (G1 to the fillet cylinder), and the curved feature's footprint arc (G0 crease).
// It reuses the shipped detectRunouts/solveImprint as the rail source (they already tier S1/S4/T1/T7/T9).
// ok=false on a bad cross-section / non-crossing → honest-reject (ADR-3), leaving the fillet whole.
func extractRunout(ef edgeFillet, im runoutImprint, cut imprintCut, res Resolution) (RailLoop, bool) {
	boundary, ok := recededBoundaryRail(im, cut, res) // s0: P- -> P+ along the receded fillet boundary, on im.plane
	if !ok {
		return RailLoop{}, false
	}
	armPlus, ok := armSectionArc(ef, cut.pPlus, res) // s1: fillet cross-section arc at P+
	if !ok {
		return RailLoop{}, false
	}
	armMinus, ok := armSectionArc(ef, cut.pMinus, res) // s3: fillet cross-section arc at P-
	if !ok {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: boundary, Adjacent: im.plane, Cont: G1},
		{Curve: armPlus, Adjacent: ef.cyl, Cont: G1},
		{Curve: cut.arc, Adjacent: im.host.Geometry().(geom.Surface), Cont: G0},
		{Curve: armMinus, Adjacent: ef.cyl, Cont: G1},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}
```

Two helpers to implement (each 4-20 lines, with their own unit tests):

- `recededBoundaryRail(im runoutImprint, cut imprintCut, res Resolution) (geom.Curve3, bool)` — the segment of the receded fillet boundary line (`im.boundary`, a `boundaryLine2` in `im.plane`'s 2D frame) between the 2D projections of `cut.pMinus`/`cut.pPlus`, mapped back to 3D via `im.back`. A straight segment ⇒ build a `geom.LineSegment3d` (grep the real line-segment type in `geom`). Guard the degenerate zero-length segment (`< res.Weld()`).
- `armSectionArc(ef edgeFillet, at math.Point3, res Resolution) (geom.Curve3, bool)` — the fillet cylinder's circular cross-section (radius `ef.cyl.Radius`) in the plane through `at` perpendicular to `ef.cyl.AxisDir`, swept the quarter between the two host tangents. Reuse the arm-end section geometry `corner`/`cornerAt` already computes (grep `EndSection`/section-arc construction on the cylinder). Guard a section that does not reach the boundary.

Note the `im.host.Geometry().(geom.Surface)` type assertion for the G0 footprint Adjacent — the rim is G0 (no ribbon), so the Adjacent is used only structurally; if the host geometry does not assert to `geom.Surface`, return `ok=false`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestExtractRunout -v`
Expected: PASS.

- [ ] **Step 5: Build + fmt + commit**

Run: `go build ./kernel/... && go vet ./kernel/ops/ && gofmt -l kernel/ops/corner_extract_runout.go`

```bash
git add kernel/ops/corner_extract_runout.go kernel/ops/corner_extract_runout_test.go
git commit -m "feat(blend): extractRunout (runout crossing -> 4-sided RailLoop)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Wire the runout strangler behind the do-no-harm fallback

**Files:**
- Modify: `kernel/ops/fillet_faces.go` (`filletResultFaces` — the runout branch) or `kernel/ops/fillet.go` (the assembly path)
- Test: `kernel/ops/corner_extract_runout_test.go` (integration case)

**Interfaces:**
- Consumes: `extractRunout` (Task 7), `resolveBlend`, `patchToFilletFace` (Task 4), `detectRunouts`/`solveImprint`, `assembleFilletBody`'s existing `obstacleImprovedSolid` do-no-harm pattern.
- Produces: a runout patch emitted as a `filletFace` and trimmed fillet arms, active only when it yields an area-improved valid solid.

Mirror `assembleFilletBody`'s do-no-harm fallback (fillet.go:186-194): build the runout-patched faces; keep them only if the result is a valid, hole-contained solid whose area improved toward parity, else the baseline. This guarantees a mis-fire cannot regress the green corpus.

- [ ] **Step 1: Write the failing test** — an integration test that runs the full fillet on the S1 synthetic body and asserts the result is a watertight solid whose area is within 1% of the analytic expected area (the boss-free trimmed fillet + the coons4 patch). Provide `s1SyntheticBody(t) (*topo.Body, []filletPick)` and assert `assembleFilletBody(...)` (or the top-level `Fillet` entry) yields `IsSolid()` + area within tolerance. Show the complete assertion code.

- [ ] **Step 2: Run test to verify it fails** — Expected: FAIL (runout branch not wired; area still shows the un-trimmed surplus).

- [ ] **Step 3: Wire the runout branch** — in `filletResultFaces` (fillet_faces.go:17), for each `edgeFillet`, call `detectRunouts(ef, res)`; for each imprint with a valid `solveImprint`, `extractRunout` → `resolveBlend` → `patchToFilletFace`, add the patch face and trim the fillet arm to the crossing generators (exclude the under-footprint strip). Return the `obstacleFired`-style bool so `assembleFilletBody` runs its do-no-harm check on the runout path too (extend `obstacleFired` to `patchesFired`, or add a parallel flag — keep the existing obstacle fallback intact). Show the exact code for the branch, mirroring the existing obstacle branch in the same function.

- [ ] **Step 4: Run test to verify it passes** — Expected: PASS (S1 synthetic body watertight, area within 1%).

- [ ] **Step 5: Corpus gate** — `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` → `50` (no regression; the real corpus S1/S4/T1/T7/T9 green in Tasks 9-11, but nothing must drop here). Confirm byte-identity on all non-runout cases.

- [ ] **Step 6: Commit** — `feat(blend): wire runout strangler behind do-no-harm fallback`.

---

### Task 9: Green S1 + S4 (cylinder + cone footprints) — DRAWEXE oracle gate

**Files:** Test/oracle only (`test-utilities/occt-blend/oracle/`), no new production code if Tasks 7-8 generalize correctly.

**Interfaces:** Consumes the wired runout path; the DRAWEXE oracle (`printf 'source S1.tcl\n' | DRAWEXE -b`).

- [ ] **Step 1: Run the S1 case** — `go test ./model/feature -run 'TestOCCTBlendSimple/S1' -v`. Expected before: FAIL(area) (+~1.15% surplus). Capture the current area.
- [ ] **Step 2: Compare to the DRAWEXE oracle** — run the S1 oracle script, record OCCT's Gauss-integrated area; confirm the wired path's area is now within 1%.
- [ ] **Step 3: If S1 still fails,** debug the extractRunout geometry against the oracle (systematic-debugging: gather the per-face area diff first). Do NOT loosen the tolerance. The likely culprits: the arm-section arc sweep bounds, the footprint arc direction, or the boundary-segment endpoints — each checkable against the S1 oracle's face dump.
- [ ] **Step 4: Repeat for S4** (cone footprint — `solveImprint`'s cone tier). Expected: `TestOCCTBlendSimple/S4` FAIL(area)→PASS.
- [ ] **Step 5: Full corpus zero-regression** — the full `TestOCCTBlendSimple` count rises by exactly the newly-greened cases; every other case byte-identical.
- [ ] **Step 6: Commit** — `test(blend): green S1+S4 runout cases (cyl+cone footprint) vs DRAWEXE oracle`.

---

### Task 10: Green T1 (torus footprint) — oracle gate

**Files:** Test/oracle only (production code only if the torus footprint exposes a gap).

- [ ] **Step 1: Run `TestOCCTBlendSimple/T1`** — Expected before: FAIL(area).
- [ ] **Step 2: Oracle compare** (torus footprint = circle; `solveImprint`'s line∩circle tier). Confirm within 1%.
- [ ] **Step 3: If a gap, fix `armSectionArc`/`recededBoundaryRail` for the torus case** (the footprint is a circle on the host plane; the tier already solves it — most likely the fix is in the arm trim, not a new solver). Show the fix.
- [ ] **Step 4: Zero-regression corpus run.**
- [ ] **Step 5: Commit** — `test(blend): green T1 runout case (torus footprint)`.

---

### Task 11: Green T7 + T9 (ellip-cyl + bspline footprints) — oracle gate

**Files:** Test/oracle only (production code only if the elliptical/bspline footprint exposes a gap).

- [ ] **Step 1: Run `TestOCCTBlendSimple/T7`** (elliptical-cylinder footprint = ellipse; `solveImprint`'s line∩ellipse tier) — Expected before: FAIL(area). Oracle compare within 1%.
- [ ] **Step 2: Run `TestOCCTBlendSimple/T9`** (bspline footprint; `solveImprint`'s marched root-find tier) — Expected before: FAIL(area). Oracle compare within 1%.
- [ ] **Step 3: Fix any gap** exposed by the ellipse/bspline footprint (the `cut.arc` for T9 is a bspline segment, so the G0 side carries a bspline `geom.Curve3` — confirm `coons4Fill`'s `asBSplineCurve` accepts it; if not, that is the fix, in the rail conversion, not a new solver). Show the fix.
- [ ] **Step 4: Full corpus** — all five runout cases (S1/S4/T1/T7/T9) now PASS; every non-runout case byte-identical; count ≥ 55.
- [ ] **Step 5: Commit** — `test(blend): green T7+T9 runout cases (ellip-cyl+bspline footprint)`.

---

## Milestone-completion gate (end of tracer)

- `go test ./kernel/ops/ ./model/feature` — all green.
- `golangci-lint run ./kernel/ops/` — 0 issues.
- `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` ≥ 55, with S1/S4/T1/T7/T9 among the PASSes and every other case byte-identical to the pre-wave output.
- The strangler seam is proven (sphere + obstacle byte-for-byte against their correct baselines; runout greened through the same `resolveBlend`), unblocking the Milestone 3 follow-up plan (curved miter + N-way).
- **No PR** — the whole corpus is not yet green (Milestone 3 remains). Accumulate on the branch.

---

## Plan Self-Review

**Spec coverage:** §2 Tracer/Strangler → Tasks 6 (sphere), 5 (obstacle), 7-8 (runout). §3 golden-diff bifurcation → Task 6 (planar reuse of `chainArcs`), Task 7 (curved via `solveImprint`). §4 shared-normal law → carried by the shipped `adjacentRibbon` (Adjacent surfaces supplied by Tasks 5/7). §5 F2 → Tasks 1 (probe) + 2 (sign fix). §6 strangler facade + do-no-harm → Tasks 6, 8. §7 M1 order A→B→C→D → Tasks 1→2→3→(4,5,6). §7 M2 → Tasks 7-11. F3 → Task 3. Milestone 3 → explicitly deferred (Scope Check).

**Placeholder scan:** two deliberate "grep for the real symbol" notes remain — `geom.NewArc3dThrough` (Task 6, Step 3) and the line-segment/section-arc types (Task 7, Step 3). These are NOT banned placeholders: the plan states the exact fields to compute and mandates a named helper + unit test if the constructor is absent. Flagged for the implementer to bind to the real `geom` API rather than invent a signature.

**Type consistency:** `ribbonSeamNonFolding(fill, sides, scale)` used identically in Tasks 1/2/5/7. `loopRibLen(loop)` replaces `coons4RibLen`/`tri3RibLen` (Task 3) and is consumed in Task 1's fixed-up test. `patchToFilletFace(patch, parent)` defined in Task 4, consumed in 6/8. `extractTrihedral(cb)`/`extractObstacle(of)`/`extractRunout(ef,im,cut,res)` signatures fixed at definition and matched at every call site. `Side{Curve, Adjacent, Cont}` + `RailLoop{Sides, Provenance}` per the shipped `corner_rail.go`.
