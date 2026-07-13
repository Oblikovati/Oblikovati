# Mid-Span Obstacle Fillet-Blend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A constant-radius edge fillet whose planar host face is notched by a through-feature (e.g. an elliptical column base) stops emitting a malformed protruding-hole face and instead emits a certified 4-sided `FillSurface` patch, making the 13 protruding-hole corpus cases watertight.

**Architecture:** Extend the existing `CornerBlendProvider` seam (ADR-4) with an *obstacle variant*. The `ops.Validate` hole-containment watchdog (commit 26f2da61) is the trigger: when a coplanar hole crosses the receded fillet boundary, the fillet rebuild (a) clips the hole into the receded outer loop (KEML → a single notched face) and (b) dispatches an obstacle `CornerBlendRequest` to `resolveCornerBlend`, which builds a `FillSurface` over four rails — the two abutting cylinder-wing section arcs (reused by pointer for G1-by-identity), the wall-tangent line, and the exact obstacle-rim sub-arc — with continuity orders G1/G1/G1/**G0** (the rim is a sharp base crease). The rebuild/assembly layer stays agnostic: it receives a `filletFace` exactly as today.

**Tech Stack:** Go (kernel `oblikovati.org/kernel/ops`, `oblikovati.org/kernel/geom`, `oblikovati.org/math`, `oblikovati.org/topo`); NURBS layer (`CoonsFill`/`FillSurface`/`MatchSurface`/knot-insertion/rational-ellipse); OCCT DRAWEXE as the area oracle; `go test`.

## Global Constraints

- Branch `feat/occt-blend-parity-corpus`. **NO PR until the whole corpus is green** — accumulate, commit per task/slice.
- Every new `.go` file carries `// SPDX-License-Identifier: GPL-2.0-only` as line 1 (run `scripts/add-spdx-headers.py` if unsure).
- Functions 4–20 lines; files <500 lines; explicit types (no `any`/`interface{}` except the one justified `testingT` case); early returns; max 2 indentation levels; no code duplication.
- TDD: every new function gets a test; use named fakes, not inline stubs. Tests are F.I.R.S.T.
- **Reuse existing geom** (`CoonsFill`, `FillSurface`, `MatchSurface`, `insertKnotHomog`/refine, rational-ellipse NURBS, `NewBSplineCurveUniformWeights`, `bspline_compatible.go` make-compatible). Do **not** reinvent the fitter.
- The trihedral/planar/junction fillet paths stay **byte-for-byte unchanged**: the obstacle path is reached only when a hole crosses the receded boundary. Junction requests (`ObstacleFeature == nil`) behave exactly as today.
- Corpus scoreboard: the 13 cases (S1 S3 S4 S9 T1 T4 T6 T7 T9 U1 U3 U4 X3) migrate `FAIL(faulty)` → `PASS` and **never** regress any currently-passing case.
- **Gate = 1% relative on TOTAL BODY area** (`assertArea` deps, `model/feature/occtparity/checkprops.go`), applied to `ops.BodyGeometryProperties(body).Area`. A stock bilinear `FillSurface` contributes +0.31% of body area for T6 → passes the 1% gate, trips the non-failing 0.1% `driftWarnRel` log. The T6 patch oracle (156.364) and body oracle (6871.45) come from OCCT DRAWEXE.
- Before "done": full local `go test ./kernel/ops/... ./model/feature/occtparity/...` + `golangci-lint run` green.

## Ubiquitous language (match the spec + code exactly)

| Term | Meaning |
|---|---|
| **Obstacle** | A through-feature (column) whose base rim notches the fillet's planar host face. |
| **Rim curve** | The obstacle's boundary curve in the host-face plane (T6: the base ellipse). |
| **Receded boundary** | The line the fillet pulled the host face's outer loop back to (T6: `y=-7` at `z=0`). |
| **Nodes P±** | The two points where the rim crosses the receded boundary; the C0 junctions to the cylinder wings. |
| **Wing** | A plain rolling-ball cylinder face on either side of the obstacle band (OCCT result_3/7). |
| **Patch** | The 4-sided fill over the obstacle band (OCCT result_5). |
| **Notched face** | The host face after KEML: outer loop + rim sub-arc merged into ONE loop (OCCT result_8). |

## File Structure

**New files:**
- `kernel/ops/corner_blend_obstacle.go` — `ObstacleFeature` type; `bsplineObstacleProvider` (`Name`/`Fits`/`Build`); rail assembly; certify with a rotating reference normal. (Target <300 lines; split rail assembly into `corner_blend_obstacle_rails.go` if it approaches the cap.)
- `kernel/ops/fillet_obstacle_detect.go` — `detectObstacle`: given a host face's projected outer + hole loops and the rim, find the crossing, Nodes, and rim sub-arc; pure geom/math, no assembly.
- `kernel/ops/fillet_obstacle_merge.go` — `mergeHoleIntoNotch`: KEML clip of the crossing hole loop into the receded outer loop (produces the single notched `filletLoop`).
- Tests: `kernel/ops/corner_blend_obstacle_test.go`, `kernel/ops/fillet_obstacle_detect_test.go`, `kernel/ops/fillet_obstacle_merge_test.go`.

**Modified files:**
- `kernel/ops/corner_blend.go` — add optional `ObstacleFeature *ObstacleFeature` field to `CornerBlendRequest`.
- `kernel/ops/fillet_faces.go` — split the wing cylinder at the obstacle band; wire `detectObstacle` + `mergeHoleIntoNotch` + `resolveCornerBlend` into the face rebuild; append the patch as a `filletFace`.
- `kernel/ops/validate.go:63` — fold `HolesContained` into `Valid`.
- `model/feature/occtparity/fillet_hole_containment_test.go` — flip the tripwire to assert the filleted body is watertight.
- `model/feature/occtparity/fillet_midspan_obstacle_test.go` (**new test file**) — per-case area + watertightness gate for the 13 cases.

---

### Task 1: `ObstacleFeature` type + request extension

**Files:**
- Modify: `kernel/ops/corner_blend.go` (add field + type near `CornerBlendRequest`, ~line 46)
- Create: `kernel/ops/corner_blend_obstacle.go`
- Test: `kernel/ops/corner_blend_obstacle_test.go`

**Interfaces:**
- Consumes: `CornerBlendRequest`, `CornerBlendProvider`, `CornerBlendKind`, `BlendKindBSpline` (corner_blend.go).
- Produces: `type ObstacleFeature struct { RimCurve geom.Curve3; Nodes [2]math.Point3; ReceddedBoundary geom.LineSegment; HostPlane geom.Plane; WingStart, WingEnd geom.Curve3; WallLine geom.Curve3 }`; `type bsplineObstacleProvider struct{}` with `Name() CornerBlendKind`.

- [ ] **Step 1: Write the failing test**

```go
// kernel/ops/corner_blend_obstacle_test.go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import "testing"

func TestObstacleProviderName(t *testing.T) {
	var p bsplineObstacleProvider
	if got := p.Name(); got != BlendKindBSpline {
		t.Errorf("obstacle provider Name() = %q, want %q", got, BlendKindBSpline)
	}
}

func TestObstacleRequestNilByDefault(t *testing.T) {
	req := CornerBlendRequest{}
	if req.ObstacleFeature != nil {
		t.Errorf("a default CornerBlendRequest must carry no ObstacleFeature (junction request unchanged)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/vmiguel/git/oblikovati-workspace/Oblikovati && go test ./kernel/ops/ -run 'TestObstacleProviderName|TestObstacleRequestNilByDefault' -v`
Expected: FAIL — `undefined: bsplineObstacleProvider`, `req.ObstacleFeature undefined`.

- [ ] **Step 3: Add the optional field to the request**

In `kernel/ops/corner_blend.go`, inside `CornerBlendRequest` (after `Setback Resolution`):

```go
	// ObstacleFeature, when non-nil, marks this as a MID-SPAN OBSTACLE request (ADR-4): a straight
	// fillet whose planar host face is notched by a through-feature, NOT a junction of arms. Junction
	// requests leave it nil and behave exactly as before. Only the obstacle provider reads it.
	ObstacleFeature *ObstacleFeature
```

- [ ] **Step 4: Create the provider file with the type and Name**

```go
// kernel/ops/corner_blend_obstacle.go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// ObstacleFeature carries the geometry of a mid-span obstacle patch (ADR-4, spec §3): the obstacle's
// rim curve, the two Nodes P± where it crosses the receded fillet boundary, and the four neighbour
// pieces the 4-sided FillSurface must weld to. WingStart/WingEnd are the section arcs of the abutting
// cylinder wings AT the Nodes — reused BY VALUE from the wing faces so the patch is G1 to them by
// identity and no T-junction crack appears. WallLine is the fillet's wall-tangent seam; HostPlane and
// the wing/rim are the neighbour surfaces the certify measures G1/G0 against.
type ObstacleFeature struct {
	RimCurve             geom.Curve3     // obstacle base rim (T6: the base ellipse), full curve
	Nodes                [2]math.Point3  // P-, P+ : rim ∩ receded boundary
	WingStart, WingEnd   geom.Curve3     // cylinder-wing section arcs at P-, P+ (the shared end rails)
	WallLine             geom.Curve3     // wall-tangent seam between the Nodes' wall points
	HostPlane            geom.Plane      // the notched host face's plane (for the rim-side G0 side)
}

// bsplineObstacleProvider is the obstacle-variant tier of the corner-blend engine: a single Coons
// FillSurface over the four rails, certified. It Fits only obstacle requests; junction requests fall
// through to the junction providers untouched.
type bsplineObstacleProvider struct{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (bsplineObstacleProvider) Name() CornerBlendKind { return BlendKindBSpline }
```

- [ ] **Step 5: Run tests to verify pass**

Run: `go test ./kernel/ops/ -run 'TestObstacleProviderName|TestObstacleRequestNilByDefault' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_blend.go kernel/ops/corner_blend_obstacle.go kernel/ops/corner_blend_obstacle_test.go
git commit -m "feat(fillet): add ObstacleFeature request variant + obstacle provider skeleton (ADR-4)"
```

---

### Task 2: Obstacle detector — crossing, Nodes, rim sub-arc

**Files:**
- Create: `kernel/ops/fillet_obstacle_detect.go`
- Test: `kernel/ops/fillet_obstacle_detect_test.go`

**Interfaces:**
- Consumes: `planeProjector` (earclip.go), `pointInLoop2D` (union_holes.go), `Resolution`/`ResolutionForPoints`, `geom.LineSegment`, `geom.Curve3.PointAt`.
- Produces: `func rimCrossings(rim []math.Point2, boundary math.Line2, res Resolution) []crossing` where `type crossing struct { T float64; P math.Point2 }` (rim parameter + point); `func obstacleNodes(rim []math.Point2, boundary math.Line2, res Resolution) ([2]int, bool)` returning the two crossing indices bracketing the dip, `ok=false` when fewer than 2 crossings (tangential/no dip → honest-reject upstream).

- [ ] **Step 1: Write the failing test** (T6 ellipse ∩ line y=-7 → two crossings at x=±10.712)

```go
// kernel/ops/fillet_obstacle_detect_test.go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"math"
	"testing"

	m "oblikovati.org/math"
)

// sampleEllipse returns n points of the T6 base ellipse (a=15,b=10) in the z=0 host plane.
func sampleEllipse(n int) []m.Point2 {
	pts := make([]m.Point2, n)
	for i := 0; i < n; i++ {
		t := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = m.P2(15*math.Cos(t), 10*math.Sin(t))
	}
	return pts
}

func TestRimCrossingsT6(t *testing.T) {
	rim := sampleEllipse(720)
	boundary := m.Line2{P: m.P2(0, -7), Dir: m.V2(1, 0)} // the receded boundary y=-7
	res := ResolutionForSize(50)
	cs := rimCrossings(rim, boundary, res)
	if len(cs) != 2 {
		t.Fatalf("ellipse ∩ y=-7 should have 2 crossings, got %d", len(cs))
	}
	xs := []float64{cs[0].P.X, cs[1].P.X}
	for _, x := range xs {
		if math.Abs(math.Abs(x)-10.712142) > 0.05 {
			t.Errorf("crossing x=%.4f, want ±10.712142", x)
		}
	}
}

func TestObstacleNodesTangentRejected(t *testing.T) {
	rim := sampleEllipse(720)
	boundary := m.Line2{P: m.P2(0, -10), Dir: m.V2(1, 0)} // tangent to the ellipse bottom
	res := ResolutionForSize(50)
	if _, ok := obstacleNodes(rim, boundary, res); ok {
		t.Errorf("a rim tangent to the boundary must be rejected (no dip, no patch)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run 'TestRimCrossingsT6|TestObstacleNodesTangentRejected' -v`
Expected: FAIL — `undefined: rimCrossings`, `obstacleNodes`, and possibly `math.Line2`/`math.V2` (confirm the 2D line type exists; if the kernel's 2D line is named differently, use that name and update the test accordingly — check `grep -rn "type Line2" kernel/`).

- [ ] **Step 3: Implement the detector**

```go
// kernel/ops/fillet_obstacle_detect.go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	stdmath "math"

	"oblikovati.org/math"
)

// crossing is one intersection of the obstacle rim (as a sampled polyline) with the receded boundary
// line: the rim index just before it and the intersection point in host-plane 2D.
type crossing struct {
	I int         // rim polyline index whose segment [I, I+1] crosses the boundary
	P math.Point2 // the crossing point
}

// rimCrossings returns the boundary crossings of the closed rim polyline, ordered as they appear along
// the rim. A crossing is a SIGN CHANGE of the signed distance to the boundary line larger than the
// model weld — so a vertex merely grazing the boundary (|d| ≤ weld on both sides) is NOT a crossing
// (the tangency guard, spec §Numerical pitfalls).
func rimCrossings(rim []math.Point2, boundary math.Line2, res Resolution) []crossing {
	tol := res.Weld()
	var out []crossing
	n := len(rim)
	for i := 0; i < n; i++ {
		a, b := rim[i], rim[(i+1)%n]
		da, db := boundary.SignedDistance(a), boundary.SignedDistance(b)
		if da > tol && db < -tol || da < -tol && db > tol {
			out = append(out, crossing{I: i, P: lerpAtZero(a, b, da, db)})
		}
	}
	return out
}

// lerpAtZero returns the point on segment a→b where the signed distance crosses zero.
func lerpAtZero(a, b math.Point2, da, db float64) math.Point2 {
	t := da / (da - db)
	return math.P2(a.X+t*(b.X-a.X), a.Y+t*(b.Y-a.Y))
}

// obstacleNodes returns the two rim crossing indices bracketing the dip past the boundary, or
// ok=false when the rim does not genuinely cross twice (tangential touch or no dip → honest-reject,
// ADR-3). Exactly two crossings is the single-dip case this slice handles; >2 (a rim weaving across)
// is a tracked follow-up and also returns ok=false here so the caller honest-rejects rather than
// mis-building.
func obstacleNodes(rim []math.Point2, boundary math.Line2, res Resolution) ([2]crossing, bool) {
	cs := rimCrossings(rim, boundary, res)
	if len(cs) != 2 {
		return [2]crossing{}, false
	}
	return [2]crossing{cs[0], cs[1]}, true
}

// dipsPast reports whether the rim actually dips PAST the boundary between the two crossings (into the
// fillet band), vs. bulging away — the mid-arc sample must be on the fillet side. side is +1 when the
// fillet band is on the negative-signed-distance side of the boundary.
func dipsPast(rim []math.Point2, c0, c1 crossing, boundary math.Line2, side float64) bool {
	mid := rim[(c0.I+1+((c1.I-c0.I+len(rim))%len(rim))/2)%len(rim)]
	return side*boundary.SignedDistance(mid) > 0
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./kernel/ops/ -run 'TestRimCrossingsT6|TestObstacleNodesTangentRejected' -v`
Expected: PASS. If `math.Line2`/`SignedDistance` don't exist under those names, grep the kernel math package (`grep -rn "SignedDistance\|type Line2\|func.*Line2" math/`) and adapt to the real names — do not invent; the plan's tolerance and crossing logic hold regardless of the type name.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_obstacle_detect.go kernel/ops/fillet_obstacle_detect_test.go
git commit -m "feat(fillet): obstacle detector — rim/boundary crossings and Node bracketing with tangency guard"
```

---

### Task 3: Rail assembly — the four `FillSurface` curves

**Files:**
- Create: `kernel/ops/corner_blend_obstacle_rails.go`
- Test: append to `kernel/ops/corner_blend_obstacle_test.go`

**Interfaces:**
- Consumes: `ObstacleFeature`, `geom.BSplineCurve`, `geom.FillSide`, ellipse-arc extraction (`geom` rational ellipse + knot insertion), `NewBSplineCurveUniformWeights`, `bspline_compatible.go` make-compatible (`grep -rn "func.*[Cc]ompatible" kernel/geom/` for the exact name).
- Produces: `func obstacleRails(of *ObstacleFeature) (c0, c1, d0, d1 geom.BSplineCurve, ok bool)` — c0=wall line, c1=rim sub-arc, d0=WingStart, d1=WingEnd, all made pairwise-compatible; `ok=false` if any wing curve is nil (the mandatory pointer nil-check) or the sub-arc extraction fails; `func obstacleSides(of *ObstacleFeature, wingL, wingR, wallPlane geom.BSplineSurface) [4]geom.FillSide` — orders {c0:1 (wall), c1:0 (rim, G0), d0:1 (wingL), d1:1 (wingR)}.

- [ ] **Step 1: Write the failing test** (rails close into a valid quad; nil wing → not ok)

```go
// append to kernel/ops/corner_blend_obstacle_test.go
func TestObstacleRailsRejectNilWing(t *testing.T) {
	of := &ObstacleFeature{ /* WingStart/WingEnd left nil */ }
	if _, _, _, _, ok := obstacleRails(of); ok {
		t.Errorf("obstacleRails must reject a nil wing pointer (regression-crack defense)")
	}
}

