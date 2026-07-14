# Generalized Corner-Blend Engine (RailLoop currency) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce the `RailLoop` currency and its fill/recognition providers
(`analyticSphere`, `analyticTorus`, `coons4`, `tri3`) behind the existing
`CornerBlendProvider` seam, per ADR-0051 — a **corpus-neutral foundation** that
unblocks the per-family rail extractors without moving any corpus case yet.

**Architecture:** ADR-0051 (`architecture/decisions/ADR-0051-generalized-corner-blend-engine.md`).
A junction of any valence is one ordered closed `RailLoop` of `Side{Curve, Adjacent, Cont}`.
Extraction (topology→RailLoop) is split from fill (RailLoop→certified patch). A tier
walk dispatches analytic-first, then `coons4` (4-sided), `tri3` (3-sided degenerate-4),
`nFan` (n-sided, later). All fill providers reuse `geom.FillSurface` and the certificate.

**Tech Stack:** Go, `kernel/ops` (GPL-2.0-only), `kernel/geom` (Coons/FillSurface/
MatchSurface/BSpline), `kernel/topo`, `oblikovati.org/math`. Tests: `go test ./kernel/ops/...`
with named fakes; the corpus gate is `go test ./model/feature -run TestOCCTBlendSimple`.

## Global Constraints

- **Corpus-neutral:** this wave wires NOTHING into `computeCorners`/`solveCorner`/the
  obstacle detection dispatch that changes which requests reach `resolveCornerBlend`.
  `TestOCCTBlendSimple` MUST stay **50 PASS, byte-for-byte on every case** (run the
  full per-case before/after diff; any delta is a regression, not progress).
- **New files:** live under `kernel/ops/blend/` per ADR-0051? — NO. Keep them in
  `kernel/ops` (package `ops`) alongside the existing `corner_blend*.go`; a sub-package
  would force exporting `filletLoop`/`edgeFillet`/certificate internals. Name files
  `corner_rail.go`, `corner_provider_sphere.go`, `corner_provider_torus.go`,
  `corner_provider_coons4.go`, `corner_provider_tri3.go`, `corner_resolve.go`.
- **SPDX** `GPL-2.0-only` header on every new `.go` (run `scripts/add-spdx-headers.py`).
- Functions **4–20 lines**; files **< 500 lines**; explicit types (no `any`/untyped);
  early returns, **≤ 2 indent levels**; no code duplication (extract shared helpers).
- Exception/`error` messages include the offending value AND expected shape.
- Named fakes, not inline stubs. Every new function gets a test; TDD (failing test first).
- Tolerances are **model-relative** (ADR-0042): use `Resolution.Weld()` / `seamAngularTol`
  / `scale`-derived thresholds. NO bare `1e-6`.
- The `Side.Adjacent` for a corner side is the fillet **arm surface** (cylinder/cone/
  torus → exact rational BSpline), never the host face (ADR-0051 note; Port Contract 1).
- Assembly/orient/weld layer stays untouched; providers depend only on `geom`+`math`+`topo`.

---

### Task 1: RailLoop currency

**Files:**
- Create: `kernel/ops/corner_rail.go`
- Test: `kernel/ops/corner_rail_test.go`

**Interfaces:**
- Consumes: `geom.Curve3`, `topo.Face`, `topo.Lineage`, `math.Point3`.
- Produces: `Continuity` (`G0`/`G1`), `Side{Curve geom.Curve3; Adjacent geom.Surface; Cont Continuity}`
  (Adjacent is the oriented arm/host SURFACE, not a topo.Face — providers use geom+math only; the
  extractor supplies it material-outward, topo identity lives on `RailLoop.Provenance`),
  `RailLoop{Sides []Side; Provenance topo.Lineage}`, `func (RailLoop) Closed(tol float64) bool`,
  `func (RailLoop) Valence() int`.

- [ ] **Step 1: Write the failing test**

