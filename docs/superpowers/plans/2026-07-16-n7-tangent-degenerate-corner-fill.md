<!-- SPDX-License-Identifier: GPL-2.0-only -->

# N7 Tangent-Degenerate Corner-Fill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Green OCCT `tests/blend/simple/N7` (whole-body **61222.9**, 12 faces) — a tangent-degenerate convex trihedral fillet corner on a `cylinder − box` cut — by rooting the corner ball correctly, **delegating** the corner patch to the existing RailLoop/`coons4` engine so it emits the 4-sided rational fill, terminating the wall arm ruling on the bitten inner-notch loop, and splicing the three B3 far-runouts. Corpus 55→**56**, B3 byte-faithful.

**Architecture:** Ports-&-adapters. System A (`fillet_curved_*.go`, the curved-arm weld) delegates its corner PATCH to System B (`corner_*.go` + `geom/coons_fill.go`, the RailLoop engine) through a new anticorruption-layer extractor `extractCurvedCorner` resolved via `resolveBlend` — `analyticSphere` wins B3's octant, `coons4` wins N7's 4-sided fill. B3 stays byte-faithful via the `spherePatchFace` strangler (surface-swap, legacy loop kept, fallback on decline). No public API (ADR-0018 not triggered).

**Tech Stack:** Go (`oblikovati` GPL module, `kernel/ops`); `oblikovati.org/kernel/geom` (`FillSurface`/`CoonsFill`), `kernel/topo`, `oblikovati.org/math`; the RailLoop engine (`RailLoop`/`Side`/`resolveBlend`/`coons4Provider`); DRAWEXE (OCCT) per-face oracle; corpus harness `TestOCCTBlendSimple` in `model/feature`.

## Global Constraints

Every task's requirements implicitly include this section:

