<!-- SPDX-License-Identifier: GPL-2.0-only -->

> **SUPERSEDED (2026-07-16)** — its premise (C1 gives z=80; C3/C4 in retrimCurvedHost) was falsified; retrimCurvedHost is never reached on real N7. Tasks N1 (C0 bitten-loop, 81f4b190) + N2 (C1 chart termination, 859f650f) LANDED and sound; N3 (C2) REVERTED as inert (a7b62edb). Continuation: `2026-07-16-n7-tangent-degenerate-corner-fill.md` (to be written). Kept for provenance.

# N7 Curved-Arm Host-Retrim Generalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generalize the curved-arm fillet host retrim (`retrimCurvedHost`) from B3's clean 90° cylinder wedge to a boolean-cut / trimmed / multi-loop host face, greening OCCT `tests/blend/simple/N7` (whole-body area **61222.9**) while keeping B3 and the M1–M4 corpus faithful.

**Architecture:** Purely an extension of the existing retrim behind the unchanged `assembleCurvedArmBody → curvedHostFaces → retrimCurvedHost` seam. Five changes (C0–C4) each **provably reduce to the current B3 code** on a clean trim (derivation §R.0–R.3). The corner solve, weld rails, `filletResolvedEdges` routing, and the do-no-harm floor (`curvedArmUnweldedError`) are untouched. No public API surface — this is GPL-module-internal `kernel/ops`.

**Tech Stack:** Go (`oblikovati` GPL module, `kernel/ops`), `oblikovati.org/kernel/geom` + `oblikovati.org/kernel/topo` + `oblikovati.org/math`; DRAWEXE (OCCT) as the per-face oracle; the corpus harness `TestOCCTBlendSimple` in `model/feature`.

## Global Constraints

Every task's requirements implicitly include this section (copied verbatim from the spec + CLAUDE.md standing rules):

- **NO PR until the whole corpus is green.** This slice targets corpus 55→**56** (N7). It does not open a PR.
- **Do-no-harm is the floor.** Any failure of any guard ⇒ return the clean unwelded state (`curvedArmUnweldedError`), never a wrong / mis-closed / inside-out solid. An honest decline always beats a fabricated face.
- **B3 stays faithful** — the B3 corpus subtest, its per-face weld areas, its watertightness and its volume regression all still pass after every task. C0–C4 reduce to current behaviour on the clean wedge; the reduction is the gate, not incidental.
- **M1–M4 tripwire byte-identical** — S1/S4/S7/T1/T4/T7 unchanged (they never enter the curved-arm retrim path). Every other corpus grid byte-identical.
- **Tessellation correctness is the highest priority.** A wrong face normal / hole / winding corrupts every downstream consumer; the orientation guard (C4) and the N7 VOLUME regression (area is orientation-blind) are load-bearing, not optional.
- **Tolerances are model-relative (ADR-0042).** `res = ResolutionForBody(body)`; on-edge / snap / closure tolerance = `res.Weld()·r` where **r = the fillet radius** (corner-local, NOT body diameter); area-balance tolerance = `res.Weld()·r·diam(L*)`. Parallel/angular tests are scale-free constants (like `retrimAxisParallelTol = 1e-9`). **Never a bare `1e-6`** — the error message on any tolerance rejection carries the offending value and the expected bound.
- **Code style:** functions 4–20 lines; files < 500 lines; explicit types (no `any`); early returns, ≤ 2 levels of indentation; no code duplication; names return < 5 grep hits. Error/`false`-return sites carry the offending value + expected shape in the comment.
- **SPDX `GPL-2.0-only` header** on every new `.go` file (`scripts/add-spdx-headers.py`).
- **TDD with named fakes**, not inline stubs. Every new function gets a test; the N7 fix gets the corpus regression test. Tests can have bad premises — assert against the **DRAWEXE oracle** ground truth, not against our own output.
- **Corpus count command (the `-v` is REQUIRED):**
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
- **DRAWEXE oracle:** source `test-utilities/occt-blend/oracle/drawenv.sh` (sets `CASROOT`, `DRAWEXE`), then `printf 'source X.tcl\n' | "$DRAWEXE" -b`; the self-contained N7 recipe restores the vendored fixture `CFI_f1234fim.rle`, `tscale ×10`, blends r=5 on `s_5,s_4,s_10`, then `explode result F; sprops` per face.

### N7 ground truth (DRAWEXE-verified; corner V=(50,0,10); Σ=61222.9)

| face | surface | area | role |
|------|---------|------|------|
| result_1 | Cylinder z∈[0,130] | **38033.8** | WALL host — **2 wires** (outer rim + **inner notch loop = the bitten loop**) |
| result_6 | Cylinder | **546.695** | s_4 vertical cyl arm, z∈[15,80] (setback 15, **runout z=80**, not 130) |
| result_3 | Torus | **212.306** | s_5 torus arm, z∈[1.46,10] |
| result_12 | Cylinder | **195.464** | s_10 planar-cyl arm, z∈[10,15] |
| result_5 | (BSpline in OCCT) | **90.194** | corner sphere patch (spherical-tri excess E=3.608) |
| result_9 | Plane z=10 | **517.428** | corner host (arms s_5+s_10), 1 wire |
| result_11 | Plane x=50 | **1606.89** | corner host (arms s_4+s_10), 1 wire |
| result_2/4/10 | Plane | 1406.8 / 810.7 / 2094.6 | notch walls (untouched) |
| result_7/8 | Plane | 7853.98 ea | end caps πR² (untouched) |

---

## File Structure

The retrim lives in two files today (both < 500 lines, keep them so):

- `kernel/ops/fillet_curved_retrim.go` — corner-host re-clip: `retrimCurvedHost`, `retrimHostSegs`, `retrimCornerHost`, `armContactRail`, `rulingOuterEnd`, `axialExtremeEnd` (to be replaced), `awayFrom`, the torus-rail helpers.
- `kernel/ops/fillet_curved_retrim_loop.go` — loop machinery: `originalHostSegs`, `outerHostLoop`, `innerHostLoops`, `bittenVertex`, `farPathSegs`, `insertSplits`/`splitSeg`/`segParam`, `planeChart` + `planeRayLoopExit`/`rayEdgeHit2d`/`raySegment2d`/`rayArc2d`.

New files this slice adds (to keep each file < 500 and one-responsibility):

- **Create** `kernel/ops/fillet_curved_host_chart.go` — the host intrinsic charts: the `hostChart` interface, `planeChart` (moved) and the new `cylChart` (θ,z), plus the shared `chartRulingExit` ray-cast. Owner of "1-D crossing in the host's own coordinates".
- **Create** `kernel/ops/fillet_curved_host_closure.go` — C4 `hostRetrimValid`: the chart signed-area balance + orientation + bitten-vertex partition gate.
- Tests: extend `kernel/ops/fillet_curved_retrim_test.go`; add `kernel/ops/fillet_curved_host_chart_test.go` and `kernel/ops/fillet_curved_host_closure_test.go`. The corpus flip is asserted by the existing `model/feature` `TestOCCTBlendSimple/N7` subtest.

Task decomposition (each ends with an independently testable deliverable):