func TestObstacleSidesContinuityOrders(t *testing.T) {
	sides := obstacleSides(&ObstacleFeature{}, geom.BSplineSurface{}, geom.BSplineSurface{}, geom.BSplineSurface{})
	// side order: c0(v=0)=wall, c1(v=1)=rim, d0(u=0)=wingL, d1(u=1)=wingR
	want := [4]int{1, 0, 1, 1}
	for i, s := range sides {
		if s.Order != want[i] {
			t.Errorf("side %d Order = %d, want %d (rim must be G0)", i, s.Order, want[i])
		}
	}
}
```
(add `"oblikovati.org/kernel/geom"` to the test imports.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run 'TestObstacleRailsRejectNilWing|TestObstacleSidesContinuityOrders' -v`
Expected: FAIL — `undefined: obstacleRails`, `obstacleSides`.

- [ ] **Step 3: Implement rail assembly**

```go
// kernel/ops/corner_blend_obstacle_rails.go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import "oblikovati.org/kernel/geom"

// obstacleRails builds the four FillSurface boundary curves for the obstacle patch and makes them
// pairwise-compatible (c0/c1 share degree+knots, d0/d1 share degree+knots — FillSurface's precondition).
// It FIRST nil-checks the wing pointers: the end rails MUST be the abutting cylinder wings' section
// arcs (reused for G1-by-identity and to kill the T-junction crack). A missing wing ⇒ ok=false ⇒ the
// provider declines and the caller honest-rejects (ADR-3) rather than build a fresh, crack-inducing arc.
func obstacleRails(of *ObstacleFeature) (c0, c1, d0, d1 geom.BSplineCurve, ok bool) {
	if of == nil || of.WingStart == nil || of.WingEnd == nil || of.WallLine == nil || of.RimCurve == nil {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	d0, ok0 := asBSplineCurve(of.WingStart)
	d1, ok1 := asBSplineCurve(of.WingEnd)
	c0, ok2 := asBSplineCurve(of.WallLine)
	c1, ok3 := obstacleRimArc(of)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	c0, c1, ok = makeRailPair(c0, c1)
	if !ok {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	d0, d1, ok = makeRailPair(d0, d1)
	return c0, c1, d0, d1, ok
}

// obstacleRimArc extracts the rim sub-arc between the two Nodes as an exact B-spline curve — for a
// rational-quadratic ellipse this is knot-insertion at the two node parameters, keeping the middle
// (spec §3, exact conic sub-arc). Returns ok=false if the rim is not a supported analytic rim.
func obstacleRimArc(of *ObstacleFeature) (geom.BSplineCurve, bool) {
	return extractSubArc(of.RimCurve, of.Nodes[0], of.Nodes[1])
}

// obstacleSides returns the four FillSide continuity orders: wall (c0) and both wings (d0,d1) are G1;
// the RIM (c1) is G0 — the fillet meets the vertical obstacle wall at a SHARP base-rim crease, and
// forcing G1 to a vertical wall inflates the patch and makes a sliver (spec §Item 2, resolved: G0).
func obstacleSides(of *ObstacleFeature, wingL, wingR, wallPlane geom.BSplineSurface) [4]geom.FillSide {
	return [4]geom.FillSide{
		{Adjacent: wallPlane, AdjEdge: geom.VMinEdge, Order: 1}, // c0 wall  → G1
		{Order: 0}, // c1 rim → G0 (no ribbon)
		{Adjacent: wingL, AdjEdge: geom.UMinEdge, Order: 1}, // d0 wingL → G1
		{Adjacent: wingR, AdjEdge: geom.UMaxEdge, Order: 1}, // d1 wingR → G1
	}
}
```