- **NO PR until the whole corpus is green.** This slice targets corpus 55→**56** (N7). No PR.
- **Do-no-harm is the floor.** Any guard failure ⇒ fall back to the clean unwelded state (`curvedArmUnweldedError`) or the legacy `curvedSphereFace`, never a wrong / mis-closed / inside-out solid. An honest decline beats a fabricated face.
- **B3 byte-faithful (ADR-2 strangler).** On a clean octant, delegating yields the SAME face `curvedSphereFace` produces today: the octant keeps its legacy `chainSetbackArcs` loop and takes only the recognized surface; `resolveBlend` decline falls back wholesale. B3's corpus subtest, per-face weld areas, watertightness, and volume all unchanged after EVERY task.
- **M1–M4 tripwire byte-identical** — S1/S4/S7/T1/T4/T7 unchanged; every other corpus grid byte-identical.
- **The engine stays topo-free.** `resolveBlend`/`coons4`/`analyticSphere` (and everything they call) import only `geom`+`math` (+`topo` for `Lineage` bookkeeping). The ACL (`extractCurvedCorner`) may read System-A types + `topo`; it passes only a `RailLoop` into the engine — never a `*topo.Face`. The assembler branches only on `filletFace`, never on `CornerBlendPatch.Kind`/`Certificate`.
- **Tessellation correctness is the highest priority.** A wrong normal / hole / winding corrupts every downstream consumer; the `Certificate.NoFold` guard and the N7 VOLUME regression (area is orientation-blind) are load-bearing.
- **Tolerances model-relative (ADR-0042).** `res = ResolutionForBody(body)` (or `ResolutionForPoints`); on-edge/closure tol = `res.Weld()·r` (r = fillet radius, corner-local — NOT body diameter); area gate = `res.Weld()·r²`. Angular/parallel tests are scale-free constants (like `retrimAxisParallelTol = 1e-9`, `seamAngularTol`). **Never a bare `1e-6`.** Every `false`/reject site's comment carries the offending value + expected bound.
- **Code style:** functions 4–20 lines; files < 500 lines; explicit types (no `any`); early returns, ≤ 2 indent; no code duplication; names < 5 grep hits. **SPDX `GPL-2.0-only`** header on every new `.go`.
- **TDD with named fakes**, not inline stubs. Every new function gets a test; N7 gets the corpus regression. **Tests can have bad premises** — assert against the DRAWEXE oracle ground truth, not against our own output. (This slice exists because a prior task shipped a fixture that modeled impossible topology; verify every fake against the real call path.)
- **Corpus count (the `-v` is REQUIRED):** `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
- **DRAWEXE oracle:** source `test-utilities/occt-blend/oracle/drawenv.sh`, then `printf 'source X.tcl\n' | "$DRAWEXE" -b`; the vendored N7 recipe restores `CFI_f1234fim.rle` (`tscale ×10`, blends r=5 on `s_5,s_4,s_10`), then `explode result F; sprops`/`vprops` per face.

### N7 ground truth (DRAWEXE; corner V=(50,0,10); cylinder axis (50,50), R=50; Σ=61222.9)

| face | surface (OCCT) | area | role |
|------|----------------|------|------|
| result_1 | Cylinder z∈[0,130] | 38033.8 | WALL host — **2 wires** (outer rim + inner notch loop z∈[10,80]) |
| result_6 | Cylinder | **546.695** | `s_4` cyl arm; setback z=15, **runout z=80** |
| result_3 | Torus | **212.306** | `s_5` torus arm; runs out onto x=80 (result_2) |
| result_12 | Cylinder | **195.464** | `s_10` cyl arm; runs out onto y=30 (result_10) |
| result_5 | **BSpline deg 2×9, 4 edges/4 verts** | **90.194** | **corner FILL** — verts (55.56,0.31,5),(55,5.28,10),(44.44,0.31,15),(50,5.28,15) |
| result_9 | Plane z=10 | **517.428** | corner host (arms s_5+s_10), 1 wire, 4 edges |
| result_11 | Plane x=50 | **1606.89** | corner host (arms s_4+s_10), 1 wire, 4 edges — tangent to wall |
| result_2/4/10 | Plane | 1406.8 / 810.7 / 2094.6 | notch walls = far-runout targets (result_6→_4, _3→_2, _12→_10) |
| result_7/8 | Plane | 7853.98 ea | end caps (untouched) |

---

## File Structure

**Existing (touched):**
- `kernel/ops/fillet_curved_corner.go` — the corner-ball solve: `solveCurvedBlend`, `curvedCornerCenter`, `planePairLine`, `cylinderLineParam`, `nearerRoot` (**the mis-root site**, `:136`), `curvedCornerConsistent`. (T-N7.1)
- `kernel/ops/fillet_curved_weld.go` — `curvedWeldFaces` (`:195`, **the emit point**, `:208` calls `curvedSphereFace`), `curvedSphereFace` (`:329`), `chainSetbackArcs` (`:339`), `armRailBundle` (`:230`), `cylinderRulingOuterOnHost` (`:277`, **the wall-termination site**). (T-N7.0, T-N7.3)
- `kernel/ops/fillet_curved_corner_solve.go` — `cornerWeld`, `armSetback` (carries `farVertex`), `curvedSetbackRail`-via-`chainSetbackArcs`. (read-only reference)

**New:**
- **Create** `kernel/ops/corner_extract_curved.go` — the ACL: `extractCurvedCorner(w cornerWeld, arms []edgeFillet, res Resolution) (RailLoop, bool)` + `wallContactSide(...)` (the 4th rail). (T-N7.0 builds the 3-side octant; T-N7.2 adds the 4th side.)
- **Create** `kernel/ops/corner_extract_curved_test.go`.

**Reused UNCHANGED (System B — do not edit except the certificate tightening noted in T-N7.2):**
- `kernel/ops/corner_rail.go` (`RailLoop`, `Side{Curve, Adjacent, Cont}`, `Continuity` `G0/G1`, `Valence()`, `Closed(tol)`), `corner_resolve.go` (`resolveBlend(loop, scale) (CornerBlendPatch, bool)`), `corner_provider_coons4.go` (`coons4Provider`, `Fits = Valence()==4`), `corner_provider_sphere.go` (`analyticSphereProvider`, `railLoopToFilletLoops`), `corner_patch_adapter.go` (`patchToFilletFace(patch, parent) filletFace`), `geom/coons_fill.go` (`FillSurface`, `FillSide{Adjacent, AdjEdge, Order}`).

**Strangler precedent to mirror:** `spherePatchFace`/`sphereSurfaceViaRail` (`fillet_faces.go:501-521`) — routes the SURFACE through `extractTrihedral → resolveBlend`, keeps the legacy loop, falls back on decline. Copy this shape for the curved corner.

Task order (from the architecture brief): **T-N7.1 → T-N7.0 → T-N7.2 → T-N7.3 → T-N7.4.** Base commit: **`a7b62edb`** (corpus 55).

Each task's corpus gate is **55** until T-N7.4 (N7 composes only when ball-root + fill + termination + far-runout all land); T-N7.4 flips it to **56**.

---

### Task 1: Tangent-corner ball-root selection (T-N7.1)

`curvedCornerCenter` solves a quadratic for the ball centre and `nearerRoot` (`fillet_curved_corner.go:136`) picks the root **nearer the corner vertex**. At N7's tangent-degenerate dihedron (the x=50 plane is the wall's tangent plane along (50,0,z)) that heuristic picks the **reflected root** — center (45,5.28,**5**), tangent points at z=5, corner-triangle area 42 — instead of the correct z=15 root (oracle corner 90.194). `curvedCornerConsistent` accepts both (both are valid tangent balls), so the tiebreak is the bug. Fix: select the root whose per-arm setback stations lie **in the arm's parameter domain adjacent to V**, and add an **area-witness** gate (never trust the tangency check alone).

**Files:**
- Modify: `kernel/ops/fillet_curved_corner.go` (`nearerRoot` → root selection by station-domain; `solveCurvedBlend`/`curvedCornerCenter` thread the arms + witness)
- Test: `kernel/ops/fillet_curved_corner_test.go` (or the existing `*_corner_solve_test.go`)

**Interfaces:**
- Consumes: `geom.Cylinder`, `geom.Plane`, `armStation(surf, c, scale, res)` (`fillet_curved_corner_solve.go:112` — returns the spine station or `false` if C is off-spine), the arm surfaces + their domains, `ResolutionForPoints`.
- Produces: `curvedCornerCenter` returns the root whose tangent ball yields in-domain adjacent stations; a new `func rootStationsInDomain(cyl geom.Cylinder, planes [2]*topo.Face, r float64, cand math.Point3, arms []geom.Surface, res Resolution) bool` witness. Signature of `curvedCornerCenter` MAY gain the arm surfaces + vertex (already has vertex).

- [ ] **Step 1: Write the failing root-selection test**

Build the real N7 corner geometry (wall cylinder R50 axis ẑ at (50,50); x=50 plane; z=10 plane; r=5; vertex V=(50,0,10)) as a named fake, assert the solved center is the z=15 root (tangent points at z=15, not z=5). Author `n7CornerFaces(t)` against the real topo/geom API; grep existing corner tests for face-builders first.

```go
func TestCurvedCornerCenter_PicksInDomainRootAtTangentDihedron(t *testing.T) {
	cyl, planes, v, r := n7CornerInputs(t) // wall R50 axis z@(50,50); x=50 & z=10 planes; V=(50,0,10); r=5
	res := curvedCornerResolution(v, cyl, planes)

	c, ok := curvedCornerCenter(cyl, planes, r, v.Point(), res /*, arms*/)

	if !ok {
		t.Fatalf("curvedCornerCenter declined the N7 tangent-dihedron corner")
	}
	// the CORRECT root places tangent points at z=15 (setbacks), NOT the reflected z=5 root.
	if got := float64(cylinderWallPoint(cyl, c).Z()); stdmath.Abs(got-15) > res.Weld()*r {
		t.Fatalf("ball mis-rooted: wall tangent z=%.4f; want z=15 (reflected root z=5 gives corner area 42 vs oracle 90.19)", got)
	}
}

