<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Corner Extractor Wave — Tracer Bullet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove the generalized corner-blend seam end-to-end by routing junctions through one `ExtractRailLoop → resolveBlend` facade — correcting the latent obstacle ribbon-sign fold, holding the green planar-trihedral + obstacle cases byte-for-byte, then greening the runout cases via the oracle-derived 3-quad hexagon tiler (S1/S4/T1/T7; T9 deferred to Milestone 3's n-sided fill).

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

## MILESTONE 2 — Runout extractor (oracle-derived 3-quad hexagon tiling)

**Supersedes the original single-4-quad M2 (empirically falsified — see below).** The S1 runout was proven by the DRAWEXE oracle (report: `scratchpad/tracer/s1-runout-topology.md`) to be a **double interference**: two features cross one fillet (horizontal r6 boss → footprint circle on host plane A; vertical r8 boss → footprint circle on host plane B). The interfered region is intrinsically a **hexagon** (6 sides: plane-A-left, fillet-left, featureB-arc, fillet-right, plane-A-right, featureA-arc). Our engine has only `coons4`/`tri3`, so — exactly as OCCT does — we tile the hexagon into **3 valence-4 `coons4` patches** (central + left + right) joined by **2 shared internal seams** (G0/watertight for this tracer; fill-to-fill G1 across the seams is a coupled multi-patch solve deferred to Milestone 3 — the area oracle does not measure internal-seam tangent). The original guess (one quad, one boss) put two coplanar sides sharing both endpoints (a flat lune, `Closed=false`); the real hole spans both host planes, so every tiled quad has a side that lifts off plane A. **T9 is deferred to Milestone 3** (its fill is 2 patches including a valence-6 one that needs the n-sided provider).

Measured constants (S1, radius 6, perpendicular hosts, `d = r·tan((π−γ)/2) = r = 6`): fillet axis `{y=-4, z=4}` along X; fillet-cut abscissa `x = ±√(R_B²−d²) = ±√48 = 6.93`; the three RailLoops' corners are tabulated per task below (all verified closed, 4 distinct corners).

### Task 7: `solveImprint` accepts `geom.Arc3d` (circular footprint)

**Files:**
- Modify: `kernel/ops/fillet_runout_imprint.go` (the `geom.Circle`-only type gate at `solveImprint`'s head)
- Test: `kernel/ops/fillet_runout_imprint_test.go`

**Interfaces:**
- Consumes: `solveImprint(im runoutImprint, res Resolution) (imprintCut, bool)`, `runoutImprint.footprintEdge *topo.Edge`, `edge.Geometry() geom.Surface`-or-curve, `geom.Circle`, `geom.Arc3d{Center math.Point3; Normal, RefDir math.UnitVector3; Radius float64}`.
- Produces: `footprintConic(edge *topo.Edge) (center math.Point3, radius float64, normal math.UnitVector3, ok bool)` (accepts `geom.Circle` OR `geom.Arc3d`).

The blocker (verified): imported STEP feature footprints arrive as `geom.Arc3d`, never `geom.Circle`, so `solveImprint`'s `circle, ok := im.footprintEdge.Geometry().(geom.Circle)` rejects every real fixture. Extend it to reconstruct the underlying circle from the `Arc3d` basis (an `Arc3d` already carries `Center`/`Radius`/`Normal` — no sample-fit; do NOT sample-and-fit, that loses exactness).

- [ ] **Step 1: Write the failing test** — drive the REAL S1 substrate (`runoutFixtureCrossingBoss(t)` from the shipped runout tests imports `simple/S1` + runs `computeEdgeFillet` + `detectRunouts`), assert both imprints (feature-A on plane A, feature-B on plane B) now yield `solveImprint ok==true` with non-degenerate `pMinus`/`pPlus` (they returned `ok==false` before). Assert the feature-B fillet-cut abscissa `|pPlus.X| ≈ 6.93` within `res.Weld()`. Show the complete test.

- [ ] **Step 2: Run RED** — `go test ./kernel/ops/ -run TestSolveImprintArc3d -v` → FAIL (`ok==false` on the Arc3d footprint).

- [ ] **Step 3: Implement `footprintConic`** and re-point `solveImprint` to use it. Add a `footprintConic` that returns the conic params for both `geom.Circle` and `geom.Arc3d`; `solveImprint` calls it and honest-rejects only when neither matches (message carries the actual `Geometry()` type). Keep the existing `geom.Ellipse`/bspline honest-reject (Task 12 / T9 add those). Show the code.

- [ ] **Step 4: Run GREEN** — both imprints yield sane `imprintCut`.

- [ ] **Step 5: Gates** — `go build ./kernel/... && go vet ./kernel/ops/ && golangci-lint run ./kernel/ops/`; corpus `TestOCCTBlendSimple` grep-count = 50 (not wired yet).

- [ ] **Step 6: Commit** — `feat(blend): solveImprint accepts geom.Arc3d circular footprints (unblocks S1/S4/T1)`.

### Task 8: Runout-region detection (multi-crossing spine clustering)

**Files:**
- Create: `kernel/ops/corner_runout_region.go`
- Test: `kernel/ops/corner_runout_region_test.go`

**Interfaces:**
- Consumes: `detectRunouts(ef, res) []runoutImprint`, `solveImprint(im, res) (imprintCut, bool)`, `edgeFillet{cyl geom.Cylinder; c0,c1 corner; edge}`, `imprintCut{pMinus, pPlus math.Point3; arc geom.Curve3}`.
- Produces: `type runoutRegion struct { imprints []runoutImprint; cuts []imprintCut; loEdge, hiEdge float64 }`, `detectRunoutRegions(ef edgeFillet, res Resolution) []runoutRegion`.

S1 has **two** imprints interfering in the **same** fillet span → **one** coupled hexagonal hole, NOT two independent runouts (advisor pitfall 5). Project each crossing's `[pMinus,pPlus]` onto the fillet spine (the cylinder axis parameter), sort, and **merge overlapping intervals** into one region. A region with ≥2 imprints is the double-interference hexagon; a region with 1 imprint is a simpler runout (still handled by the same tiler with the missing feature side degenerating — but S1/S4/T1/T7 are all double, so scope this task to the ≥2 case and honest-skip singletons for now).

- [ ] **Step 1: Failing test** — on the S1 fixture, `detectRunoutRegions(ef, res)` returns exactly **1** region whose `imprints` has length 2 (both bosses), and whose spine interval covers `x ∈ [-6.93, 6.93]`. Show the test (build `ef` via the real pipeline as Task 7 does).

- [ ] **Step 2: RED** — `undefined: detectRunoutRegions`.

- [ ] **Step 3: Implement** — call `detectRunouts`, `solveImprint` each (skip `ok==false`), project `[pMinus,pPlus]` onto the spine via `ef.cyl.AxisDir`, sort by lo, merge overlapping, bundle into `runoutRegion`s. Show the clustering code (each func 4-20 lines).

- [ ] **Step 4: GREEN.**

- [ ] **Step 5: Gates** — build/vet/lint; corpus = 50.

- [ ] **Step 6: Commit** — `feat(blend): detectRunoutRegions clusters coupled multi-crossing runouts`.

### Task 9: Hexagon → 3-quad tiler (`extractRunout`)

**Files:**
- Create: `kernel/ops/corner_extract_runout.go`
- Test: `kernel/ops/corner_extract_runout_test.go`

**Interfaces:**
- Consumes: `runoutRegion` (Task 8), `edgeFillet`, `imprintCut`, `geom.Arc3dByThreePoints(a,mid,b)`, `Side{Curve; Adjacent; Cont}`, `RailLoop`, `G0`/`G1`, `ribbonSeamNonFolding(fill, rails, sides, scale)` (Task 1), the host-plane `geom.Plane` + fillet `geom.Cylinder` + feature-wall `geom.Cylinder` surfaces.
- Produces: `extractRunout(region runoutRegion, ef edgeFillet, res Resolution) ([]RailLoop, bool)` — the **3** RailLoops (central/left/right).

Emit the three RailLoops exactly as the oracle measured (report §"The three RailLoops"). Continuity is load-bearing. The shipped `coons4Provider` (`corner_provider_coons4.go`) builds a G1 ribbon ONLY for a side with `Cont==G1` AND a non-nil `Adjacent`; a G0 side (`Cont<=0`) or a nil `Adjacent` gets `FillSide{Order:0}` (position-only, no ribbon). So:
- **G1** where a real analytic host is tangent and available at extraction time: the fillet ¼-circle → the fillet cylinder (`ef.cyl`); the host-plane runout curve → its host plane. These Adjacents exist now.
- **G0, `Adjacent: nil`** on the un-blended feature-arc sides — they carry **NO** tangent (feeding a feature-wall tangent as a G1 ribbon INVERTS the patch, advisor pitfall 4; setting `Cont: G0` makes `ribbonSide` skip the ribbon regardless, but pass `nil` to state intent).
- **G0, `Adjacent: nil`** on the two internal seams **for this tracer**. A fill-to-fill G1 seam needs the neighbour patch's surface, which does not exist at extraction time — it is a **coupled multi-patch solve deferred to Milestone 3**. The area oracle (`checkprops -s`) measures the union area + watertightness, NOT internal-seam tangent, so G0 seams (shared rail → position-continuous → watertight) pass the gate. A G0 seam leaves a tangent crease between patches; that visual-smoothness refinement is M3's, not a correctness gap for the area gate.

**CENTRAL** (bridges the two feature walls) — corners `(-3.38,-10,4.96)→(3.38,-10,4.96)→(3.38,-7.25,10)→(-3.38,-7.25,10)`:
`[featA-arc r6 (G0, adj=nil) | seam_right (G0, adj=nil) | featB-arc r8 (G0, adj=nil) | seam_left (G0, adj=nil)]` — a pure-position Coons patch (all four sides G0); its admissibility is `NoFold` on the plain Coons fill, not a ribbon.

**RIGHT** — corners `(3.38,-10,4.96)→(6.93,-10,4)→(6.93,-4,10)→(3.38,-7.25,10)`:
`[plane-A runout curve (G1, adj=host plane A) | fillet ¼-circle (G1, adj=ef.cyl fillet cylinder) | featB-arc portion (G0, adj=nil) | seam_right (G0, adj=nil)]`

**LEFT** — mirror of RIGHT (x → −x).

Seam placement is a **free parameter** (the fill is area-validated). Derive the seam endpoints from the arc parameterization (e.g. project the feature-A∩feature-B mutual point, or the arc midpoints between the fillet-cut points and the loop's symmetry plane) — **do NOT hard-code x = 3.38**. The seam runs `(±x_s, -7.25, 10) → (±x_s, -10, 4.96)` connecting the featureB arc (plane B) to the plane-A/featureA junction. **The two flanking patches must share the SAME seam curve object** (same start/end points, same orientation on both loops) so the union is watertight — build each seam once and hand it to both loops.

Helpers (each 4-20 lines, own unit test): `armSectionArc(ef, atCut math.Point3, res) (geom.Curve3, bool)` (fillet cross-section ¼-circle via `geom.Arc3dByThreePoints(ta, mid, tb)` translated to the cut station — the same construction `fillet_faces.go` uses at the corner arcs); `planeARunoutCurve(...)` (the host-plane-A curve between the featureA arc and the fillet-cut, on the plane); `internalSeam(...)` (the free-placement seam curve, a `geom.LineSegment` or arc; `Adjacent: nil`, `Cont: G0` for the tracer). Confirm `geom.Arc3dByThreePoints(start, onArc, end) (Arc3d, error)` (kernel/geom/arc3d.go:39) exists — it does; do NOT re-add it.

- [ ] **Step 1: Failing test** — on the S1 region (Task 8), `extractRunout` returns 3 loops; assert each `loop.Closed(res.Weld())` and `Valence()==4` with 4 pairwise-distinct corners at the measured coordinates above; assert the union's outer boundary is the 6-side hexagon (drop the 2 seams). Then `TestExtractRunoutQuadsFillNonFolding`: `coons4Fill` each loop and `ribbonSeamNonFolding(fill, rails, sides, res)` (pass rails from `coons4Fill`) → all non-folding. Show the tests with the measured corners.

- [ ] **Step 2: RED** — `undefined: extractRunout`.

- [ ] **Step 3: Implement** the hexagon boundary + the 3-quad split + the per-side G0/G1 roles. Include the flat-lune guard (advisor pitfall 1): reject any loop where two **consecutive** sides share the same carrier plane and the enclosed area `< res.Weld()²` — that means a mis-tile. Show the code.

- [ ] **Step 4: GREEN.**

- [ ] **Step 5: Gates** — build/vet/gofmt/lint; corpus = 50 (still unwired). No `topo` import except `topo.Lineage{}` in the produced RailLoops.

- [ ] **Step 6: Commit** — `feat(blend): extractRunout tiles the runout hexagon into 3 coons4 RailLoops`.

### Task 10: Wire runout into the fillet assembly (do-no-harm) + oracle-gate S1

**Files:**
- Modify: `kernel/ops/fillet_faces.go` (`filletResultFaces` runout branch) and/or `kernel/ops/fillet.go` (`assembleFilletBody`)
- Test: `kernel/ops/corner_extract_runout_test.go` (integration) + the corpus gate

**Interfaces:**
- Consumes: `detectRunoutRegions` (Task 8), `extractRunout` (Task 9), `resolveBlend(loop, scale)`, `patchToFilletFace(patch, parent)` (Task 4), the do-no-harm pattern in `assembleFilletBody` (fillet.go:186-194, `obstacleImprovedSolid`).

For each `edgeFillet`: `detectRunoutRegions`; for each region, `extractRunout` → 3 loops → `resolveBlend` each → `patchToFilletFace`; **split the fillet cylinder** so the plain quarter-cyl survives only outside the region's spine interval (`|x| > 6.93` for S1), replacing the interfered span with the 3 patches. Extend the `obstacleFired` do-no-harm flag to cover the runout path (or a parallel `runoutFired`) so `assembleFilletBody` keeps the runout result only if it yields an area-improved valid solid; else baseline. Mirror the existing obstacle branch exactly.

- [ ] **Step 1: Failing integration test** — drive the full fillet on the S1 body; assert the result `IsSolid()` and total area within 1% of the oracle `3662.79`. Show the assertion.

- [ ] **Step 2: RED** — area still shows the un-trimmed surplus (~+1.15%).

- [ ] **Step 3: Wire the runout branch** (mirror the obstacle branch in `filletResultFaces`) + the fillet split + the do-no-harm flag. Show the exact code.

- [ ] **Step 4: GREEN + S1 oracle gate** — `go test ./model/feature -run 'TestOCCTBlendSimple/S1' -v` → PASS; area within OCCT's 1% (`printf 'source S1.tcl\n' | ../occt-build/lin64/gcc/bin/DRAWEXE -b` → `checkprops -s 3662.79`).

- [ ] **Step 5: Zero-regression corpus** — `TestOCCTBlendSimple` count = 51 (50 + S1), every other case's PASS/FAIL byte-identical (diff the name set).

- [ ] **Step 6: Commit** — `feat(blend): wire runout 3-quad fill behind do-no-harm fallback; green S1`.

### Task 11: Green S4 + T1 (cone→circle, torus→circle)

**Files:** Test/oracle only unless a footprint exposes a gap; possibly `kernel/ops/fillet_runout_imprint.go` if the cone/torus footprint edge isn't already a circular `Arc3d` on the host plane.

S4 (cone boss, footprint = cone∩⊥plane = **circle**) and T1 (torus boss, footprint = torus∩plane = **circle**) are the SAME 3×val-4 family with circular footprints — Task 7's `Arc3d` solve + Task 9's tiler should green them with no new geometry. **Classify the footprint from the actual surface∩plane, not the feature's surface type** (advisor pitfall 7 — a cone crossed by an oblique plane would be an ellipse; verify the S4/T1 host planes are ⟂ the feature axis so the footprint is genuinely circular).

- [ ] **Step 1: Run `TestOCCTBlendSimple/S4`** — FAIL(area) before. Oracle-compare; confirm within 1%.
- [ ] **Step 2: If a gap,** debug against the S4 oracle (systematic-debugging: per-face area diff first). Do NOT loosen tolerance.
- [ ] **Step 3: Repeat for T1** (torus footprint).
- [ ] **Step 4: Zero-regression corpus** (count rises to 53).
- [ ] **Step 5: Commit** — `test(blend): green S4+T1 runout cases (cone/torus circular footprints)`.

### Task 12: Green T7 (elliptical-cylinder → ellipse footprint)

**Files:** Modify `kernel/ops/fillet_runout_imprint.go` (`footprintConic` + the line∩ellipse solve); test/oracle.

T7's feature-A is an elliptical cylinder (STEP `SURFACE_OF_LINEAR_EXTRUSION` over an ELLIPSE), footprint = **ellipse**. Extend Task 7's `footprintConic`/`solveImprint` to accept `geom.Ellipse`; the crossing is line∩ellipse (same quadratic after the affine normalization `x→x/a, y→y/b`). The 3-quad tiler (Task 9) is unchanged — only the featureA-arc side becomes an elliptical arc (still G0, position-only).

- [ ] **Step 1: Failing test** — `solveImprint` on a `geom.Ellipse` footprint yields a sane `imprintCut` (was honest-rejected).
- [ ] **Step 2: RED.**
- [ ] **Step 3: Implement** the ellipse branch in `footprintConic` + the line∩ellipse root solve (model-scaled discriminant guard). Show the code.
- [ ] **Step 4: GREEN + `TestOCCTBlendSimple/T7` oracle gate** — FAIL→PASS within 1%.
- [ ] **Step 5: Zero-regression corpus** (count rises to 54; S1/S4/T1/T7 all PASS).
- [ ] **Step 6: Commit** — `test(blend): green T7 runout case (elliptical-cylinder footprint)`.


---

## Milestone-completion gate (end of tracer)

- `go test ./kernel/ops/ ./model/feature` — all green.
- `golangci-lint run ./kernel/ops/` — 0 issues.
- `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` ≥ 54, with **S1/S4/T1/T7** among the PASSes and every other case byte-identical to the pre-wave output. **T9 is deferred to Milestone 3** (its fill needs the n-sided provider — see M2 header) and is NOT expected to pass here.
- The strangler seam is proven (sphere + obstacle byte-for-byte against their correct baselines; the runout hexagon greened through the same `resolveBlend` via the 3-quad tiler), unblocking the Milestone 3 follow-up plan (curved miter + N-way + n-sided fill for T9).
- **No PR** — the whole corpus is not yet green (Milestone 3 remains). Accumulate on the branch.

---

## Plan Self-Review

**Spec coverage:** §2 Tracer/Strangler → Tasks 6 (sphere), 5 (obstacle), 7-12 (runout). §3 golden-diff bifurcation → Task 6 (planar reuse of `chainArcs`), Tasks 7/9 (curved via `solveImprint` + oracle-derived 3-quad tiler). §4 shared-normal law → carried by the shipped `adjacentRibbon` (Adjacent surfaces supplied by Tasks 5/9). §5 F2 → Tasks 1 (probe) + 2 (sign fix). §6 strangler facade + do-no-harm → Tasks 6, 10. §7 M1 order A→B→C→D → Tasks 1→2→3→(4,5,6). §7 M2 (oracle-derived 3-quad hexagon tiling) → Tasks 7-12; T9 deferred to Milestone 3 (n-sided fill). F3 → Task 3. Milestone 3 → explicitly deferred (Scope Check).

**Placeholder scan:** the M2 tasks name helpers to bind against the real `geom` API rather than invent signatures — `footprintConic` (Task 7), `armSectionArc`/`planeARunoutCurve`/`internalSeam` (Task 9). These are NOT banned placeholders: each task states the exact fields to compute (the measured RailLoop corners, the `d=r`/`x=±√48` constants) and mandates a named helper + unit test. The runout topology is grounded in the DRAWEXE oracle (`scratchpad/tracer/s1-runout-topology.md`), replacing the empirically-falsified single-quad guess.

**Type consistency:** `ribbonSeamNonFolding(fill, rails, sides, scale)` used identically in Tasks 1/2/5/9. `loopRibLen(loop)` replaces `coons4RibLen`/`tri3RibLen` (Task 3) and is consumed in Task 1's fixed-up test. `patchToFilletFace(patch, parent)` defined in Task 4, consumed in 6/10. `extractTrihedral(cb)`/`extractObstacle(of)` signatures fixed at definition. The M2 chain is `solveImprint(im,res)→imprintCut` (Task 7) → `detectRunoutRegions(ef,res)→[]runoutRegion` (Task 8) → `extractRunout(region,ef,res)→[]RailLoop` (Task 9) → `resolveBlend`/`patchToFilletFace` (Task 10). `Side{Curve, Adjacent, Cont}` + `RailLoop{Sides, Provenance}` per the shipped `corner_rail.go`.