Helper stubs to implement in the same file (each 4–20 lines), reusing geom:
- `asBSplineCurve(c geom.Curve3) (geom.BSplineCurve, bool)` — type-switch a `geom.Curve3` to `geom.BSplineCurve`, converting a `geom.LineSegment` via `NewBSplineCurveUniformWeights(1, [start,end], clamped-knots)` and an `geom.Arc3d` via its exact rational form (grep `kernel/geom/` for the arc→NURBS converter, e.g. `Arc3d.AsBSpline`/`conicToBSpline`; if none, degree-2 rational per Piegl-Tiller §7.3 — but check first, do not reinvent).
- `extractSubArc(c geom.Curve3, p0, p1 math.Point3) (geom.BSplineCurve, bool)` — invert the two node points to rim parameters (`geom` curve point-inversion), knot-insert at both, slice the middle control net. Reuse `insertKnotHomog`/refine.
- `makeRailPair(a, b geom.BSplineCurve) (geom.BSplineCurve, geom.BSplineCurve, bool)` — call the existing make-compatible (degree-elevate + knot-merge) so the pair shares degree/knots; return ok=false on failure.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./kernel/ops/ -run 'TestObstacleRails|TestObstacleSides' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/corner_blend_obstacle_rails.go kernel/ops/corner_blend_obstacle_test.go
git commit -m "feat(fillet): obstacle rail assembly — reuse wing arcs, wall line, exact rim sub-arc; rim side G0"
```

---

### Task 4: `bsplineObstacleProvider.Fits`/`Build` + certify (rotating-normal anti-fold)

**Files:**
- Modify: `kernel/ops/corner_blend_obstacle.go` (add `Fits`, `Build`, `certifyObstaclePatch`)
- Create: `kernel/ops/corner_blend_obstacle_certify.go`
- Test: append to `kernel/ops/corner_blend_obstacle_test.go`

**Interfaces:**
- Consumes: `obstacleRails`, `obstacleSides`, `geom.FillSurface`, `CornerBlendPatch`, `Certificate`, `geom.BSplineSurface.PointAt`/`DerivativesAt`/`NormalAt`.
- Produces: `func (bsplineObstacleProvider) Fits(req CornerBlendRequest) bool` (true iff `req.ObstacleFeature != nil`); `func (bsplineObstacleProvider) Build(req CornerBlendRequest) (CornerBlendPatch, Certificate, bool)`; `func certifyObstaclePatch(s geom.BSplineSurface, of *ObstacleFeature, scale Resolution) Certificate`.

- [ ] **Step 1: Write the failing test** — Build a T6 patch from real rails and assert (a) it builds, (b) certificate is Valid, (c) area within 1% of 156.364, (d) `NoFold` true.

```go
// append to kernel/ops/corner_blend_obstacle_test.go — uses a helper that assembles a real T6
// ObstacleFeature from analytic geometry (wing arcs = quarter circles r=6 at x=±10.712142, wall
// line at y=-13 z=-6, rim = ellipse a=15 b=10, nodes at (±10.712142,-7,0)). Put the builder in the
// test file as newT6Obstacle(t) — named fake geometry, not an inline stub.
func TestObstacleBuildT6AreaAndCert(t *testing.T) {
	of := newT6Obstacle(t)
	req := CornerBlendRequest{ObstacleFeature: of, Setback: ResolutionForSize(50)}
	var p bsplineObstacleProvider
	if !p.Fits(req) {
		t.Fatal("provider must Fit an obstacle request")
	}
	patch, cert, ok := p.Build(req)
	if !ok {
		t.Fatal("Build failed on a valid T6 obstacle")
	}
	if !cert.NoFold {
		t.Errorf("T6 patch must not fold (|S_u×S_v| stays ~137..212, never 0)")
	}
	if !cert.Valid(req.Setback) {
		t.Errorf("T6 patch certificate must be Valid, got %+v", cert)
	}
	area := surfaceArea(patch.Surface) // small Gauss-Legendre integrator, add as a test helper
	if rel := math.Abs(area-156.364) / 156.364; rel > 0.01 {
		t.Errorf("patch area %.3f vs oracle 156.364 (rel %.4f%% > 1%%)", area, rel*100)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run 'TestObstacleBuildT6AreaAndCert' -v`
Expected: FAIL — `Fits`/`Build`/`certifyObstaclePatch` undefined.

- [ ] **Step 3: Implement Fits/Build**

```go
// add to kernel/ops/corner_blend_obstacle.go
func (bsplineObstacleProvider) Fits(req CornerBlendRequest) bool { return req.ObstacleFeature != nil }

// Build fills the obstacle band with a single Coons FillSurface over the four rails (wing arcs reused
// by value → G1 by identity; wall G1; rim G0) and certifies it. A missing wing / failed rail / failed
// fill ⇒ ok=false so the tier moves on / the caller honest-rejects (ADR-3). The patch's loops are the
// four rails in order, wound to the fill's outward normal (assembleBody consumes them as today).
func (bsplineObstacleProvider) Build(req CornerBlendRequest) (CornerBlendPatch, Certificate, bool) {
	of := req.ObstacleFeature
	c0, c1, d0, d1, ok := obstacleRails(of)
	if !ok {
		return CornerBlendPatch{}, Certificate{}, false
	}
	wingL, okL := asBSplineSurface(of.WingStart) // the wing SURFACE for the G1 ribbon (see note)
	wingR, okR := asBSplineSurface(of.WingEnd)
	wall := planeAsBSplineSurface(of.HostPlaneWall()) // wall plane as a NURBS for MatchSurface
	if !okL || !okR {
		return CornerBlendPatch{}, Certificate{}, false
	}
	surf, err := geom.FillSurface(c0, c1, d0, d1, obstacleSides(of, wingL, wingR, wall))
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	cert := certifyObstaclePatch(surf, of, req.Setback)
	return CornerBlendPatch{Surface: surf, Loops: obstaclePatchLoops(of), Kind: BlendKindBSpline}, cert, true
}
```

Note on the G1 ribbon inputs: `FillSurface`'s `MatchSurface` needs the *neighbour surface* (`BSplineSurface`) per side, not just the shared curve. The wing is a `geom.Cylinder`; convert it to a NURBS band spanning the shared arc via the existing analytic-surface→NURBS path (`grep -rn "func.*Cylinder.*BSpline\|RebuildSurface" kernel/geom/`). The wall is a `geom.Plane` → a trivial degree-1 NURBS through the seam. Implement `asBSplineSurface`, `planeAsBSplineSurface`, `HostPlaneWall`, `obstaclePatchLoops` as small helpers; `obstaclePatchLoops` builds `[]filletLoop` sampling each rail (bit-identical stations to the wings — reuse the wing edge's discretization).

- [ ] **Step 4: Implement certify with the rotating reference normal**

```go
// kernel/ops/corner_blend_obstacle_certify.go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
)

// certifyObstaclePatch measures the patch's admissibility (spec §3 certify): G0 max deviation of the
// four rails from the fill, G1 angular deviation to the wall+wings (the rim side is G0 so excluded),
// and the anti-fold gate. NoFold uses a ROTATING reference normal — the dot of CONSECUTIVE-station
// normals — because a fillet's normal legitimately sweeps ~90° across the profile; a fixed-axis gate
// would false-positive on a perfectly regular patch. A fold is a normal REVERSAL (dot ≤ 0) between
// neighbouring stations, i.e. |S_u×S_v| passing through 0.
func certifyObstaclePatch(s geom.BSplineSurface, of *ObstacleFeature, scale Resolution) Certificate {
	nu, nv := 24, 24 // curvature-adaptive floor; densify if MaxAngleDev >> station residual
	noFold := true
	var prev geom.Vector3
	for i := 0; i <= nu; i++ {
		for j := 0; j <= nv; j++ {
			u, v := float64(i)/float64(nu), float64(j)/float64(nv)
			du, dv := s.DerivativesAt(u, v)
			n := du.Cross(dv)
			if n.Length() < scale.Weld() { // Jacobian ~0 ⇒ degenerate ⇒ fold
				noFold = false
			}
			if j > 0 && n.Dot(prev) <= 0 { // reversal vs the previous station along v
				noFold = false
			}
			prev = n
		}
	}
	return Certificate{
		Closed:      true, // rails form a closed quad by construction (consistentCorners in FillSurface)
		WeldsArms:   true, // all four sides spanned; obstacle has no "arms" but the flag means boundary-complete
		NoFold:      noFold,
		MaxDev:      obstacleMaxDev(s, of),
		MaxAngleDev: obstacleMaxAngleDev(s, of),
	}
}
```

Implement `obstacleMaxDev` (max distance of dense rail samples to the fill boundary; ~0 by G0 construction, guard against fill-compat drift) and `obstacleMaxAngleDev` (max angle between the fill normal and the wall/wing normals along their sides). Both 4–20 lines. Confirm `geom.Vector3.Cross/Dot/Length` names (`grep -rn "func.*Vector3.*Cross" math/`).

- [ ] **Step 5: Run to verify pass**

Run: `go test ./kernel/ops/ -run 'TestObstacleBuildT6AreaAndCert' -v`
Expected: PASS. If area is >1% (e.g. the bilinear Coons lands ~178 at the patch level), that is EXPECTED per the advisor — but the **corpus** gate is body-level, so this *patch-level* assertion should use the body tolerance context: keep the assertion at 1% patch only if it passes; if the stock Coons overshoots the patch, relax THIS test to assert body-level parity in Task 7 and here assert only `cert.Valid` + `NoFold` + area finite. (Do not fake the number — record the actual patch area in the commit message so Task 7 can decide if a FairSurface pass is needed.)

- [ ] **Step 6: Commit**

```bash
git add kernel/ops/corner_blend_obstacle.go kernel/ops/corner_blend_obstacle_certify.go kernel/ops/corner_blend_obstacle_test.go
git commit -m "feat(fillet): obstacle provider Build + certify (rotating-normal anti-fold); T6 patch"
```

---

### Task 5: KEML hole-into-notch merge (the notched host face)

**Files:**
- Create: `kernel/ops/fillet_obstacle_merge.go`
- Test: `kernel/ops/fillet_obstacle_merge_test.go`

**Interfaces:**
- Consumes: `filletLoop`, `crossing`/`obstacleNodes` (Task 2), `math.Point3`/`Point2`, `planeProjector`.
- Produces: `func mergeHoleIntoNotch(outer, hole filletLoop, nodes [2]crossing, flat func(math.Point3) math.Point2, back func(math.Point2) math.Point3) (filletLoop, bool)` — returns the single notched loop (outer with its receded front edge re-routed along the hole sub-arc that stays ABOVE the boundary, i.e. the ellipse arc between the Nodes), `ok=false` on a malformed input.

- [ ] **Step 1: Write the failing test** — merging the T6 receded outer (rectangle y[-7,12]) with the ellipse hole yields ONE loop whose bbox y-min is −10 (the notch dips to the ellipse) and which is a simple (non-self-intersecting) polygon.

```go
// kernel/ops/fillet_obstacle_merge_test.go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import "testing"

func TestMergeHoleIntoNotchT6(t *testing.T) {
	outer := recededTopOuterT6() // rectangle x[-20,20] y[-7,12], as a filletLoop (test helper)
	hole := ellipseHoleT6()      // full ellipse a=15 b=10 as a filletLoop (test helper)
	nodes, ok := obstacleNodesForT6(t) // crossings at (±10.712,-7)
	if !ok {
		t.Fatal("expected two crossings for T6")
	}
	flat, back := zPlaneProjector() // z=0 plane <-> 2D
	notch, ok := mergeHoleIntoNotch(outer, hole, nodes, flat, back)
	if !ok {
		t.Fatal("merge failed")
	}
	if ymin := loopMinY(notch); ymin > -9.9 {
		t.Errorf("notch should dip to the ellipse (y≈-10), got y-min %.3f", ymin)
	}
	if selfIntersects2D(notch, flat) {
		t.Errorf("notched loop must be a simple polygon")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run 'TestMergeHoleIntoNotchT6' -v`
Expected: FAIL — `undefined: mergeHoleIntoNotch`.

- [ ] **Step 3: Implement the KEML merge** — walk the outer loop; where its edge crosses between the two Nodes along the receded boundary, splice in the hole's sub-arc that stays on the *host* side of the boundary (the ellipse arc between P− and P+), oriented so the merged loop stays consistently wound. Keep it 4–20 lines per function (a `spliceSubArc` helper + orientation).

- [ ] **Step 4: Run to verify pass**

Run: `go test ./kernel/ops/ -run 'TestMergeHoleIntoNotchT6' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_obstacle_merge.go kernel/ops/fillet_obstacle_merge_test.go
git commit -m "feat(fillet): KEML merge of a crossing hole loop into the receded outer loop (notched face)"
```

---

### Task 6: Wire detect → merge → resolveCornerBlend into the fillet rebuild + fold `HolesContained` into `Valid`

**Files:**
- Modify: `kernel/ops/fillet_faces.go` (`filletResultFaces`, and a new `obstacleFacesFor` helper); split the wing cylinder at the band.
- Modify: `kernel/ops/validate.go` (line 63 — fold `HolesContained`)
- Modify: `model/feature/occtparity/fillet_hole_containment_test.go` (flip tripwire)
- Test: append to `kernel/ops/corner_blend_obstacle_test.go` a body-level test that filleting a synthetic slab-with-column produces a watertight, `HolesContained` body.

**Interfaces:**
- Consumes: `detectObstacle` (compose Task 2 pieces into `func detectObstacle(f *topo.Face, ef edgeFillet, subs map[uint64]math.Point3, res Resolution) (*ObstacleFeature, [2]crossing, bool)`), `mergeHoleIntoNotch` (Task 5), `resolveCornerBlend` + `bsplineObstacleProvider` (Tasks 1/4), `cylinderFace`.
- Produces: obstacle-aware branch in `filletResultFaces`.

- [ ] **Step 1: Write the failing test** — after fillet, `ops.Validate(body).HolesContained == true` AND `Valid == true` on the synthetic slab+column (build it in-code with `ops` primitives; do not depend on the STEP corpus here so it stays a fast unit test).

```go
// append to kernel/ops/corner_blend_obstacle_test.go
func TestFilletSlabColumnWatertight(t *testing.T) {
	body := slabWithColumn(t)              // slab z[-20,0] + elliptical column z[0,30] (test helper via ops)
	res, ok, reason := filletFrontTopEdge(body, 6) // drive the real fillet feature at r=6 (test helper)
	if !ok {
		t.Fatalf("fillet failed: %s", reason)
	}
	rep := ops.Validate(res)
	if !rep.HolesContained {
		t.Errorf("filleted slab+column must be hole-contained (no protrusion): %v", rep.Issues)
	}
	if !rep.Valid {
		t.Errorf("filleted slab+column must be a valid solid: %v", rep.Issues)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run 'TestFilletSlabColumnWatertight' -v`
Expected: FAIL — the protruding hole makes `HolesContained` false (and once folded, `Valid` false) until the wiring lands.

- [ ] **Step 3: Fold `HolesContained` into `Valid`**

`kernel/ops/validate.go`, replace line 63:
```go
	r.Valid = r.Manifold && r.OrientationOK && r.HolesContained &&
		(!b.IsSolid() || (r.Closed && r.EulerConsistent))
```
Update the doc comment on `HolesContained` (validate.go:27) and the not-yet-folded note (validate.go:60) to state it is now folded.

- [ ] **Step 4: Wire the obstacle branch in `filletResultFaces`**

In `transformFace`/`filletResultFaces`: for each face `f` with ≥2 loops touched by a fillet whose receded outer loop now has a crossing hole (`detectObstacle` returns ok), replace the default `transformFace` output for `f` with the **notched** face (`mergeHoleIntoNotch`), split the corresponding `cylinderFace` into two wings at the band, populate the `ObstacleFeature` (wing arcs = the two new wing section arcs by value, wall line, rim sub-arc, nodes), call `resolveCornerBlend(req, []CornerBlendProvider{bsplineObstacleProvider{}})`, and append the returned `CornerBlendPatch` as a `filletFace{surface: patch.Surface, loops: patch.Loops, parent: <junction lineage>}`. On `ok=false` from resolve, **honest-reject**: leave the existing path (the case stays FAIL(faulty), no regression, ADR-3) — do not emit a half-built shell.

Keep the change surgical: a new `obstacleFacesFor(body, ef, f, res) ([]filletFace, bool)` returns `(notched+wings+patch, true)` or `(nil, false)`; `filletResultFaces` calls it before the default `transformFace(f, ...)` and `cylinderFace(ef, ...)` and uses its faces when ok. **Junction and non-obstacle faces take the existing path unchanged.**

- [ ] **Step 5: Run to verify pass**

Run: `go test ./kernel/ops/ -run 'TestFilletSlabColumnWatertight' -v`
Expected: PASS.

- [ ] **Step 6: Flip the tripwire test**

`model/feature/occtparity/fillet_hole_containment_test.go`: change the filleted-body assertion (lines 47–51) from expecting `HolesContained == false` to expecting `true`, and update the comment to record that the fix landed:
```go
	rep := ops.Validate(res[0])
	if !rep.HolesContained {
		t.Errorf("filleted T6 body: the mid-span obstacle patch must make the base plane hole-contained; got protrusion: %v", rep.Issues)
	}
	if !rep.Valid {
		t.Errorf("filleted T6 body must now be a valid solid (HolesContained folded into Valid): %v", rep.Issues)
	}
```

- [ ] **Step 7: Run the ops + tripwire suites**

Run: `go test ./kernel/ops/... && go test ./model/feature/occtparity/ -run 'TestFilletProtrudingHoleTripwire' -v`
Expected: PASS (and no other ops test regressed).

- [ ] **Step 8: Commit**

```bash
git add kernel/ops/fillet_faces.go kernel/ops/validate.go model/feature/occtparity/fillet_hole_containment_test.go kernel/ops/corner_blend_obstacle_test.go
git commit -m "feat(fillet): dispatch mid-span obstacle patch in rebuild; fold HolesContained into Valid; flip tripwire"
```

---

### Task 7: Corpus gate — 13 cases migrate to PASS; body-area parity; regression guard

**Files:**
- Create: `model/feature/occtparity/fillet_midspan_obstacle_test.go`
- Test: the 13 cases + a trihedral/planar regression guard.

**Interfaces:**
- Consumes: `RunCase`, `Corpus`, `CorpusFixtureDir`, `Record`, `assertArea` (all occtparity).

- [ ] **Step 1: Write the gate test** — each of the 13 cases runs through `RunCase` (which asserts OCCT body area within 1%) and additionally asserts `HolesContained`.

```go
// model/feature/occtparity/fillet_midspan_obstacle_test.go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
)

var midspanObstacleCases = map[string]bool{
	"S1": true, "S3": true, "S4": true, "S9": true, "T1": true, "T4": true,
	"T6": true, "T7": true, "T9": true, "U1": true, "U3": true, "U4": true, "X3": true,
}

func TestMidSpanObstacleCorpusPasses(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, r := range Corpus() {
		if r.Grid != "simple" || !midspanObstacleCases[r.Case] {
			continue
		}
		r := r
		t.Run(r.Case, func(t *testing.T) {
			body, err := importInput(filepath.Join(dir, r.InputStep))
			if err != nil {
				t.Skipf("import-divergence (not a fillet defect): %v", err)
			}
			sets, ok := scoreLocate(r, body)
			if !ok {
				t.Skipf("%s: picked edge not locatable", r.Case)
			}
			res, filletOK, reason := runFillet(body, sets)
			if !filletOK || len(res) != 1 {
				t.Fatalf("%s: fillet not a solid: %s", r.Case, reason)
			}
			if rep := ops.Validate(res[0]); !rep.HolesContained || !rep.Valid {
				t.Fatalf("%s: not watertight/hole-contained: %v", r.Case, rep.Issues)
			}
			got := ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Area
			assertArea(t, "simple/"+r.Case, got, r.ExpectedArea, r.Deps)
		})
	}
}
```

- [ ] **Step 2: Run and record which cases pass**

Run: `go test ./model/feature/occtparity/ -run 'TestMidSpanObstacleCorpusPasses' -v`
Expected: initially some cases may still FAIL(area) or FAIL(faulty) — record each. T6 should pass body area within 1% (+0.31%). If any case's body area exceeds 1%, that case needs the FairSurface tightening pass (spec §3) — add it in the provider (one `geom.FairSurface` iteration on the fill before certify) ONLY for the failing cases and re-run. Do **not** widen the gate.

- [ ] **Step 3: Add the regression guard** — assert a trihedral/planar case (e.g. a known-passing `simple` case that is NOT in the obstacle set) still passes byte-for-byte, and that the obstacle path was not entered for it (its faces unchanged). Use an existing passing case id from the scoreboard.

```go
func TestTrihedralUnchangedByObstaclePath(t *testing.T) {
	// pick a currently-PASS planar/trihedral case with no mid-span obstacle; assert it still PASSes.
	RunCase(t, caseByID(t, "simple", "A1"), CorpusFixtureDir()) // replace A1 with a real passing id
}
```

- [ ] **Step 4: Run the full occtparity suite**

Run: `go test ./model/feature/occtparity/...`
Expected: PASS — the 13 obstacle cases green, no previously-passing case regressed.

- [ ] **Step 5: Full local suite + lint**

Run: `go test ./kernel/... ./model/... && golangci-lint run`
Expected: PASS, 0 lint issues.

- [ ] **Step 6: Commit**

```bash
git add model/feature/occtparity/fillet_midspan_obstacle_test.go
git commit -m "test(fillet): corpus gate — 13 mid-span obstacle cases pass body-area parity; trihedral regression guard"
```

---

## Live test (before considering the slice done)

Per CLAUDE.md "Live tests": drive the real head via `Oblikovati.AddIns.MCPBridge` — import T6 (or build the slab+column), apply the r=6 fillet through the actual `AddFillet` + `Recompute` path, capture a screenshot via the MCP screenshot API, and confirm the rendered mesh shows a clean rounded front edge notched around the column with **no phantom fill / crack artifacts** across the obstacle band. A green unit suite is necessary but not sufficient — the mesh is what the user sees.

## Self-Review notes

- **Spec coverage:** ADR-4 trigger (watchdog) → Task 6; ObstacleFeature request → Task 1; rails w/ reused wings + G0 rim → Task 3; certify rotating-normal anti-fold → Task 4; KEML notch → Task 5; body-area gate + 13 cases → Task 7; `HolesContained`→`Valid` + tripwire → Task 6. Item-3 (P± singularity) is a non-issue in this formulation (advisor) → no dedicated task, but the certify's `NoFold` guards it.
- **Open verifications the executor must resolve against real code (named, not hand-waved):** the kernel 2D line type + `SignedDistance` (Task 2); the arc→NURBS / plane→NURBS / cylinder→NURBS converters and `makeRailPair` make-compatible name (Tasks 3–4); `geom.Vector3` `Cross/Dot/Length` (Task 4); a real currently-PASS non-obstacle case id for the regression guard (Task 7). Each has a `grep` pointer in its step; use the real name, do not invent.
- **Area risk:** the stock bilinear Coons is expected to overshoot the *patch* (~178 vs 156.364) but clear the *body* gate (+0.31% < 1%). Task 4's patch-level area assertion is therefore advisory; the binding gate is Task 7's body-area `assertArea`. If any case exceeds the body gate, add exactly one `FairSurface` pass in the provider (spec §3) — the safe, monotone tightening direction.
- **Honest-reject preserved (ADR-3):** every failure path (nil wing, tangential touch, >2 crossings, invalid cert) returns `ok=false` and leaves the existing (rejecting) path, so no case regresses from PASS to a broken shell.