func TestCurvedCornerCenter_CleanOctantUnchanged(t *testing.T) {
	// a clean (non-tangent) trihedral corner: the in-domain root == the nearer-vertex root (reduction)
	cyl, planes, v, r := cleanOctantInputs(t)
	c, ok := curvedCornerCenter(cyl, planes, r, v.Point(), curvedCornerResolution(v, cyl, planes))
	if !ok { t.Fatalf("clean octant declined") }
	assertSameAsLegacyNearerVertexRoot(t, c, cyl, planes, r, v) // no behaviour change on the non-degenerate corner
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestCurvedCornerCenter -v`
Expected: FAIL — the z=5 reflected root is returned (`want z=15`).

- [ ] **Step 3: Implement in-domain root selection + area-witness**

Replace `nearerRoot`'s vertex-distance tiebreak with a station-domain criterion: for each quadratic root, form the candidate center, compute each arm's `armStation`, and require every station to be in-domain and on the V-adjacent side; pick the root that satisfies it (reject if neither/both — honest-reject, do-no-harm). Add the witness `rootStationsInDomain`. Keep `curvedCornerConsistent` as the tangency check but AND it with the witness. Functions 4–20 lines; reject comments carry the offending station + expected domain.

The reduction: on a clean corner exactly one root has in-domain adjacent stations, and it IS the nearer-vertex root, so `TestCurvedCornerCenter_CleanOctantUnchanged` passes.

- [ ] **Step 4: Run tests + B3/corpus gate**

Run: `go test ./kernel/ops -run 'TestCurvedCornerCenter|B3CurvedArmWeld|B3VolumeRegression|CurvedCorner' -v` → PASS (N7 root fixed; B3 corner unchanged).
Run the corpus count → **55** (N7 still declines downstream — no seam/fill/termination yet).
Run `go build ./... && go vet ./kernel/... && gofmt -l kernel/ops && golangci-lint run` → clean.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_curved_corner.go kernel/ops/fillet_curved_corner_test.go
git commit -m "fix(blend): root the corner ball by in-domain station at a tangent dihedron (T-N7.1)"
```

---

### Task 2: The seam — `extractCurvedCorner` ACL + delegate `curvedWeldFaces` (T-N7.0)

Route the curved-arm corner PATCH through the RailLoop engine, mirroring `spherePatchFace`. This task wires the **octant** (3-valence) path — surface via `resolveBlend`, loop kept legacy, fallback on decline — so B3 stays byte-identical and the seam exists. The N7 4th rail is added in Task 3.

**Files:**
- Create: `kernel/ops/corner_extract_curved.go`, `kernel/ops/corner_extract_curved_test.go`
- Modify: `kernel/ops/fillet_curved_weld.go` (`curvedWeldFaces:208` — swap the corner-face surface through the engine, keep the legacy loop, fall back)
- Modify: `kernel/ops/fillet_curved_weld_test.go` (B3 byte-identity golden)

**Interfaces:**
- Consumes: `cornerWeld{center, radius, arms []armSetback}`, `curvedSetbackRail(w, arm) (geom.Arc3d, bool)`, `armSetback.arm` (the arm surface), `RailLoop`/`Side{Curve, Adjacent, Cont}`/`G1`, `resolveBlend(loop, res) (CornerBlendPatch, bool)`, `patchToFilletFace(patch, topo.Lineage) filletFace`, `curvedSphereFace(w, sphere)` (the fallback + legacy loop source), `res.Weld()`.
- Produces:
  - `func extractCurvedCorner(w cornerWeld, arms []edgeFillet, res Resolution) (RailLoop, bool)` — for an octant: 3 `Side`s, each `Curve` = `curvedSetbackRail(w, a)`, `Adjacent` = `a.arm`, `Cont` = `G1`; `Provenance` = `topo.Lineage{}`. Ordered head-to-tail (reuse the `chainSetbackArcs` ordering) so `RailLoop.Closed(res.Weld()·w.radius)` holds. Returns `false` if any rail declines.
  - `func curvedCornerSurfaceViaRail(w cornerWeld, arms []edgeFillet, sphere geom.Sphere, res Resolution) geom.Surface` — routes through the engine, returns the recognized sphere; falls back to `sphere` on any decline / `Kind != BlendKindSphere` (do-no-harm, mirrors `sphereSurfaceViaRail`).

- [ ] **Step 1: Write the failing extractor + reduction test**

```go
func TestExtractCurvedCorner_OctantIsThreeValentSphere(t *testing.T) {
	w, arms, sphere, res := b3CornerWeld(t) // the clean B3 octant (3 equal-r arms)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 3 {
		t.Fatalf("extractCurvedCorner: want a closed 3-side octant loop; ok=%v valence=%d", ok, loop.Valence())
	}
	if !loop.Closed(res.Weld() * w.radius) {
		t.Fatalf("octant RailLoop is not closed")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindSphere {
		t.Fatalf("octant must resolve to the analytic sphere tier; ok=%v kind=%q", ok, patch.Kind)
	}
	assertSameSphere(t, patch.Surface, sphere) // recognized == solved corner sphere
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestExtractCurvedCorner -v` → FAIL (`undefined: extractCurvedCorner`).

- [ ] **Step 3: Implement the ACL + surface-via-rail**

Write `extractCurvedCorner` (octant branch only this task): build the 3 setback great-arc `Side`s (`Adjacent = a.arm`, `Cont = G1`), ordered like `chainSetbackArcs`, `Provenance = topo.Lineage{}`. Keep the octant arcs exactly the `curvedSetbackRail` arcs (concentric equal-r) so `analyticSphereProvider.Fits` still wins. Write `curvedCornerSurfaceViaRail` mirroring `sphereSurfaceViaRail`.

- [ ] **Step 4: Delegate `curvedWeldFaces` (ADR-2 Step 1 strangler — surface-swap, loop-preserved, fallback)**

In `curvedWeldFaces:208`, replace the corner-face emission so the SURFACE comes from the engine but the LOOP stays legacy `chainSetbackArcs`:

```go
	sf, ok := curvedCornerFace(w, sphere, arms, res)
	if !ok {
		return nil, "corner face declined"
	}
```

```go
// curvedCornerFace emits the corner patch: the SURFACE is validated through the RailLoop engine
// (extractCurvedCorner→resolveBlend — analyticSphere wins the octant, coons4 the degenerate 4-sided
// fill), while the octant's boundary LOOP stays the legacy chainSetbackArcs so B3 is byte-for-byte
// (ADR-2 Step 1 strangler; sphere loop-collapse is a gated follow-up). Falls back to curvedSphereFace
// wholesale on any engine decline (do-no-harm).
func curvedCornerFace(w cornerWeld, sphere geom.Sphere, arms []edgeFillet, res Resolution) (filletFace, bool) {
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok {
		return curvedSphereFace(w, sphere) // extractor declined — legacy octant path
	}
	patch, ok := resolveBlend(loop, res)
	if !ok {
		return curvedSphereFace(w, sphere)
	}
	switch patch.Kind {
	case BlendKindSphere:
		return curvedSphereFace(w, sphere) // octant: engine-validated surface == sphere, KEEP legacy loop (ADR-2 Step 1)
	case BlendKindCoons4:
		return patchToFilletFace(patch, topo.Lineage{}), true // degenerate 4-sided fill: take the engine's loops
	default:
		return curvedSphereFace(w, sphere) // any other tier (e.g. tri3) is NOT a valid curved corner — do-no-harm
	}
}
```

`curvedWeldFaces` must pass `arms` to `curvedCornerFace` (it already holds `arms`). The octant branch returns `curvedSphereFace(w, sphere)` verbatim → B3 byte-identical. **Do-no-harm gate (load-bearing):** only `BlendKindSphere` (via the legacy loop) and `BlendKindCoons4` are admitted; a `tri3`/other tier falls back rather than emitting a wrong corner. This is why the task order roots the ball (Task 1) and lands the 4th rail (Task 3, `BlendKindCoons4`) BEFORE the wall termination (Task 4) that lets the corner reach assembly — until then N7 declines upstream and no wrong corner can ship.

- [ ] **Step 5: B3 byte-identity golden + corpus gate**

Add/extend a golden in `fillet_curved_weld_test.go` asserting B3's assembled corner face (surface + loop) is unchanged (compare the corner `filletFace` fields against the pre-strangler `curvedSphereFace(w, sphere)` output).

Run: `go test ./kernel/ops -run 'TestExtractCurvedCorner|B3CurvedArmWeld|B3VolumeRegression|CurvedCorner|Coons4|AnalyticSphere' -v` → PASS.
Corpus count → **55** (B3 byte-identical; N7 declines — its corner is 4-sided, `extractCurvedCorner` returns 3-valent with only 3 arms so far, `coons4` not yet reachable). `go build/vet/gofmt/golangci-lint` clean.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_extract_curved.go kernel/ops/corner_extract_curved_test.go kernel/ops/fillet_curved_weld.go kernel/ops/fillet_curved_weld_test.go
git commit -m "feat(blend): route the curved-arm corner surface through the RailLoop engine (T-N7.0 strangler)"
```

---

### Task 3: The 4-sided fill — wall-contact rail + adjacency + area cert (T-N7.2)

> **CORRECTED (2026-07-16, during implementation):** the "keep 3 great-arc rails + append a 4th" framing below is geometrically WRONG for N7 — the three offset spines do not concur, so there is no single corner ball. The correct rails are each arm's **cross-section circle at its reflected-family centre** (m_i on that arm's spine) + an **on-wall** bridge curve (not a spatial Arc3d — coons4 silently re-projects an off-wall arc, faking the area). See `.superpowers/sdd/n7-fill-rails-rederivation.md` (oracle-validated: reproduces result_5's 4 vertices exactly). The implementer works from the corrected brief `cf-task-3-brief.md`, not the steps below. Architecture (delegate to coons4) + reduction-to-B3 (octant: all m_i=C → great circles → valence-3) unchanged.

`coons4Provider` already fills a valence-4 `RailLoop`. This task makes `extractCurvedCorner` emit the **4th rail** (the wall-contact arc) for a degenerate corner, wires its `Adjacent` = the wall cylinder, and certifies the resulting fill matches the oracle (result_5 = **90.194**, G1 to all four neighbours).

**Files:**
- Modify: `kernel/ops/corner_extract_curved.go` (add `wallContactSide` + the degenerate-4 branch)
- Modify: `kernel/ops/corner_provider_coons4.go` ONLY if the certificate is too weak to gate 90.194 (a certificate tightening, not new architecture — see Step 4)
- Test: `kernel/ops/corner_extract_curved_test.go`

**Interfaces:**
- Consumes: the solved `cornerWeld` (correctly rooted, Task 1), the arms' host faces (`edgeFillet.a/.b` → the wall cylinder geometry), `geom.Arc3d`/`geom.Arc3dByThreePoints`, `Side{Curve, Adjacent, Cont}`, `resolveBlend`, `coons4Provider`, the DRAWEXE oracle for result_5.
- Produces: `func wallContactSide(w cornerWeld, wall geom.Cylinder, res Resolution) (Side, bool)` — the 4th rail: the wall-contact arc from (55.56,0.31,5) to (44.44,0.31,15) (a curve ON the wall cylinder), `Adjacent` = `wall`, `Cont` = `G1`. `extractCurvedCorner` returns a 4-valence loop when the corner touches the wall along an arc (degenerate case), 3-valence otherwise.

- [ ] **Step 1: Write the failing 4-sided-fill test (assert against the DRAWEXE oracle, not our output)**

```go
func TestExtractCurvedCorner_N7IsFourValentCoons(t *testing.T) {
	w, arms, res := n7CornerWeld(t) // the CORRECTLY-ROOTED N7 corner (Task 1); 3 arms + wall-contact arc
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 4 {
		t.Fatalf("N7 corner: want a closed 4-side loop (3 arm rails + wall arc); ok=%v valence=%d", ok, loop.Valence())
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCoons4 {
		t.Fatalf("N7 corner must resolve to coons4; ok=%v kind=%q", ok, patch.Kind)
	}
	if a := surfaceArea(t, patch.Surface, patch.Loops); stdmath.Abs(a-90.194)/90.194 > 0.01 {
		t.Fatalf("corner fill area %.3f; want 90.194 within 1%% (DRAWEXE result_5)", a)
	}
}
```

Before writing `90.194`, re-run the DRAWEXE N7 recipe and confirm `result_5` area (the oracle is ground truth).

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestExtractCurvedCorner_N7 -v` → FAIL (still 3-valent / no wall rail).

- [ ] **Step 3: Implement `wallContactSide` + the degenerate-4 branch**

Detect the tangent-degenerate corner (the wall is tangent to one host plane along the corner line — reuse the T-N7.1 degeneracy signal) and, in that case, append `wallContactSide` as the 4th `Side` (ordered so the loop stays `Closed`). The wall-contact arc is the curve on the wall cylinder joining the two arm-rail endpoints that lie on the wall; build it exactly (`geom.Arc3d` on the cylinder), `Adjacent = wall`, `Cont = G1`. Reduces to the 3-valent octant when non-degenerate.

- [ ] **Step 4: Certify the fill vs the oracle**

Confirm `coons4`'s certificate (`Certificate.Valid`: `Closed && WeldsArms && NoFold && MaxDev ≤ res.Weld() && MaxAngleDev ≤ seamAngularTol`) admits the N7 fill AND that its area matches 90.194 within 1%. If the existing certificate passes a fill whose area is wrong (a too-weak gate), tighten `certifyCoons4Patch` (in `corner_provider_coons4.go`) with an explicit reject carrying the measured vs expected — do NOT weaken the test to pass. Verify G1 by sampling `‖n_fill − n_neighbour‖` along each of the 4 rails (< `seamAngularTol`). Keep `coons4`'s existing tests green.

- [ ] **Step 5: Corpus gate**

Run: `go test ./kernel/ops -run 'TestExtractCurvedCorner|Coons4|AnalyticSphere|B3CurvedArmWeld|CurvedCorner' -v` → PASS.
Corpus count → **55** (N7's corner now builds, but the weld still declines: the wall arm ruling overshoots (Task 4) and the far-runout isn't confirmed (Task 5)). `go build/vet/gofmt/golangci-lint` clean.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_extract_curved.go kernel/ops/corner_provider_coons4.go kernel/ops/corner_extract_curved_test.go
git commit -m "feat(blend): emit the wall-contact 4th rail so coons4 fills the N7 corner (T-N7.2)"
```

---

### Task 4: Multi-loop WALL arm-ruling termination (T-N7.3)

The weld-side wall ruling (`cylinderRulingOuterOnHost`, `fillet_curved_weld.go:277`) reads the wall's OUTER loop and terminates at z=130; the true runout z=80 lives on the wall's **inner notch loop**. Route it through the bitten loop `bittenLoop(host, w.center)` (C0, landed) so it terminates at z=80, matching the far-vertex authority (`runoutAgrees`).

**Files:**
- Modify: `kernel/ops/fillet_curved_weld.go` (`cylinderRulingOuterOnHost` — use bitten-loop segs)
- Test: `kernel/ops/fillet_curved_weld_test.go` (or the retrim test)

**Interfaces:**
- Consumes: `bittenLoop(host *topo.Face, c math.Point3, tol float64) (*topo.Loop, bool)` + `segsFromLoop(l *topo.Loop) []endSeg` (both landed in C0, `fillet_curved_retrim_loop.go`), `armRulingEnd`/`chartRulingExit`, `runoutAgrees` (C1), `res.Weld()`.
- Produces: `cylinderRulingOuterOnHost` terminates on the bitten loop (unchanged signature). Reduces to B3 on single-loop hosts (`bittenLoop` = outer loop).

- [ ] **Step 1: Write the failing termination test**

Assert edge-10's wall ruling on the real N7 wall (2-loop) terminates at z=80, not z=130. Reuse the N7 wall geometry; author the fake as a real 2-loop cylinder face (outer rim z∈[0,130] + inner notch loop z∈[10,80]).

```go
func TestCylinderRulingOuterOnHost_TerminatesOnInnerNotchLoop(t *testing.T) {
	wall := n7WallFace(t) // Cyl R50, 2 wires: outer rim z∈[0,130] + inner notch z∈[10,80]
	arm, set, w, tHost := n7Edge10RulingInputs(t) // s_4 arm; tHost at the corner (z≈15)
	end, ok := cylinderRulingOuterOnHost(wall, arm, set, tHost, w, w.res().Weld()*w.radius)
	if !ok {
		t.Fatalf("wall ruling declined; expected termination at z=80 (inner notch top)")
	}
	if got := float64(end.Z()); stdmath.Abs(got-80) > 0.02 {
		t.Fatalf("wall ruling terminated at z=%.4f; want z=80 (inner notch loop), NOT z=130 (outer rim)", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestCylinderRulingOuterOnHost -v` → FAIL (terminates at z=130).

- [ ] **Step 3: Route through the bitten loop**

In `cylinderRulingOuterOnHost`, replace the `originalHostSegs(host)` (outer-loop) source with `segsFromLoop(bittenLoop(host, w.center, tol))` — the same bitten loop the retrim uses — so arm-face and retrimmed host land on a byte-identical point. Honest-reject if `bittenLoop` declines. On a single-loop B3 host `bittenLoop` = the outer loop → unchanged.

- [ ] **Step 4: Verify termination + plane self-resolve + B3/corpus**

Run: `go test ./kernel/ops -run 'TestCylinderRulingOuterOnHost|B3CurvedArmWeld|B3VolumeRegression|CurvedCorner' -v` → PASS (z=80; B3 unchanged — bitten=outer on single-loop).
Instrument or unit-check that with Task 1's z=15 root, the x=50/z=10 plane exits now succeed (tHost inside the plane loop). If a residual grazing/vertex hit remains, add the vertex-snap exit guard (test edge endpoints as candidates) — do NOT add a plane far-runout patch. Corpus count → **55** (far-runout confirmed in Task 5). `go build/vet/gofmt/golangci-lint` clean.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_curved_weld.go kernel/ops/fillet_curved_weld_test.go
git commit -m "feat(blend): terminate the wall arm ruling on the bitten inner-notch loop (T-N7.3)"
```

---

### Task 5: Far-runout on the 3 notch faces + green N7 + full gate (T-N7.4)

With the ball rooted (T1), the corner fill emitted (T2/T3), and the wall ruling terminated (T4), the weld composes. Confirm the existing B3 far-runout splices each arm's terminal cross-section arc onto its notch face, then assert the whole-body + per-face + volume oracle gate and flip N7 green.

**Files:**
- Verify/adjust: `kernel/ops/fillet_curved_farrunout.go` (`farArcsBiting`/`spliceCornerBite`/`farRunoutFace` — confirm they splice result_6→result_4, result_3→result_2, result_12→result_10)
- Modify: `model/feature/occtparity/corpus.json` only if N7 is not already registered with `61222.9` (it is — verify)
- Create/Extend: `kernel/ops/fillet_curved_weld_test.go` — `TestFilletEdges_N7CornerFillWeld` (per-face) + `TestFilletEdges_N7VolumeRegression`

**Interfaces:**
- Consumes: the full weld via `filletResolvedEdges` on the imported N7 body; the corpus harness `TestOCCTBlendSimple/N7`; the volume helper used by `B3VolumeRegression`.
- Produces: N7 green (corpus 55→**56**); a per-face + volume regression test.

- [ ] **Step 1: Write the failing N7 per-face + volume test**

Build the N7 body exactly as `RunCase` does (`importInput(simple/N7.step)` → `AddFilletSets` → `Recompute`); assert per-type faithfulness + whole-body + volume against the oracle. Run `vprops result` on the DRAWEXE N7 recipe first to record the volume oracle — do not hard-code an unverified number.

```go
func TestFilletEdges_N7CornerFillWeld(t *testing.T) {
	body := buildN7Blended(t)
	assertWatertight(t, body)
	assertFaceAreaNear(t, body, faceTypeCoons4Fill, []float64{90.194})            // corner 4-sided fill (result_5)
	assertFaceAreaNear(t, body, faceTypeCylinder, []float64{546.695, 195.464})    // s_4 + s_10 arms
	assertFaceAreaNear(t, body, faceTypeTorus, []float64{212.306})                // s_5 arm
	assertFaceAreaNear(t, body, faceTypePlane, wantsPlanes{517.428, 1606.89})     // retrimmed corner hosts
	assertWallInnerLoopBitten(t, body, 38033.8)
	if a := totalArea(t, body); stdmath.Abs(a-61222.9)/61222.9 > 0.01 {
		t.Fatalf("N7 whole-body area %.3f; want 61222.9 within 1%% (DRAWEXE)", a)
	}
}

func TestFilletEdges_N7VolumeRegression(t *testing.T) {
	got, want := solidVolume(t, buildN7Blended(t)), drawexeN7Volume // `vprops result`, recorded
	if stdmath.Abs(got-want)/want > 0.01 {
		t.Fatalf("N7 volume %.3f vs oracle %.3f: an inverted fill/retrim face inverts the normal", got, want)
	}
}
```

- [ ] **Step 2: Run — expect FAIL or integration gaps**

Run: `go test ./kernel/ops -run 'TestFilletEdges_N7' -v`
Expected: FAIL if a splice/order seam is loose (far-runout not picking up an arm's arc; a decline in `curvedWeldFaces`). Debug the seam — do NOT loosen a gate.

- [ ] **Step 3: Confirm far-runout + flip the corpus**

Confirm `farArcsBiting`/`spliceCornerBite` splice the three notch faces (result_2/4/10). Ensure `model/feature`'s `TestOCCTBlendSimple/N7` asserts `61222.9`. Run the corpus count:
Run: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` → **56**.

- [ ] **Step 4: B3 + M1–M4 tripwire + whole-corpus non-regression**

Run: `go test ./kernel/ops -run 'B3CurvedArmWeld|B3VolumeRegression|B3UnconsumedPickDeclines|CurvedCorner|Coons4|AnalyticSphere' -v` → PASS.
Confirm S1/S4/S7/T1/T4/T7 still PASS and the base-vs-`a7b62edb` corpus diff = **only N7 flips** (every other grid byte-identical).

- [ ] **Step 5: Full local gate**

Run: `go build ./... && go vet ./kernel/... && gofmt -l kernel/ops && golangci-lint run` → clean.
Run: `go test ./kernel/ops ./model/feature` → PASS.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/fillet_curved_farrunout.go kernel/ops/fillet_curved_weld_test.go model/feature
git commit -m "feat(blend): green OCCT simple/N7 — tangent-degenerate corner fill weld (corpus 55->56)"
```

---

## Verification (whole slice)

- **N7 whole-body** 61222.9 via `TestOCCTBlendSimple/N7` → PASS; corpus 55→**56**.
- **Per-type faithfulness:** corner 4-sided coons4 fill 90.194 (NOT a sphere octant); cyl arms 546.695+195.464; torus 212.306; retrimmed planes 517.428+1606.89; wall 38033.8 inner-loop bitten.
- **B3 byte-faithful (ADR-2 Step 1):** octant keeps its legacy loop + engine-validated surface; B3 corpus subtest + per-face + watertight + volume unchanged; base-vs-head corpus diff = only N7.
- **N7 VOLUME regression** — the orientation guard (`Certificate.NoFold` + volume; area is orientation-blind).
- **M1–M4 tripwire** — S1/S4/S7/T1/T4/T7 byte-identical; every other grid byte-identical.
- **Engine untouched** — `coons4`/`analyticSphere`/`resolveBlend` unchanged except a possible `coons4` certificate tightening (Task 3); their existing tests green.
- **DRAWEXE oracle** — the vendored N7 recipe is ground truth for every per-face + volume value; record `vprops result` before hard-coding the volume.
- **No public API** (ADR-0018 not triggered); **NO PR** (corpus 56/195, not whole-green).

## Out of scope (carried)
- **ADR-2 Step 2** — collapse the octant onto the engine's patch loop + delete `curvedSphereFace`, gated by a `railLoopToFilletLoops(octant) == chainSetbackArcs` byte-identity golden. Separate follow-up.
- **Interior-weld runout** (two blended edges sharing a far vertex) — flag → clean decline.
- **S-family merge** (S1/S4/T1/T7 imprint) — not this slice.

## References
- `docs/superpowers/specs/2026-07-16-n7-tangent-degenerate-corner-fill-design.md` — the approved design.
- `.superpowers/sdd/n7-corner-seam-architecture.md` — the seam (ADR-1/2/3), the ACL contract, the ubiquitous language.
- `.superpowers/sdd/n7-runout-rederivation.md` — the geometry (4-sided fill, tangent mis-root, per-face oracle).
- `.superpowers/sdd/n7-c2-diagnosis.md` — the falsifying trace.