```go
// corner_rail_test.go
package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func lineSide(a, b math.Point3, cont Continuity) Side {
	return Side{Curve: geom.LineSegment{StartPoint: a, EndPoint: b}, Cont: cont}
}

func TestRailLoopClosedAndValence(t *testing.T) {
	p0 := math.P3(0, 0, 0)
	p1 := math.P3(1, 0, 0)
	p2 := math.P3(1, 1, 0)
	loop := RailLoop{Sides: []Side{
		lineSide(p0, p1, G1), lineSide(p1, p2, G1), lineSide(p2, p0, G0),
	}}
	if !loop.Closed(1e-9) {
		t.Fatalf("triangle loop should be Closed within 1e-9")
	}
	if got := loop.Valence(); got != 3 {
		t.Fatalf("Valence = %d, want 3", got)
	}
}

func TestRailLoopOpenIsNotClosed(t *testing.T) {
	p0, p1, p2 := math.Pt3(0, 0, 0), math.Pt3(1, 0, 0), math.Pt3(1, 1, 0)
	// LineSegment.Domain() is [0,1]; PointAt(0)=StartPoint, PointAt(1)=EndPoint.
	open := RailLoop{Sides: []Side{lineSide(p0, p1, G1), lineSide(p1, p2, G1)}} // gap p2->p0
	if open.Closed(1e-9) {
		t.Fatalf("open loop must not report Closed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestRailLoop -v`
Expected: FAIL — `undefined: RailLoop`, `Side`, `Continuity`, `G0`, `G1`.
(Confirm `geom.NewLineSegment3` and `math.Pt3` exist; if the constructor names differ,
use the actual ones — `grep -rn 'func NewLineSegment3\|func Pt3' kernel/geom math` — and
adjust the test helper. Do NOT invent a constructor.)

- [ ] **Step 3: Write minimal implementation**

```go
// corner_rail.go
// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Continuity is the cross-boundary smoothness a fill must honour along one Side: G0 keeps only
// position (a sharp crease — e.g. a runout's boss-footprint rail), G1 also matches the tangent
// plane of the Side's Adjacent surface. It maps 1:1 onto geom.FillSurface's per-side Order.
type Continuity int

const (
	G0 Continuity = 0 // position only (crease)
	G1 Continuity = 1 // tangent-plane continuous to Adjacent
)

// Side is one boundary rail of a corner/miter/runout patch: the EXACT curve the fill interpolates,
// the surface across it the fill must meet at Cont, and that continuity. For a corner patch Adjacent
// is the fillet ARM surface (a cylinder/cone/torus), not the host face — the two arms at a shared
// patch corner already agree on the host tangent plane, which is what makes the G1 ribbons twist-
// compatible (ADR-0051; Port Contract 1). Adjacent may be nil for a pure-G0 side.
type Side struct {
	Curve    geom.Curve3
	Adjacent *topo.Face
	Cont     Continuity
}

// RailLoop is an ordered, closed cycle of Sides bounding one patch — the single request currency for
// every junction valence (ADR-0051 ADR-A). Provenance carries the generating tokens (root vertex +
// arm edge ids) so the topological-naming pass stays representation-invariant (ADR-0043); it is never
// read by a fill provider.
type RailLoop struct {
	Sides      []Side
	Provenance topo.Lineage
}

// Valence is the number of sides — 2 ⇒ miter, 3 ⇒ trihedral corner, ≥4 ⇒ higher-valence; it is the
// tier-walk dispatch key (ADR-0051 ADR-C).
func (l RailLoop) Valence() int { return len(l.Sides) }

// Closed reports whether each side's end meets the next side's start within tol (a ring, no gap).
// tol is caller-supplied and model-relative (ADR-0042) — pass scale.Weld(), never a bare constant.
func (l RailLoop) Closed(tol float64) bool {
	n := len(l.Sides)
	if n < 2 {
		return false
	}
	for i := 0; i < n; i++ {
		end := curveEnd(l.Sides[i].Curve)
		start := curveStart(l.Sides[(i+1)%n].Curve)
		if end.DistanceTo(start) > tol {
			return false
		}
	}
	return true
}
```