- **N1 (C0):** bitten-loop selection — retrim the loop the corner lands on (outer OR inner), carry the rest.
- **N2 (C1):** chart-based arm-ruling termination replacing `axialExtremeEnd`, with the filleted-edge far-vertex authority cross-check.
- **N3 (C2):** `planeRayLoopExit` robustness — endpoint candidates + vertex-snap (the concrete N7 plane decline).
- **N4 (C3):** area-primary far-path on a boolean-cut loop.
- **N5 (C4):** fail-loud host chart signed-area closure invariant.
- **N6:** integrate → green N7; per-face + volume + B3/M4 non-regression; family sweep + interior-weld decline flag.

Base commit for the whole slice: **`215c1fb6`** (corpus 55, spec committed).

---

### Task N1: Bitten-loop selection (C0 / R.0)

Replace `retrimCurvedHost`'s "retrim the OUTER loop, carry inner loops verbatim" with "retrim the **bitten loop** `L*` (the loop containing the vertex nearest the corner-sphere centre C), carry **every other** loop verbatim." On B3 every corner host is single-loop and `L*` is that (outer) loop → byte-identical. On N7 the wall's `L*` is the inner notch window.

**Files:**
- Modify: `kernel/ops/fillet_curved_retrim.go:130-143` (`retrimCurvedHost`), `:152-160` (`retrimHostSegs` — take the selected segs)
- Modify: `kernel/ops/fillet_curved_retrim_loop.go` (add `bittenLoop`, `segsFromLoop`, `loopsExcept`; `bittenVertex` stays for the far-path)
- Test: `kernel/ops/fillet_curved_retrim_test.go`

**Interfaces:**
- Consumes: `topo.Face.Loops() []*topo.Loop`, `topo.Loop.IsOuter() bool`, `topo.Loop.EdgeUses()`, `endSegFromUse`, `unchangedLoop`, `cornerWeld{center math.Point3}`, `Resolution.Weld()`.
- Produces:
  - `func bittenLoop(host *topo.Face, c math.Point3, tol float64) (*topo.Loop, bool)` — the loop whose nearest vertex to `c` is globally minimal; `false` if a second loop ties within `tol` (ambiguous — pathological symmetric part) or the face has no loops.
  - `func segsFromLoop(l *topo.Loop) []endSeg` — the loop's edge uses as `endSeg`s (generalizes `originalHostSegs`, which becomes `segsFromLoop(outerHostLoop(host))`).
  - `func loopsExcept(host *topo.Face, keep *topo.Loop) []filletLoop` — every loop except `keep`, via `unchangedLoop` (generalizes `innerHostLoops`).

- [ ] **Step 1: Write the failing test**

Add to `kernel/ops/fillet_curved_retrim_test.go`. Use a named fake host with two loops — an outer square rim far from C and an inner triangle whose nearest vertex is closest to C — plus a degenerate tie case. `newTwoLoopFace` / `newTieLoopFace` are named fake builders (put them in the test file; reuse existing `endSeg`/loop test helpers if present).

```go
func TestBittenLoop_SelectsInnerNotchLoop(t *testing.T) {
	c := math.NewPoint3(50, 0, 10) // corner-sphere centre near the inner loop
	host := newTwoLoopFace(t,
		/*outer rim vertices, min dist to c ≈ 60*/ squareRim(0, 100),
		/*inner notch vertices, nearest vertex = (50,0,10)*/ notchWindow(50, 0, 10))
	tol := 0.02 // res.Weld()*r for r=5

	l, ok := bittenLoop(host, c, tol)

	if !ok {
		t.Fatalf("bittenLoop: expected the inner notch loop, got ok=false")
	}
	if l.IsOuter() {
		t.Fatalf("bittenLoop selected the OUTER rim; want the inner notch loop (nearest to C=%v)", c)
	}
}

func TestBittenLoop_SingleLoopReducesToOuter(t *testing.T) {
	c := math.NewPoint3(10, -38.7298, 90) // B3 corner centre
	host := newSingleLoopFace(t, squareRim(0, 100))

	l, ok := bittenLoop(host, c, 0.02)

	if !ok || !l.IsOuter() {
		t.Fatalf("bittenLoop on a single-loop host must return that (outer) loop; ok=%v outer=%v", ok, l.IsOuter())
	}
}

func TestBittenLoop_TieRejects(t *testing.T) {
	c := math.NewPoint3(0, 0, 0)
	host := newTieLoopFace(t, /*two loops equidistant to c within tol*/)

	if _, ok := bittenLoop(host, c, 0.02); ok {
		t.Fatalf("bittenLoop must reject an ambiguous two-loop tie (do-no-harm), got ok=true")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestBittenLoop -v`
Expected: FAIL — `undefined: bittenLoop` (and the fake builders you added compile but the function is missing).

- [ ] **Step 3: Implement `bittenLoop`, `segsFromLoop`, `loopsExcept`**

Add to `kernel/ops/fillet_curved_retrim_loop.go`:

```go
// bittenLoop is the loop whose vertex nearest the corner-sphere centre c is globally minimal — the
// wire the trihedral corner actually bites, which may be the OUTER rim (B3) or an INNER notch window
// (N7's boolean-cut wall, where the corner lands on the hole loop, not the boundary). Generalizes the
// T5.3 "retrim the outer loop" assumption (derivation R.0). Rejects (false) when two loops tie for
// nearest within tol (an ambiguous symmetric part — do-no-harm) or the face carries no loops.
func bittenLoop(host *topo.Face, c math.Point3, tol float64) (*topo.Loop, bool) {
	var best *topo.Loop
	bestD, tie := stdmath.Inf(1), false
	for _, l := range host.Loops() {
		d := loopMinDistToCentre(l, c)
		switch {
		case d < bestD-tol:
			best, bestD, tie = l, d, false
		case stdmath.Abs(d-bestD) <= tol && l != best:
			tie = true
		}
	}
	if best == nil || tie {
		return nil, false // no loop, or an ambiguous nearest-loop tie: cannot pick the bitten wire
	}
	return best, true
}

// loopMinDistToCentre is the distance from c to the loop's nearest vertex.
func loopMinDistToCentre(l *topo.Loop, c math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, u := range l.EdgeUses() {
		if d := float64(useFromVertex(u).Point().DistanceTo(c)); d < best {
			best = d
		}
	}
	return best
}

// segsFromLoop turns one loop's edge uses into endSegs (generalizes originalHostSegs, which retrimmed
// only the outer loop, to any loop bittenLoop selects).
func segsFromLoop(l *topo.Loop) []endSeg {
	uses := l.EdgeUses()
	segs := make([]endSeg, 0, len(uses))
	for _, u := range uses {
		segs = append(segs, endSegFromUse(u))
	}
	return segs
}

// loopsExcept carries every loop of host except keep through unchanged (generalizes innerHostLoops:
// once the bitten loop can be inner, the carried-through set is "all others", not "all inner").
func loopsExcept(host *topo.Face, keep *topo.Loop) []filletLoop {
	var out []filletLoop
	for _, l := range host.Loops() {
		if l != keep {
			out = append(out, unchangedLoop(l))
		}
	}
	return out
}
```

- [ ] **Step 4: Wire it into `retrimCurvedHost` / `retrimHostSegs`**

In `kernel/ops/fillet_curved_retrim.go`, rewrite `retrimCurvedHost` (keep it 4–20 lines) to select `L*` first, retrim its segs, and carry the rest:

```go
func retrimCurvedHost(host *topo.Face, w cornerWeld, res Resolution) (filletFace, bool) {
	tol := res.Weld() * w.radius
	star, ok := bittenLoop(host, w.center, tol)
	if !ok {
		return filletFace{}, false // no unambiguous bitten loop — do-no-harm
	}
	segs := segsFromLoop(star)
	if len(segs) < 3 {
		return filletFace{}, false // a host loop needs ≥3 edges to bite a corner from
	}
	loop, ok := retrimHostSegs(host, segs, w, res)
	if !ok {
		return filletFace{}, false
	}
	// the bitten loop, retrimmed, first; every other loop (incl. the outer rim on the wall) verbatim.
	loops := append([]filletLoop{loopFromSegs(loop)}, loopsExcept(host, star)...)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}
```

`retrimHostSegs` already takes `segs` — no signature change; it just now receives `L*`'s segs. Delete the now-unreachable direct `originalHostSegs(host)` call inside it (the caller supplies segs).

- [ ] **Step 5: Run the new tests + the reduction gate**

Run: `go test ./kernel/ops -run 'TestBittenLoop|RetrimCurvedHost|B3CurvedArmWeld|B3VolumeRegression' -v`
Expected: PASS (new tests green; B3 weld + volume unchanged — B3 hosts are single-loop so `L*`=outer).

- [ ] **Step 6: Corpus + tripwire gate**

Run: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
Expected: **55** (N7 still declines — termination not yet generalized; B3 and all others unchanged).
Run: `go build ./... && go vet ./kernel/... && gofmt -l kernel/ops && golangci-lint run`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add kernel/ops/fillet_curved_retrim.go kernel/ops/fillet_curved_retrim_loop.go kernel/ops/fillet_curved_retrim_test.go
git commit -m "feat(blend): retrim the bitten loop (outer or inner), not just the outer (C0)"
```

---

### Task N2: Chart-based arm-ruling termination (C1 / R.1)

Replace `axialExtremeEnd` (which slides a cylinder arm's ruling to the host loop's **global** axial extreme, z=130 on N7) with a first-forward-crossing of the ruling against the **bitten loop**, computed in the host's intrinsic chart (so it is a 1-D problem), and cross-checked against the filleted edge's far vertex (the authority). N7 s_4 → z=**80**. On a clean B3 wall the first crossing IS the global extreme, so this reduces to `axialExtremeEnd`.

**Files:**
- Create: `kernel/ops/fillet_curved_host_chart.go` (the `hostChart` interface, move `planeChart` here from `_retrim_loop.go`, add `cylChart`, add the shared `chartRulingExit`)
- Modify: `kernel/ops/fillet_curved_retrim_loop.go` (move `planeChart`/`planeRayLoopExit`/`rayEdgeHit2d`/`raySegment2d`/`rayArc2d` to the new file; make them take `hostChart`)
- Modify: `kernel/ops/fillet_curved_retrim.go` (`rulingOuterEnd` → `armRulingEnd`; delete `axialExtremeEnd`; `armContactRail` passes the far vertex)
- Modify: `kernel/ops/fillet_curved_corner_solve.go` (`armSetback` gains `farVertex`; `solveArmSetback` stamps it)
- Test: `kernel/ops/fillet_curved_host_chart_test.go`, extend `fillet_curved_retrim_test.go`

**Interfaces:**
- Consumes: `geom.Cylinder{Origin, AxisDir, Radius, Ref}`, `geom.Plane`, `endSeg{from,to math.Point3; curve geom.Arc3d; arc bool}`, `edgeFillet{edge *topo.Edge}`, `topo.Edge.StartVertex()/EndVertex()`.
- Produces:
  - `type hostChart interface { to2(p math.Point3) math.Point2; to3(q math.Point2) math.Point3 }`
  - `func newCylChart(cyl geom.Cylinder) cylChart` implementing `hostChart` with `to2` = (θ, z), θ=`atan2` about the axis relative to `Ref`, z=axial.
  - `func chartRulingExit(ch hostChart, segs []endSeg, o2 math.Point2, d2 math.Vector2, tol float64) (math.Point3, bool)` — nearest forward (t>0) crossing of the ruling ray (chart origin o2, chart direction d2) with any edge of `segs`, honouring each edge's chart span; the generalized `planeRayLoopExit` body.
  - `func hostChartFor(surf geom.Surface) (hostChart, bool)` — the single plane/cylinder chart dispatch, reused by `rulingChartRay` (N2), `farPathSegs` (N4), and `retrimCornerHost`/`hostRetrimValid` (N5).
  - `armSetback.farVertex math.Point3` — the filleted edge's terminus away from C, projected authority for the runout.
  - `func armRulingEnd(host *topo.Face, cylArm geom.Cylinder, arm armSetback, tHost, v math.Point3, segs []endSeg, tol float64) (math.Point3, bool)` — replaces `rulingOuterEnd`; builds the host chart, casts the ruling, asserts the exit is within `tol` of `project_H(arm.farVertex)`.

- [ ] **Step 1: Write the failing termination test**

`kernel/ops/fillet_curved_host_chart_test.go`. Build a synthetic notched cylinder-wall loop in the (θ,z) chart: a full-height wall rim at z=130 (the global extreme) AND a notch-top rim arc at z=80 whose θ-span contains the ruling θ₀. The ruling starts at tHost (z=15) and runs up. Assert it stops at **z=80**, not 130 — the exact N7 s_4 defect. Use the vendored N7 wall geometry values (cyl R=50 axis ẑ; ruling at θ₀ = the s_4 azimuth; notch-top arc z=80).

```go
func TestArmRulingEnd_StopsAtNotchTopNotGlobalExtreme(t *testing.T) {
	cyl := geom.MustCylinder(math.NewPoint3(0, 0, 0), zAxis(), xRef(), 50) // wall R=50, axis ẑ
	tHost := math.NewPoint3(50, 0, 15)                                     // s_4 setback foot, θ₀=0
	v := math.NewPoint3(50, 0, 10)                                         // bitten corner vertex (below)
	segs := notchedWallLoop(t, /*outer rim z=130, notch-top rim z=80 spanning θ=0*/)
	arm := armSetback{arm: cyl, farVertex: math.NewPoint3(50, 0, 80)}
	tol := 0.02 // res.Weld()*r, r=5

	end, ok := armRulingEnd(hostFaceFor(t, cyl, segs), cyl, arm, tHost, v, segs, tol)

	if !ok {
		t.Fatalf("armRulingEnd: expected the z=80 notch-top runout, got ok=false")
	}
	if got := float64(end.Z()); stdmath.Abs(got-80) > tol {
		t.Fatalf("armRulingEnd terminated at z=%.4f; want z=80 (notch top), NOT z=130 (global extreme)", got)
	}
}