Add the two small curve-endpoint helpers (reuse existing ones if present — first
`grep -rn 'func curveStart\|func curveEnd\|\.PointAt(' kernel/ops`; a `geom.Curve3`
exposes its parametric domain, evaluate at the ends):

```go
// curveStart / curveEnd evaluate a rail at its parametric domain ends. Extracted so Closed and the
// extractors share one definition of "a rail's endpoints".
func curveStart(c geom.Curve3) math.Point3 { lo, _ := c.Domain(); return c.PointAt(lo) }
func curveEnd(c geom.Curve3) math.Point3   { lo, hi := c.Domain(); _ = lo; return c.PointAt(hi) }
```

(Confirm `geom.Curve3` has `Domain() (float64,float64)` and `PointAt(float64) math.Point3` —
`grep -rn 'Domain()\|PointAt(' kernel/geom/curve*.go`. Add the `math` import.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestRailLoop -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/corner_rail.go kernel/ops/corner_rail_test.go
git commit -m "feat(blend): RailLoop currency for the generalized corner engine (ADR-0051)"
```

---

### Task 2: analyticSphereProvider (migrate the sphere recognition onto RailLoop)

**Files:**
- Create: `kernel/ops/corner_provider_sphere.go`
- Test: `kernel/ops/corner_provider_sphere_test.go`

**Interfaces:**
- Consumes: `RailLoop`, `Side`, `Continuity` (Task 1); `geom.Sphere`, `geom.Plane`,
  the existing `solve3`/`outwardPlaneNormal` (fillet.go), `Resolution`.
- Produces: `type railProvider interface { Name() CornerBlendKind; Fits(RailLoop) bool;
  Build(RailLoop, Resolution) (CornerBlendPatch, Certificate, bool) }`;
  `analyticSphereProvider` implementing it; `func sphereFromPlanarRails(RailLoop, float64) (geom.Sphere, bool)`.

**Context for the implementer:** the existing `solveBlend` (fillet.go:854) builds the corner
sphere from 3 planar faces and is called *directly* by `solveCorner` — that call site is NOT
touched (corpus-neutral). This task re-expresses the SAME 3×3 tangency solve as a `RailLoop`
provider so a future `extractTrihedral` can feed it.

**Recognition is purely off the RAILS (verified geometry, simpler & more robust than a plane
solve):** each arm end-section arc is the sphere∩arm-cylinder intersection; for equal fillet
radius `r` both surfaces have radius `r`, so every end-section is a **great circle of the corner
sphere** — its `geom.Arc3d.Center` equals the sphere center and its `Radius` equals `r`. Therefore
the predicate is: **all sides' `Curve` are `geom.Arc3d`, with equal `Radius` `r` and a common
`Center` `C` (each within `scale.Weld()`) ⇒ `Sphere{Center: C, Radius: r}`.** No planes, no
`solve3`, no orientation logic — and it does NOT read `Side.Adjacent` at all (Adjacent matters only
for the G1 fills; the sphere IS the surface). This parallels the torus recognition (Task 6). Extract
`railRadius(RailLoop) (float64, bool)` (common arc radius; `nil`/ok=false if any side isn't an arc or
radii disagree) and `railArcsConcentric(RailLoop, tol) (math.Point3, bool)` (common center or not).
Verify by checking every rail's sampled points lie on the sphere within `scale.Weld()` (they do by
construction when center+radius match).

- [ ] **Step 1: Write the failing test** — build a symmetric trihedral corner (three mutually
  perpendicular planes offset to radius `r`, three quarter-arcs as rails), assert `Fits` is true,
  `Build` returns a patch whose `Surface` is a `geom.Sphere` of radius `r` centered at
  `(r,r,r)` from the corner, `cert.Valid(scale)` true, and every rail lies on the sphere within
  `scale.Weld()`. Add a negative test: perturb one rail arc's `Center` off the common center (by
  ≫ `scale.Weld()`) → `Fits` false (recognition depends on concentric equal-radius arcs, not on
  `Adjacent`).

```go
func TestAnalyticSphereFitsTrihedralPlanar(t *testing.T) {
	loop, scale := trihedralPlanarLoop(t, 4.0) // helper: 3 ⟂ planes at r=4, 3 quarter-arcs, corner at origin
	p := analyticSphereProvider{}
	if !p.Fits(loop) {
		t.Fatalf("3 equal-radius planar rails must Fit the sphere provider")
	}
	patch, cert, ok := p.Build(loop, scale)
	if !ok || !cert.Valid(scale) {
		t.Fatalf("sphere Build ok=%v valid=%v", ok, cert.Valid(scale))
	}
	sph, isSphere := patch.Surface.(geom.Sphere)
	if !isSphere || math.Abs(sph.Radius-4.0) > scale.Weld() {
		t.Fatalf("want geom.Sphere r=4, got %T r=%v", patch.Surface, sphereRadius(patch.Surface))
	}
}
```

  (Write `trihedralPlanarLoop` in the test file as a named helper producing the loop + a
  `Resolution` scale via the existing `blendScale()` helper in `corner_blend_test.go`. Because
  `Side.Adjacent` is now `geom.Surface`, the helper sets each side's `Adjacent` to an **oriented
  `geom.Plane` directly** (normal pointing material-outward) — NO `topo.Face` fixture is needed.
  The sphere provider derives the corner-interior side from the loop geometry, not from a supplied
  normal sign, so it never calls `outwardPlaneNormal`/`topo`.)

- [ ] **Step 2: Run test to verify it fails** — `go test ./kernel/ops/ -run TestAnalyticSphere -v` → FAIL (`undefined: analyticSphereProvider`).

- [ ] **Step 3: Write minimal implementation** — `railProvider` interface; `analyticSphereProvider`
  with `Fits` (all Adjacent planar + equal radius via `railRadius`) and `Build` (call
  `sphereFromPlanarRails`, wrap the `geom.Sphere` as `CornerBlendPatch{Surface: sph, Loops:
  railLoopToFilletLoops(loop), Kind: BlendKindSphere}`, certify with `certifySphereOnRails`).
  Add `BlendKindSphere CornerBlendKind = "analytic-sphere"` to `corner_blend.go`. Keep each
  function 4–20 lines; `sphereFromPlanarRails` reuses `solve3`+`outwardPlaneNormal`. The
  certificate: `Closed` from `loop.Closed(scale.Weld())`, `WeldsArms` true (every rail on the
  sphere), `NoFold` true (a sphere never folds), `MaxDev = max_i dist(rail_i, sphere)`,
  `MaxAngleDev` = max angle between the sphere normal and each planar side's required tangent
  (0 for a true tangent sphere). Provide `railLoopToFilletLoops` (sample the ring like the
  obstacle provider's `obstaclePatchLoops`/`boundaryRing`, reused — extract the shared ring
  sampler into `corner_rail.go` so both call it, no duplication).

- [ ] **Step 4: Run test to verify it passes** — `go test ./kernel/ops/ -run TestAnalyticSphere -v` → PASS.

- [ ] **Step 5: Commit** — `feat(blend): analyticSphereProvider recognizes equal-radius trihedral corners on RailLoop`.

---

### Task 3: coons4Provider (generalize the obstacle FillSurface path to any 4-sided RailLoop)

**Files:**
- Create: `kernel/ops/corner_provider_coons4.go`
- Test: `kernel/ops/corner_provider_coons4_test.go`

**Interfaces (NO file extraction — everything below is package `ops` already; coons4 just CALLS it):**
- Reuse directly (do NOT move): `asBSplineCurve(geom.Curve3) (geom.BSplineCurve, bool)`
  (corner_blend_obstacle_rails.go — Curve3→BSpline, LineSegment exact / else `RebuildCurve`),
  `makeRailPair`/`pinnedRail`/`pinEnds`/`reverseBSplineCurve` (rail compat + corner pinning),
  `extrudeRibbon`/`orientInward`/`inwardCrossU`/`inwardCrossV`/`pinFillBoundary`/`refineForG1`
  (corner_blend_obstacle.go), and the certify internals `obstacleNoFold`/`columnFolds`/
  `normalsReverse`/`railDev`/`seamCrease`/`fillEdge`/`fillParam`/`creaseAngle`/`vectorAngle`
  (corner_blend_obstacle_certify.go — all already generic in `(surf, rail/nbr, edge, scale)`).
- Produces: `coons4Provider` (implements `railProvider`); `coons4Fill(loop RailLoop, scale Resolution)
  (geom.BSplineSurface, [4]ribbonSide, Certificate, bool)`; `loopRails(loop RailLoop) (c0,c1,d0,d1
  geom.BSplineCurve, ok bool)` (4 sides→4 pinned compatible rails, corners = the loop's shared
  endpoints); `adjacentRibbon(rail geom.BSplineCurve, adj geom.Surface, base geom.BSplineSurface,
  atMaxU, atMaxV bool, isU bool, length float64) (geom.BSplineSurface, bool)` (the ONE new geometry).
- **Also fold in the Task-2 carried Important finding:** make `sampleRailOpen`
  (corner_blend_obstacle.go) delegate to `sampleCurve3Open` (corner_provider_sphere.go) — a
  `geom.BSplineCurve` satisfies `geom.Curve3`, so this removes the duplication the Task-2 reviewer
  flagged. Re-run the obstacle tests to prove the delegation is behavior-preserving.

**Context:** the general 4-sided fill. `loopRails` builds the four rails via `asBSplineCurve` + the
obstacle's `makeRailPair`/`pinnedRail`/`pinEnds` machinery, taking the four corner points from the
loop's consecutive shared endpoints (`curveStart`/`curveEnd`). Map to the Coons convention (side
order `c0=v0, c1=v1, d0=u0, d1=u1` — write the RailLoop-index→Coons-edge mapping in a comment and
pin it with an assertion). The **new** piece is `adjacentRibbon`: for a `Cont==G1` side, build a
degree-(p,1) ruled ribbon whose VMin edge IS the rail and whose second row is each rail control
point pushed along the Adjacent surface's inward tangent — generalizing `extrudeRibbon`'s single
`dir` to a PER-CONTROL-POINT direction `n_i × t_i` where `n_i = adj.NormalAt(adj.ParamAt(c_i))` and
`t_i` = the rail tangent (reuse `orientInward` against the plain-Coons interior derivative so the
sign is robust, exactly as the obstacle does). For a planar/cylindrical Adjacent this reduces to
`extrudeRibbon`; for a cone/torus it tracks the true normal. Then `FillSurface` with
`FillSide{Adjacent: ribbon_i, AdjEdge: VMinEdge, Order: int(Sides[i].Cont)}` (G0 sides get
`Order:0`, no ribbon) → `pinFillBoundary` → certify. `certifyCoons4Patch(surf, rails, ribbons,
orders, scale)` is a thin (~12-line) function calling the generic internals: `NoFold: obstacleNoFold(surf,
scale)`; `MaxDev:` max `railDev` over the 4 rails; `MaxAngleDev:` max `seamCrease` over the G1 sides
only (skip `Order==0` sides — the obstacle rim excludes itself the same way); `Closed/WeldsArms` from
`loop.Closed(scale.Weld())` + all four sides spanned.

- [ ] **Step 1: Write the failing test** — construct a synthetic 4-sided loop that is NOT
  analytically special: two opposite sides are equal-radius circular arcs (on two perpendicular
  cylinders), two are straight rails on planes, all `Cont=G1`, `Adjacent` set to the respective
  `geom.Cylinder`/`geom.Plane`. Assert `coons4Provider.Fits` (Valence 4) true, `Build` returns
  `ok && cert.Valid(scale)`, the fill interpolates all four rails within `scale.Weld()` (sample
  endpoints + midpoints), `cert.NoFold` true, and each G1 side's crease (`seamCrease`) is below
  `seamAngularTol`. Add a degenerate test: a loop whose two adjacent rails are near-colinear (sliver)
  → `cert.Valid` false (honest-reject), NOT a panic. (Confirm `geom.Cylinder` exists + its ctor via
  `grep -rn 'type Cylinder' kernel/geom`; if the synthetic cylinder ribbon is awkward, a plane
  Adjacent on all four sides is an acceptable first fixture — the ribbon path is exercised either way.)

- [ ] **Step 2: Run test to verify it fails** — FAIL (`undefined: coons4Provider`).

- [ ] **Step 3: Write minimal implementation** — `coons4Provider` + `loopRails` + `adjacentRibbon` +
  `coons4Fill` + `certifyCoons4Patch`, all calling the reused helpers above; do the `sampleRailOpen`
  delegation. Keep functions 4–20 lines. `Build` orchestrates `loopRails → refineForG1 → adjacentRibbon
  ×(G1 sides) → FillSurface → pinFillBoundary → certifyCoons4Patch`.

- [ ] **Step 4: Run test to verify it passes** — `go test ./kernel/ops/ -run 'TestCoons4|Obstacle|TestFillet' -v`
  → PASS (new coons4 tests AND every existing obstacle/fillet test — proving the `sampleRailOpen`
  delegation and any shared-helper touch stayed behavior-preserving).

- [ ] **Step 5: Commit** — `feat(blend): coons4Provider — general 4-sided ribbon-G1 fill over a RailLoop`.

---

### Task 4: ~~Re-express bsplineObstacleProvider on the shared coons4 core~~ — SUPERSEDED (dropped)

**Status: dropped 2026-07-14 during execution.** The rationale for this task was "one 4-sided fill
path, no drift." But everything is package `ops`: `coons4` already CALLS the very same helpers the
obstacle provider uses (`asBSplineCurve`, `extrudeRibbon`, `pinFillBoundary`, `refineForG1`, and the
`obstacleNoFold`/`railDev`/`seamCrease` certify internals) — there is no logic duplication to remove.
Rerouting the *green* obstacle `Build` through `coons4Fill` would force `coons4Fill` to accept the
obstacle's direction-supplied ribbons (a ribbon-override parameter), re-coupling the two paths and
putting the byte-for-byte corpus gate at risk for **zero** real de-duplication. So the obstacle
provider is left **untouched** (byte-for-byte trivially preserved), and "migrate provider_obstacle"
is satisfied by shared-helper reuse, not a Build reroute. The one genuine duplication the Task-2
reviewer found (`sampleRailOpen`/`sampleCurve3Open`) is de-duped inside **Task 3** instead.

---

### Task 5: tri3Provider (3-sided degenerate-4 fill, Port Contract 1)

**Files:**
- Create: `kernel/ops/corner_provider_tri3.go`
- Test: `kernel/ops/corner_provider_tri3_test.go`

**Interfaces:**
- Consumes: `RailLoop` (Valence 3), `coons4Fill`/`certifyPatch`/ribbon helpers, `geom.FillSurface`.
- Produces: `tri3Provider` (implements `railProvider`); `func degenerate4(loop RailLoop) (c0,c1,d0,d1 geom.BSplineCurve, poleOrder [4]int, ok bool)`
  (collapse the pole corner: `c0`=rail opposite the pole, `d0`/`d1`=the two rails meeting at the
  pole, `c1`=the pole point as a degenerate curve; `poleOrder` = per-side Cont with the pole side 0);
  `func choosePole(RailLoop) int` (the corner whose two incident arms have the most consistent
  tangent — max dot of the two Adjacent normals at the corner).

**Context (Port Contract 1, ADR-0051 tier 4):** a trihedral corner's 3 corners are G1-compatible
(the two arms at each corner share the host tangent plane), so a polynomial degenerate-4 Coons
satisfies all 3 G1 ribbons — no Gregory. Collapse ONE corner (the one `choosePole` picks) to a
pole; the pole's tangent cone is geometrically correct (a real tangent-point corner of the blend).
**Anti-fold at the pole:** sample the fold sign on INTERIOR stations only (v ∈ [0, 1−δ], δ ≈ one
grid step — never the collapsed v=1 row where |S_u×S_v|→0 by construction), using the rotating
reference normal already in `certifyPatch` (`sign(n_prev · (S_u×S_v))`, skip near-zero-magnitude
stations). `certifyPatch` needs a `skipPole bool` (or an excluded-v-window) parameter so the tri3
path excludes the pole row; add it and default it off for coons4.

- [ ] **Step 1: Write the failing test** — a synthetic 3-sided loop that is NOT a sphere: three
  circular arcs of the SAME radius but on three cylinders whose axes do NOT admit a common tangent
  sphere (so `analyticSphereProvider.Fits` is false), all `Cont=G1`. Assert `tri3Provider.Fits`
  (Valence 3) true, `Build` returns `ok && cert.Valid(scale)`, the fill interpolates all three arcs
  within `scale.Weld()`, `cert.MaxAngleDev < seamAngularTol`, and `cert.NoFold` true (proving the
  interior-station anti-fold survives the pole). Add a test that a deliberately twisted loop (one
  rail flipped) yields `cert.Valid` false (honest-reject, no panic at the pole).

- [ ] **Step 2: Run test to verify it fails** — FAIL (`undefined: tri3Provider`).

- [ ] **Step 3: Write minimal implementation** — `choosePole` → `degenerate4` → `coons4Fill` with
  the collapsed `c1` and `poleOrder` → `certifyPatch(..., skipPole=true)`. Keep functions 4–20
  lines. The degenerate `c1` is a BSpline all of whose control points equal the pole point (a
  valid collapsed edge); verify `CoonsFill` accepts it (it should — the corner-compatibility
  conditions hold), else honest-reject.

- [ ] **Step 4: Run test to verify it passes** — `go test ./kernel/ops/ -run TestTri3 -v` → PASS.

- [ ] **Step 5: Commit** — `feat(blend): tri3Provider — 3-sided degenerate-4 fill with pole anti-fold (Port Contract 1)`.

---

### Task 6: ~~analyticTorusProvider~~ — DEFERRED (analytic promotion, not this wave)

**Status: deferred 2026-07-14 during execution** to the extractor-wiring phase, as an oracle-grounded
analytic promotion (ADR-0051 ADR-2 — a family is promoted to exact by inserting a provider earlier,
later, with zero caller change). Two reasons this cannot be built RIGHT now:
1. A torus **patch** is bounded by **mixed-radius** arcs — the minor circles (radius `r` = MinorRadius)
   AND the parallel circles (radius `R + r·cos v`). The "all rails radius `r`" sketch above is simply
   wrong, and the true rail geometry of a torus corner blend is only known from real corpus cases (the
   DRAWEXE oracle), which don't exist until an extractor produces them.
2. A robust fallback (fit a torus to the rails) is a nonlinear 5–6-DOF least-squares with no existing
   in-kernel helper — disproportionate, and not needed here.
`coons4`/`tri3` already FILL torus-bounded loops correctly (certified), so this is a promote-to-exact
optimization, NOT a coverage gap. Building it on a guessed predicate would be the cheap shortcut CLAUDE.md
forbids. The foundation tier walk (Task 7) is therefore `[analyticSphere, coons4, tri3]` + honest-reject;
`analyticTorus` slots in ahead of `coons4` when the extractor phase grounds its recognition in oracle data.

---

### Task 7: resolveBlend tier assembly + honest-reject (corpus-neutral)

**Files:**
- Create: `kernel/ops/corner_resolve.go`
- Test: `kernel/ops/corner_resolve_test.go`

**Interfaces:**
- Consumes: the three foundation providers (`analyticSphereProvider`, `coons4Provider`, `tri3Provider`)
  + `railProvider`.
- Produces: `func blendTiers() []railProvider` (the foundation order: **sphere → coons4 → tri3**;
  analyticTorus and nFan are DEFERRED promotions, inserted later per ADR-2); `func resolveBlend(loop
  RailLoop, scale Resolution) (CornerBlendPatch, bool)` (walks tiers, returns the first `Fits`+`cert.Valid`
  patch, else `false` = honest-reject).

**Context:** this is the generalized analogue of `resolveCornerBlend` for the RailLoop path. It is
NOT yet called from `computeCorners`/`solveCorner`/obstacle detection — no extractor exists yet, so
no corpus request reaches it (corpus-neutral; the gate below proves it). The tier-ordering tests
(ADR-2) pin that an analytic provider, when it Fits, wins over `coons4`/`tri3`.

- [ ] **Step 1: Write the failing test** — (a) a trihedral spherical loop (Task-2 `sphereTriLoop`)
  resolves to `BlendKindSphere` (sphere wins over coons4/tri3 by order); (b) a generic 4-sided loop
  (Task-3 `quarterCylLoop`) resolves to `BlendKindCoons4`; (c) a generic 3-sided loop whose rails are
  NOT concentric (so analyticSphere declines) resolves to `BlendKindTri3`; (d) a degenerate loop no
  provider certifies → `ok=false` (honest-reject). Use a `fakeRailProvider` named type for an
  ordering-isolation test (earlier Fit+Valid wins over a later one).

- [ ] **Step 2: Run** → FAIL (`undefined: resolveBlend`).

- [ ] **Step 3: Implement** `blendTiers`+`resolveBlend` (mirror `resolveCornerBlend`'s 4–20-line
  walk; `cert.Valid(scale)` gate).

- [ ] **Step 4: Verify corpus-neutral** — `go test ./kernel/ops/... -v` PASS AND
  `go test ./model/feature -run TestOCCTBlendSimple` → **50 PASS, full per-case diff ZERO change**
  vs the branch baseline (this is the release gate for the whole wave).

- [ ] **Step 5: Commit** — `feat(blend): resolveBlend tier walk (sphere→coons4→tri3, honest-reject)`.

---

## Verification

**Per task:** `go test ./kernel/ops/ -run <TaskPattern> -v` green; every new function has a test
with named fakes; functions 4–20 lines; files < 500; `golangci-lint run ./kernel/...` clean;
SPDX headers present (`scripts/add-spdx-headers.py --check`).

**Whole-wave (the corpus-neutral gate, MUST hold after Task 3 and Task 7):**
`go test ./model/feature -run TestOCCTBlendSimple` → 50 PASS, and the per-case scoreboard diff
against the pre-branch-tip baseline shows **byte-for-byte identical** results on every case. Any
change is a regression — this wave adds capability WITHOUT touching dispatch.

**Not in this plan (next, oracle-gated slice):** `extractTrihedral`/`extractRunout`/`extractMiter`
(topology→RailLoop) and wiring `resolveBlend` into `computeCorners`/`solveCorner` behind the
curved-host guard — the increments that actually move S1/S4/T1/T7/T9 and the ~30 trihedral +
~20 miter cases faulty→PASS, each pinned by its DRAWEXE oracle (`test-utilities/occt-blend/oracle`).

## Execution order

Task 1 → Task 2 / Task 3 (parallel-safe; both depend only on Task 1, but Task 3's helper-extraction
touches `corner_blend_obstacle.go`, so run Task 3 before Task 4) → Task 4 → Task 5 → Task 6 → Task 7.
Commit per task to `feat/occt-blend-parity-corpus`. NO PR (corpus not yet fully green).