func TestArmRulingEnd_CleanWallReducesToGlobalExtreme(t *testing.T) {
	cyl := geom.MustCylinder(math.NewPoint3(0, 0, 0), zAxis(), xRef(), 50)
	tHost := math.NewPoint3(50, 0, 15)
	v := math.NewPoint3(50, 0, 10)
	segs := cleanWallLoop(t, /*single top rim at z=100, no notch*/ 100)
	arm := armSetback{arm: cyl, farVertex: math.NewPoint3(50, 0, 100)}

	end, ok := armRulingEnd(hostFaceFor(t, cyl, segs), cyl, arm, tHost, v, segs, 0.02)

	if !ok || stdmath.Abs(float64(end.Z())-100) > 0.02 {
		t.Fatalf("clean wall: armRulingEnd must equal the global rim z=100 (axialExtremeEnd reduction); got z=%.4f ok=%v", float64(end.Z()), ok)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestArmRulingEnd -v`
Expected: FAIL — `undefined: armRulingEnd` / `cylChart`.

- [ ] **Step 3: Plumb the far vertex into `armSetback`**

In `kernel/ops/fillet_curved_corner_solve.go`, add the field and stamp it from the filleted edge's far terminus (the vertex further from C):

```go
type armSetback struct {
	arm       geom.Surface
	station   float64
	railDir0  math.UnitVector3
	railDir1  math.UnitVector3
	farVertex math.Point3 // the filleted edge's terminus away from C — the runout authority (R.1a)
}
```

In `solveArmSetback` (has `ef edgeFillet`), set `farVertex` to whichever edge endpoint is farther from `c`:

```go
	return armSetback{
		arm: ef.armSurface, station: station, railDir0: d0, railDir1: d1,
		farVertex: fartherEndpoint(ef.edge, c),
	}, true
```

```go
// fartherEndpoint is the filleted edge's vertex farther from the corner centre c — the arm's far
// runout terminus (near vertex is the corner). Used as the ruling-termination authority (R.1a).
func fartherEndpoint(e *topo.Edge, c math.Point3) math.Point3 {
	s, t := e.StartVertex().Point(), e.EndVertex().Point()
	if s.DistanceTo(c) >= t.DistanceTo(c) {
		return s
	}
	return t
}
```

- [ ] **Step 4: Create the host-chart file with `cylChart` + `chartRulingExit`**

Create `kernel/ops/fillet_curved_host_chart.go` (SPDX header). Move `planeChart` + its `to2`/`to3` here from `_retrim_loop.go`, define the interface, add `cylChart`, and move the ray-cast (`planeRayLoopExit` body → `chartRulingExit`, plus `rayEdgeHit2d`/`raySegment2d`/`rayArc2d` retargeted to `hostChart`). `planeRayLoopExit` stays as a thin wrapper so its planar call sites and their byte-identity are preserved.

```go
// hostChart is a host surface's intrinsic 2-D developed chart: the plane's isometric (u,v) or the
// cylinder's (θ,z). A rolling arm's contact ruling is a 1-D ray in this chart, so the crossing with
// the trimmed boundary is a 2-D ray/segment test rather than a 3-D SSI (derivation R.1; Patrikalakis
// & Maekawa §point-in-face).
type hostChart interface {
	to2(p math.Point3) math.Point2
	to3(q math.Point2) math.Point3
}

// cylChart develops a cylinder wall to (θ, z): θ = atan2 about AxisDir relative to Ref (seam at ±π),
// z = signed axial coordinate. A cylinder arm's wall ruling is θ = θ₀ constant (a vertical chart ray);
// a torus arm's wall contact is z = C_z constant (horizontal). This one chart handles both and
// replaces axialExtremeEnd's global-extreme scan.
type cylChart struct {
	origin math.Point3
	axis   math.Vector3 // unit
	ref    math.Vector3 // unit, θ=0 direction (⊥ axis)
	bi     math.Vector3 // unit, axis × ref — θ=+π/2 direction
	radius float64
}

func newCylChart(cyl geom.Cylinder) cylChart {
	axis := cyl.AxisDir.AsVector().Normalize()
	ref := cyl.Ref.AsVector().Normalize()
	return cylChart{origin: cyl.Origin, axis: axis, ref: ref, bi: axis.Cross(ref), radius: cyl.Radius}
}

func (c cylChart) to2(p math.Point3) math.Point2 {
	w := c.origin.VectorTo(p)
	theta := stdmath.Atan2(float64(w.Dot(c.bi)), float64(w.Dot(c.ref)))
	return math.NewPoint2(theta, float64(w.Dot(c.axis)))
}

func (c cylChart) to3(q math.Point2) math.Point3 {
	theta, z := float64(q.X()), float64(q.Y())
	radial := c.ref.Scale(math.Scalar(stdmath.Cos(theta) * c.radius)).
		Add(c.bi.Scale(math.Scalar(stdmath.Sin(theta) * c.radius)))
	return c.origin.TranslateBy(radial).TranslateBy(c.axis.Scale(math.Scalar(z)))
}
```

`chartRulingExit(ch, segs, o2, d2, tol)` is the current `planeRayLoopExit` inner loop, made chart-agnostic: for each seg, `rayEdgeHit2d(ch, seg, o2, d2, tol)` (which uses `ch.to2` on the seg endpoints / arc), keep the nearest **forward** (`t > tol`) hit, return `ch.to3` is not needed — the hit already carries the 3-D point via the seg. **θ-seam safety:** in `raySegment2d`/`rayArc2d`, when the chart is periodic (cyl), compare θ modulo 2π (reuse `wrapToSweep` semantics) so a ruling near the ±π seam does not miss a straddling edge. Keep `rayArc2d`'s existing circle-sweep filter (`arcParam`/`wrapToSweep`).

- [ ] **Step 5: Replace `rulingOuterEnd`/`axialExtremeEnd` with `armRulingEnd`**

In `kernel/ops/fillet_curved_retrim.go`, delete `axialExtremeEnd`, rewrite the terminator to use the chart, and add the far-vertex authority:

```go
// armRulingEnd is the far end of a cylinder arm's straight ruling on a host: the first forward crossing
// of the ruling with the BITTEN loop, computed in the host's intrinsic chart (θ,z on a wall; u,v on a
// plane), then cross-checked against the filleted edge's far vertex (the authority, R.1a). Replaces
// axialExtremeEnd, which slid to the loop's GLOBAL axial extreme and overshot any intermediate trim
// edge (N7 s_4: global rim z=130, true runout z=80). Honest-rejects (false) when the crossing and the
// edge far vertex disagree beyond tol (the edge ends at an interior weld — out of scope — or a wrong
// edge was hit).
func armRulingEnd(host *topo.Face, cylArm geom.Cylinder, arm armSetback, tHost, v math.Point3, segs []endSeg, tol float64) (math.Point3, bool) {
	ch, o2, d2, ok := rulingChartRay(host, cylArm, tHost, v)
	if !ok {
		return math.Point3{}, false
	}
	end, ok := chartRulingExit(ch, segs, o2, d2, tol)
	if !ok {
		return math.Point3{}, false
	}
	if float64(end.DistanceTo(projectToHost(host.Geometry(), arm.farVertex))) > tol {
		return math.Point3{}, false // crossing ≠ edge far vertex: interior weld or wrong edge — decline
	}
	return end, true
}
```

Add a **single** chart constructor both this task and N4/N5 use (no duplicated dispatch): `func hostChartFor(surf geom.Surface) (hostChart, bool)` returning `newCylChart(cyl)` for a `geom.Cylinder`, `newPlaneChart(...)` for a `geom.Plane`, `false` otherwise. `rulingChartRay` calls `hostChartFor` then computes the ruling's chart origin (`ch.to2(tHost)`) and chart direction: for a `geom.Cylinder` host, direction = (0, ±1) with the axial sign = `awayFrom` (θ fixed → vertical chart ray); for a `geom.Plane` host, direction = the projected arm axis `awayFrom(axis, tHost, v)` in-chart (matching today's `planeRayLoopExit`). `projectToHost` is the closest-point of the far vertex on the host surface (radial foot on a cylinder, orthogonal foot on a plane) — a 4–20-line switch mirroring `onHostSurface`. Update `armContactRail` (`:218`) to call `armRulingEnd(host, s, arm, tHost, v, segs, tol)` (it already has `arm armSetback`).

- [ ] **Step 6: Run the termination tests + reduction gate**

Run: `go test ./kernel/ops -run 'TestArmRulingEnd|RetrimCurvedHost|B3CurvedArmWeld|B3VolumeRegression|CurvedCorner' -v`
Expected: PASS (z=80 case green; clean-wall reduction green; B3 weld + volume unchanged; corner solve unchanged).

- [ ] **Step 7: Corpus + tripwire + lint**

Run the corpus count → still **55** (N7 needs C2/C3/C4 to fully close; may still decline at the plane exit or closure). Run `go build ./... && go vet ./kernel/... && gofmt -l kernel/ops && golangci-lint run` → clean.

- [ ] **Step 8: Commit**

```bash
git add kernel/ops/fillet_curved_host_chart.go kernel/ops/fillet_curved_retrim.go kernel/ops/fillet_curved_retrim_loop.go kernel/ops/fillet_curved_corner_solve.go kernel/ops/fillet_curved_host_chart_test.go kernel/ops/fillet_curved_retrim_test.go
git commit -m "feat(blend): chart-based arm-ruling termination with far-vertex authority (C1)"
```

---

### Task N3: Plane-exit robustness — endpoint candidates + vertex-snap (C2 / R.1 pitfalls)

N7's s_4 ruling on the x=50 plane (`result_11`) exits **exactly at a loop vertex** (the z=80 junction). Today `raySegment2d`'s strict `u∈(0,1)` / `t>tol` and `segParam`'s endpoint exclusion reject it → "no valid exit" → the plane host declines. Fix: also test edge **endpoints** as candidate exits, and if a landing is within `res.Weld()·r` of an existing vertex, **snap** to it (no split — a split there makes a zero-length sliver). Grazing/collinear rulings (|d×e| below an angular floor) fall through to C1's far-vertex authority, never a silent drop.

**Files:**
- Modify: `kernel/ops/fillet_curved_host_chart.go` (`chartRulingExit` / `rayEdgeHit2d` — endpoint candidates + snap; the collinear angular floor)
- Test: `kernel/ops/fillet_curved_host_chart_test.go`

**Interfaces:**
- Consumes: `chartRulingExit`, `hostChart`, `endSeg`.
- Produces: `chartRulingExit` now also returns a hit landing exactly on a shared vertex; a new `func snapToVertex(p math.Point3, segs []endSeg, tol float64) (math.Point3, bool)` returns the coincident loop vertex within `tol` (so the far-path splice reuses it, no sliver).

- [ ] **Step 1: Write the failing vertex-exit test**

```go
func TestChartRulingExit_LandsOnLoopVertex(t *testing.T) {
	ch := newPlaneChart(math.NewPoint3(50, 0, 0), xAxisNormal(), yInPlane()) // x=50 plane
	segs := lShapedLoop(t, /*a vertex exactly at the ruling's exit (z=80 junction)*/ math.NewPoint3(50, 0, 80))
	o2 := ch.to2(math.NewPoint3(50, 0, 10)) // tHost
	d2 := ch.to2(math.NewPoint3(50, 0, 80)).Sub(o2).Normalize()
	tol := 0.02

	end, ok := chartRulingExit(ch, segs, o2, d2, tol)

	if !ok {
		t.Fatalf("chartRulingExit rejected an exit that lands exactly on a loop vertex (the N7 x=50 decline)")
	}
	if got := end.DistanceTo(math.NewPoint3(50, 0, 80)); float64(got) > tol {
		t.Fatalf("chartRulingExit landed at %v; want the loop vertex (50,0,80) (snap, no split)", end)
	}
}

func TestSnapToVertex_ReusesExistingCorner(t *testing.T) {
	segs := lShapedLoop(t, math.NewPoint3(50, 0, 80))
	p := math.NewPoint3(50, 0, 80.005) // within tol of the vertex

	v, ok := snapToVertex(p, segs, 0.02)
	if !ok || float64(v.DistanceTo(math.NewPoint3(50, 0, 80))) > 1e-9 {
		t.Fatalf("snapToVertex must return the exact loop vertex; got %v ok=%v", v, ok)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run 'TestChartRulingExit_LandsOnLoopVertex|TestSnapToVertex' -v`
Expected: FAIL — the ruling that ends on a vertex is rejected (endpoint excluded); `undefined: snapToVertex`.

- [ ] **Step 3: Implement endpoint candidates + snap + collinear floor**

In `chartRulingExit`, after the existing interior-hit scan, also evaluate each seg's **endpoints** as candidate exits (parameter along the ruling ≥ tol, forward), and keep the nearest across interior+endpoint hits. Then `snapToVertex(hit, segs, tol)` — if the winning landing is within `tol` of an existing loop vertex, return that vertex point exactly. Add the collinear guard in `raySegment2d`: when `|d2 × e2| / (|d2||e2|) < sinFloor` (`sinFloor = 1e-6`, an angle → scale-free), skip that segment as a crossing (it is parallel; the far-vertex authority in C1 handles a collinear runout), rather than dividing by a near-zero denominator.

```go
// snapToVertex returns the loop vertex within tol of p (so a landing that coincides with an existing
// corner reuses it instead of splitting the edge into a zero-length sliver, derivation R.1 pitfalls).
func snapToVertex(p math.Point3, segs []endSeg, tol float64) (math.Point3, bool) {
	for _, s := range segs {
		if float64(s.from.DistanceTo(p)) <= tol {
			return s.from, true
		}
	}
	return math.Point3{}, false
}
```

`insertSplits`/`splitSeg` already no-op when a point is not strictly interior, so a snapped vertex correctly produces no split.

- [ ] **Step 4: Run the tests + reduction gate**

Run: `go test ./kernel/ops -run 'TestChartRulingExit|TestSnapToVertex|RetrimCurvedHost|B3CurvedArmWeld|B3VolumeRegression' -v`
Expected: PASS (vertex-exit + snap green; B3 planar exits unchanged — B3 rulings exit mid-edge, so the endpoint/snap branch is inert on B3).

- [ ] **Step 5: Corpus + lint**

Corpus count → **55** (N7 may now reach the far-path / closure but is not yet gated green). `go build/vet/gofmt/golangci-lint` → clean.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/fillet_curved_host_chart.go kernel/ops/fillet_curved_host_chart_test.go
git commit -m "feat(blend): plane-exit robustness — endpoint candidates + vertex-snap (C2)"
```

---

### Task N4: Area-primary far-path on a boolean-cut loop (C3 / R.2)

On the bitten loop `L*` (now potentially many edges), the two rail landings `L_A`,`L_B` cut it into two sub-paths. B3's `farPathSegs` keeps the sub-path **not containing the bitten vertex v** — a proxy that mis-picks on a boolean-cut loop where the notch boundary makes "containing v" ambiguous. Replace with the **larger-enclosed-chart-area** criterion (a fillet runout bites a *small* corner, so the far path is the larger region), mirroring `cornerBiteArea` (`_farrunout.go:129`), with the exclude-v test kept as a **cross-check** (disagreement ⇒ reject).

**Files:**
- Modify: `kernel/ops/fillet_curved_retrim_loop.go` (`farPathSegs` → area-primary; add `chartSignedArea`, `subPathArea`)
- Test: `kernel/ops/fillet_curved_retrim_test.go`

**Interfaces:**
- Consumes: `insertSplits`, `indexOfSegFrom`, `segsForward`, `pathHasVertex`, `hostChart`, `endSeg`.
- Produces:
  - `func chartSignedArea(ch hostChart, path []endSeg) float64` — shoelace of the chart-projected polyline (the developed area / winding sign; a cylinder (θ,z) shoelace is a true developed area).
  - `farPathSegs` gains a `ch hostChart` parameter: keeps the larger-area sub-path (closed by the two rails), oriented `fromP→toP`, and rejects if the area pick and the exclude-v pick disagree.

- [ ] **Step 1: Write the failing far-path test**

Build a synthetic boolean-cut loop (a large rim with a small notch) whose two landings split it into a large far arc and a small bite that contains v. Assert `farPathSegs` returns the **large** sub-path and that it excludes v.

```go
func TestFarPathSegs_KeepsLargerAreaSubPath(t *testing.T) {
	ch := newPlaneChart(originP(), zNormal(), xInPlane())
	segs := notchedRimLoop(t, /*big rim, small corner bite region around v*/)
	fromP, toP := landingA(), landingB() // the two rail landings straddling the small bite
	v := bittenCornerVertex()            // inside the SMALL sub-path
	tol := 0.02

	far, ok := farPathSegs(ch, segs, fromP, toP, v, tol)

	if !ok {
		t.Fatalf("farPathSegs: expected the larger-area far sub-path, got ok=false")
	}
	if pathHasVertex(far, v, tol) {
		t.Fatalf("farPathSegs kept the sub-path containing the bitten vertex v=%v (small bite); want the larger far side", v)
	}
	if a := stdmath.Abs(chartSignedArea(ch, far)); a < stdmath.Abs(chartSignedArea(ch, complementOf(t, segs, far))) {
		t.Fatalf("kept sub-path area %.4f is smaller than its complement — area-primary pick failed", a)
	}
}
```

Also add a **reduction** assertion: on a B3-shaped clean loop, `farPathSegs` returns the same segs the exclude-v version returns (area pick agrees with exclude-v).

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestFarPathSegs -v`
Expected: FAIL — `farPathSegs` signature lacks `ch`; `undefined: chartSignedArea`.

- [ ] **Step 3: Implement area-primary `farPathSegs` + `chartSignedArea`**

```go
// chartSignedArea is the shoelace area of the path's chart-projected polyline (arcs sampled at their
// midpoint). Positive = CCW in the chart. On a cylinder (θ,z) chart this is the developed area, whose
// SIGN is the winding used by the orientation guard (C4) and whose MAGNITUDE ranks the two sub-paths.
func chartSignedArea(ch hostChart, path []endSeg) float64 {
	var sum float64
	for _, s := range path {
		a, b := ch.to2(s.from), ch.to2(s.to)
		sum += float64(a.X()*b.Y() - b.X()*a.Y())
		if s.arc { // account for the arc bulge via its midpoint (two triangles)
			m := ch.to2(s.mid)
			sum += float64(a.X()*m.Y()-m.X()*a.Y()) + float64(m.X()*b.Y()-b.X()*m.Y()) -
				float64(a.X()*b.Y()-b.X()*a.Y())
		}
	}
	return 0.5 * sum
}

// farPathSegs returns the surviving "far" boundary of L* between the two rail landings fromP→toP: the
// larger-enclosed-area sub-path (a runout bites a SMALL corner, so the far path is the larger region —
// mirrors cornerBiteArea, _farrunout.go:129), oriented fromP→toP. The exclude-v test is kept as a
// cross-check: the larger sub-path must NOT contain the bitten vertex v; disagreement ⇒ reject.
func farPathSegs(ch hostChart, segs []endSeg, fromP, toP, v math.Point3, tol float64) ([]endSeg, bool) {
	ring := insertSplits(segs, []math.Point3{fromP, toP}, tol)
	i, j := indexOfSegFrom(ring, fromP, tol), indexOfSegFrom(ring, toP, tol)
	if i < 0 || j < 0 {
		return nil, false // a rail landing point does not lie on L* — cannot close
	}
	fwd, bwd := segsForward(ring, i, j), reverseEndSegs(segsForward(ring, j, i))
	far := fwd
	if stdmath.Abs(chartSignedArea(ch, bwd)) > stdmath.Abs(chartSignedArea(ch, fwd)) {
		far = bwd
	}
	if pathHasVertex(far, v, tol) {
		return nil, false // area pick disagrees with exclude-v (the kept side holds the bite) — reject
	}
	return far, true
}
```

Thread `ch` from `retrimCornerHost` (it builds the host chart once and passes it to both `armRulingEnd` and `farPathSegs`). `segsForward(ring, j, i)` reversed gives the `fromP→toP` orientation for the backward side.

- [ ] **Step 4: Run the tests + reduction gate**

Run: `go test ./kernel/ops -run 'TestFarPathSegs|RetrimCurvedHost|B3CurvedArmWeld|B3VolumeRegression' -v`
Expected: PASS (larger-area pick green; B3 far-path identical — B3's far side is both larger-area and v-free, so the pick is unchanged).

- [ ] **Step 5: Corpus + lint**

Corpus → **55** (N7 gated by C4 next). `go build/vet/gofmt/golangci-lint` clean.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/fillet_curved_retrim_loop.go kernel/ops/fillet_curved_retrim_test.go
git commit -m "feat(blend): area-primary far-path on a boolean-cut loop (C3)"
```

---

### Task N5: Fail-loud host chart signed-area closure invariant (C4 / R.3)

The host analogue of Slice A's spherical-excess (Gauss–Bonnet) closure: gate the assembled retrim loop before it becomes a face. (1) endpoints coincide within `res.Weld()·r` and the cycle is a single non-self-intersecting loop; (2) chart signed-area balance `A_kept + A_bite = A(L*)` (residual > `res.Weld()·r·diam(L*)` ⇒ reject); (3) `sign(A_kept)` matches the host's outward orientation (flipped ⇒ reject — the tessellation-corruption / inside-out guard); (4) bitten-vertex partition (v ∈ discarded, v ∉ kept). Any failure ⇒ do-no-harm floor.

**Files:**
- Create: `kernel/ops/fillet_curved_host_closure.go` (`hostRetrimValid` + helpers)
- Modify: `kernel/ops/fillet_curved_retrim.go` (`retrimCornerHost` calls the gate before returning)
- Test: `kernel/ops/fillet_curved_host_closure_test.go`

**Interfaces:**
- Consumes: `hostChart`, `chartSignedArea`, `endSeg`, `Resolution.Weld()`, `cornerWeld{center,radius}`, the bitten-loop segs.
- Produces:
  - `func hostRetrimValid(ch hostChart, star, kept, bite []endSeg, v math.Point3, res Resolution, r float64) bool` — the four-part gate. `star` = original `L*` segs; `kept` = assembled retrim loop (rails + far); `bite` = the discarded near sub-path (rails + near path).

- [ ] **Step 1: Write the failing closure test**

Two cases: (a) the correct N7-shaped assembled loop passes; (b) a **mutated** loop with the far-path reversed (inverted orientation) is REJECTED — the inside-out guard that area alone (orientation-blind) cannot catch.

```go
func TestHostRetrimValid_AcceptsBalancedRejectsFlipped(t *testing.T) {
	ch := newCylChart(geom.MustCylinder(originP(), zAxis(), xRef(), 50))
	star := notchWindowLoop(t)                 // L* (inner notch), CW hole in chart
	kept := assembledRetrim(t, star)           // rails + larger far path, correct orientation
	bite := discardedBite(t, star)             // rails + small near path
	v := bittenCornerVertex()
	res := testResolution()
	r := 5.0

	if !hostRetrimValid(ch, star, kept, bite, v, res, r) {
		t.Fatalf("hostRetrimValid rejected a balanced, correctly-oriented retrim")
	}
	flipped := reverseEndSegs(kept) // inverts sign(A_kept) — inside-out face
	if hostRetrimValid(ch, star, flipped, bite, v, res, r) {
		t.Fatalf("hostRetrimValid accepted a FLIPPED (inside-out) loop — orientation guard failed")
	}
}

func TestHostRetrimValid_RejectsAreaImbalance(t *testing.T) {
	// kept+bite whose chart areas do NOT sum to A(L*) (a stray crossing / wrong-side pick)
	if hostRetrimValid(chartFixture(t), starFixture(t), keptTooSmall(t), biteFixture(t), vFixture(), testResolution(), 5.0) {
		t.Fatalf("hostRetrimValid accepted an area-imbalanced partition")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops -run TestHostRetrimValid -v`
Expected: FAIL — `undefined: hostRetrimValid`.

- [ ] **Step 3: Implement `hostRetrimValid`**

Create `kernel/ops/fillet_curved_host_closure.go`:

```go
// hostRetrimValid is the host-side closure certificate (derivation R.3): the flat-chart analogue of
// Slice A's spherical-excess weld check. It gates the assembled corner-host loop before it becomes a
// face, so a mis-closed, imbalanced, or inverted retrim is declined to the do-no-harm floor rather
// than corrupting every downstream tessellation consumer. ch is the host chart; star=L* original segs,
// kept=assembled retrim loop, bite=discarded near sub-path; v=bitten vertex; r=fillet radius.
func hostRetrimValid(ch hostChart, star, kept, bite []endSeg, v math.Point3, res Resolution, r float64) bool {
	tol := res.Weld() * r
	if !loopCloses(kept, tol) || !loopCloses(bite, tol) {
		return false // an open cycle would emit a cracked face
	}
	aStar, aKept, aBite := chartSignedArea(ch, star), chartSignedArea(ch, kept), chartSignedArea(ch, bite)
	if stdmath.Abs(aKept+aBite-aStar) > tol*chartDiameter(ch, star) {
		return false // area imbalance: a wrong-side pick or a stray crossing (residual carries the value)
	}
	if !sameSign(aKept, aStar) {
		return false // flipped winding vs the host loop's outward orientation — inside-out face
	}
	return !pathHasVertex(kept, v, tol) && pathHasVertex(bite, v, tol) // v ∈ discarded, ∉ kept
}
```

`loopCloses(path, tol)` = consecutive `to`/`from` coincide and last-to == first-from within tol; `chartDiameter(ch, star)` = max pairwise chart-point distance (or bounding-box diagonal) of `L*`; `sameSign(a,b)` = `a*b > 0`. Each ≤ 20 lines, explicit types, no bare constants. **Self-intersection**: reuse the existing planar loop-simplicity check if one exists in `kernel/ops`; otherwise a chart-segment pairwise non-crossing test (2 levels max). Grep for an existing `selfIntersect`/`simpleLoop` helper before adding one (no duplication).

- [ ] **Step 4: Gate `retrimCornerHost`**

In `retrimCornerHost`, build the host chart once (`ch, ok := hostChartFor(host.Geometry())`; reject if `!ok`), pass it to `armRulingEnd`/`farPathSegs`, then after assembling `kept` and the discarded `bite` (rails + the near sub-path), call the gate; on failure return `nil,false`:

```go
	ch, ok := hostChartFor(host.Geometry())
	if !ok {
		return nil, false // host is neither a plane nor a cylinder — no chart, decline
	}
	far, ok := farPathSegs(ch, segs, outerB, outerA, v, tol)
	if !ok {
		return nil, false
	}
	kept := append(append([]endSeg{railA}, reverseEndSegs([]endSeg{railB})...), far...)
	bite := nearBite(segs, outerA, outerB, railA, railB, v, tol) // rails + the discarded near sub-path
	if !hostRetrimValid(ch, segs, kept, bite, v, res, w.radius) {
		return nil, false // closure certificate failed — do-no-harm floor
	}
	return kept, true
```

`nearBite` mirrors `farPathSegs` but returns the *smaller*/v-containing sub-path closed by the two rails (reuse the split ring; do not recompute the area). Keep `retrimCornerHost` ≤ 20 lines by extracting the assembly into a small `assembleKept`/`assembleBite` pair if needed.

- [ ] **Step 5: Run closure tests + reduction gate**

Run: `go test ./kernel/ops -run 'TestHostRetrimValid|RetrimCurvedHost|B3CurvedArmWeld|B3VolumeRegression' -v`
Expected: PASS — the flipped/imbalanced loops reject; B3's retrim passes the certificate (it is balanced and correctly oriented by construction).

- [ ] **Step 6: Corpus + lint**

Corpus → **55** (N7 now closes internally but the corpus flip is asserted/verified in N6 with the full oracle gate). `go build/vet/gofmt/golangci-lint` clean.

- [ ] **Step 7: Commit**

```bash
git add kernel/ops/fillet_curved_host_closure.go kernel/ops/fillet_curved_retrim.go kernel/ops/fillet_curved_host_closure_test.go
git commit -m "feat(blend): fail-loud host chart signed-area closure invariant (C4)"
```

---

### Task N6: Integrate → green N7; per-face + volume + family verification

With C0–C4 in place the N7 corner should retrim and weld end-to-end. This task asserts the whole-body + per-face oracle gate, the VOLUME regression (the orientation guard — area is orientation-blind), confirms B3 and the M1–M4 tripwire, sweeps the convex boolean-cut family for greens/clean-declines, and flags any interior-weld-runout member for decline (out of scope per the derivation).

**Files:**
- Modify: `model/feature/…` the `TestOCCTBlendSimple` corpus expectations if N7 needs its whole-body area registered (check whether N7 is already listed with `61222.9`; if declared `skip`/`pending`, flip it to an asserting expectation).
- Create/Extend: `kernel/ops/fillet_curved_weld_test.go` — `TestFilletEdges_N7RetrimWeld` (per-face faithfulness) + `TestFilletEdges_N7VolumeRegression`.
- Test: the DRAWEXE N7 recipe under `test-utilities/occt-blend/` (already vendored) is the oracle for any value.

**Interfaces:**
- Consumes: the whole assembled weld via `filletResolvedEdges` on the N7 fixture; the corpus harness; `getMass`/volume helper used by `B3VolumeRegression`.
- Produces: N7 green in the corpus (55→**56**); a per-face + volume regression test; the family-sweep finding recorded in the ledger.

- [ ] **Step 1: Write the failing N7 per-face + volume test**

Model the N7 fixture in-kernel (the same `cylinder R50 h130 − box notch`, r=5 blend on the three edges) exactly as the corpus builds it, then assert the per-type face areas from the oracle table and the whole-body volume:

```go
func TestFilletEdges_N7RetrimWeld(t *testing.T) {
	body := buildN7Blended(t) // cyl R50 h130 − notch, AddFillet r=5 on s_5,s_4,s_10, Recompute
	assertWatertight(t, body)

	// per-type faithfulness (topology, not just Σ — the M3/S7/B3 lesson), fixture frame, tol = 1% Gauss:
	assertFaceAreaNear(t, body, faceTypeCylinder, []float64{546.695, 195.464}) // s_4 (z∈[15,80]) + s_10
	assertFaceAreaNear(t, body, faceTypeTorus, []float64{212.306})             // s_5
	assertFaceAreaNear(t, body, faceTypeSphere, []float64{90.194})             // corner (E=3.608)
	assertFaceAreaNear(t, body, faceTypePlane, wantsPlanes{517.428, 1606.89})  // the two retrimmed corner hosts
	assertWallInnerLoopBitten(t, body, 38033.8)                                // wall keeps its inner notch loop, retrimmed

	if a := totalArea(t, body); stdmath.Abs(a-61222.9)/61222.9 > 0.01 {
		t.Fatalf("N7 whole-body area %.3f; want 61222.9 within 1%% (DRAWEXE oracle)", a)
	}
}

func TestFilletEdges_N7VolumeRegression(t *testing.T) {
	body := buildN7Blended(t)
	got := solidVolume(t, body)               // getMass-equivalent
	want := drawexeN7Volume                    // DRAWEXE `vprops result` on the N7 recipe (record the number)
	if stdmath.Abs(got-want)/want > 0.01 {
		t.Fatalf("N7 volume %.3f vs oracle %.3f (>1%%): an inverted retrim face inverts the normal", got, want)
	}
}
```

Before writing the numbers, **run the DRAWEXE N7 recipe** to (a) confirm the 12 face areas above and (b) record `vprops result` for the volume oracle — do not hard-code an unverified volume.

- [ ] **Step 2: Run to verify it fails (or that N7 still declines)**

Run: `go test ./kernel/ops -run 'TestFilletEdges_N7' -v`
Expected: initially FAIL if any integration seam is loose (e.g. `curvedHostFaces` order, the far-path chart threading), OR PASS if C0–C4 already compose. If FAIL, this is the integration debugging surface — fix the seam (do NOT loosen a gate).

- [ ] **Step 3: Flip the corpus expectation + verify the count**

Ensure `model/feature`'s `TestOCCTBlendSimple/N7` asserts `61222.9` (not skipped). Run the corpus count:
Run: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
Expected: **56**.

- [ ] **Step 4: B3 + M1–M4 tripwire non-regression**

Run: `go test ./kernel/ops -run 'B3CurvedArmWeld|B3VolumeRegression|B3UnconsumedPickDeclines|CurvedCorner|CurvedSetbackRail' -v` → PASS.
Run the corpus and confirm S1/S4/S7/T1/T4/T7 still PASS and every other grid is byte-identical (diff the corpus output vs base `215c1fb6`): only N7 flips PASS.

- [ ] **Step 5: Convex boolean-cut family sweep + interior-weld flag**

For each other corpus member with a boolean-cut host at a convex trihedral corner whose corner solve passes: run it and record green-or-clean-decline (never a wrong solid). **Flag** any member with two blended edges sharing a far vertex (interior-weld runout — out of scope, R.1a): confirm it cleanly DECLINES (the far-vertex authority in `armRulingEnd` returns false → floor), and record it as follow-up debt. Record the sweep result in the ledger.

- [ ] **Step 6: Full local gate**

Run: `go build ./... && go vet ./kernel/... && gofmt -l kernel/ops && golangci-lint run` → clean.
Run: `go test ./kernel/ops ./model/feature` → PASS.

- [ ] **Step 7: Commit**

```bash
git add kernel/ops/fillet_curved_weld_test.go model/feature
git commit -m "feat(blend): green OCCT simple/N7 — boolean-cut host retrim (corpus 55->56)"
```

---

## Verification (whole slice)

- **N7 whole-body** 61222.9 (`deps=1%`) via `TestOCCTBlendSimple/N7` → PASS; corpus 55→**56**.
- **Per-type faithfulness** (topology, not just Σ): cyl arms 546.695 (z∈[15,80]) + 195.464, torus 212.306, sphere 90.194, retrimmed planes 517.428 + 1606.89, wall 38033.8 with its inner loop bitten.
- **B3 faithful** — B3 corpus subtest + per-face weld areas + watertight + volume all unchanged (C0–C4 reduce to current code on the clean wedge; verify base-vs-head corpus diff = only N7 flips).
- **N7 VOLUME regression** — the orientation guard (area is orientation-blind; a flipped retrim face inverts the normal and fails volume at matching area).
- **M1–M4 tripwire** — S1/S4/S7/T1/T4/T7 byte-identical; every other grid byte-identical.
- **DRAWEXE oracle** — the vendored N7 recipe (`drawenv.sh`) is ground truth for any per-face / volume value; record `vprops result` before hard-coding the volume.
- **NO PR** — corpus 56/195, not whole-green.

## Out of scope (carried, per the derivation)

- **Interior-weld runout** — a filleted edge whose far end is *another* blended corner (σ_far interior, not on ∂H). N7's s_4/s_5/s_10 all run out onto unfilleted faces, so N7 is safe; N6 flags any corpus member with two blended edges sharing a far vertex → clean decline, recorded as follow-up.
- **Concave / non-tangent hosts** — untouched; this generalizes the *trim*, not the corner-solve family (concave-bore N1/O1 and B7-class 270° corner remain separate follow-up slices).
- **BSpline corner parity** — OCCT emits N7's corner as a BSpline; our analytic sphere-triangle reproduces its area (90.194, E=3.608). The retrim does not touch it; face 90.194 is gated to catch a rail mis-placement carried from the corner solve.

## References

- `.superpowers/sdd/n7-retrim-generalization-derivation.md` — R.0–R.3, the chart termination, the area-balance closure, the pitfalls, the DRAWEXE N7 ground truth.
- `docs/superpowers/specs/2026-07-16-n7-retrim-generalization-design.md` — the approved design (C0–C4).
- `.superpowers/sdd/m5-weld-setback-retrim-derivation.md` — the Slice A B3 retrim this generalizes.
- `docs/superpowers/plans/2026-07-16-curved-arm-fillet-weld.md` — the Slice A plan (task/format template).
