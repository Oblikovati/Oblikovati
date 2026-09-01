// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/math"
)

// distToAxis returns the perpendicular distance from p to the line through c with unit direction u
// — the cylinder-membership check (|dist - r| must be ~0 for a point on the fillet tube).
func distToAxis(p, c math.Point3, u math.Vector3) float64 {
	w := c.VectorTo(p)
	perp := w.Sub(u.Scale(w.Dot(u)))
	return float64(perp.Length())
}

// threeFaceFan is the canonical valid fan the arc/certificate tests share: axis +x through the
// origin, r=2, three planar far faces fanning ta(0,2,0) -> tb(0,-2,0) across two far edges.
func threeFaceFan() endCornerFan {
	return endCornerFan{
		radius: 2, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0),
		ta: math.P3(0, 2, 0), tb: math.P3(0, -2, 0),
		fan: []fanFace{
			{face: 101, normal: math.V3(0, 1, 0), entryEdge: 0, exitEdge: 201},
			{face: 102, normal: math.V3(0, 0, 1), entryEdge: 201, exitEdge: 202},
			{face: 103, normal: math.V3(0, -1, 0), entryEdge: 202, exitEdge: 0},
		},
		farEdges: []fanEdge{
			{edge: 201, from: math.P3(0, 0, 0), to: math.P3(0, 7, 7), leftFace: 101, rightFace: 102},
			{edge: 202, from: math.P3(0, 0, 0), to: math.P3(0, -7, 7), leftFace: 102, rightFace: 103},
		},
	}
}

// A synthetic fan: axis along +x through the origin, radius 2. A far edge from the apex (0,0,0)
// straight along +y crosses the cylinder (distance-2 tube about the x-axis) at y=2.
func TestSplitOnFarEdgeAnalytic(t *testing.T) {
	t.Parallel()
	fan := endCornerFan{
		radius: 2,
		center: math.P3(0, 0, 0),
		axis:   math.V3(1, 0, 0),
		apex:   math.P3(0, 0, 0),
	}
	fe := fanEdge{from: math.P3(0, 0, 0), to: math.P3(0, 10, 0)}
	p, ok := splitOnFarEdge(fan, fe)
	if !ok {
		t.Fatal("expected a crossing")
	}
	if d := p.DistanceTo(math.P3(0, 2, 0)); d > 1e-9 {
		t.Errorf("split at %v, want (0,2,0) (dist %.3g)", p, d)
	}
}

// An edge oblique to the axis (both x and y components) still crosses the tube at exactly one
// point in (0,1); the expected crossing (-1.8,2,0) was hand-derived from the quadratic
// (A=100, B=0, C=-4 -> t=0.2) and independently checked against distance-to-axis-line == r.
func TestSplitOnFarEdgeAnalyticOblique(t *testing.T) {
	t.Parallel()
	fan := endCornerFan{
		radius: 2,
		center: math.P3(0, 0, 0),
		axis:   math.V3(1, 0, 0),
	}
	fe := fanEdge{from: math.P3(-3, 0, 0), to: math.P3(3, 10, 0)}
	p, ok := splitOnFarEdge(fan, fe)
	if !ok {
		t.Fatal("expected a crossing")
	}
	if d := p.DistanceTo(math.P3(-1.8, 2, 0)); d > 1e-9 {
		t.Errorf("split at %v, want (-1.8,2,0) (dist %.3g)", p, d)
	}
}

// A far edge that never comes within the fillet radius of the axis (at least distance 5 from the
// axis, always outside the r=2 tube) must report no crossing.
func TestSplitOnFarEdgeAnalyticMiss(t *testing.T) {
	t.Parallel()
	fan := endCornerFan{
		radius: 2,
		center: math.P3(0, 0, 0),
		axis:   math.V3(1, 0, 0),
	}
	fe := fanEdge{from: math.P3(5, 5, 0), to: math.P3(5, 10, 0)}
	if _, ok := splitOnFarEdge(fan, fe); ok {
		t.Fatal("expected no crossing for an edge that never nears the tube")
	}
}

// Far edge exactly parallel to the +x axis at radial distance 3 from it; with r=2 the tube
// is never reached, and the quadratic degenerates (a=0,b=0) -> the axis-parallel fallback
// in smallestRootIn01 must return ok=false rather than divide by zero.
func TestSplitOnFarEdgeParallelMiss(t *testing.T) {
	t.Parallel()
	fan := endCornerFan{radius: 2, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0)}
	fe := fanEdge{from: math.P3(0, 3, 0), to: math.P3(10, 3, 0)}
	if _, ok := splitOnFarEdge(fan, fe); ok {
		t.Error("parallel non-crossing edge must not report a split")
	}
}

// TestSplitOnFarEdgeSelectsNearApexV5 is the regression gate for the valence-6 area over-shoot: on
// V5's real runout fan, each interior far edge crosses the fillet cylinder TWICE — a near-apex
// crossing (~1 cm from the apex, inside the fillet's minor-arc band, OCCT's) and a far one deep in
// the body (~9-14 cm from the apex, outside the band). splitOnFarEdge MUST return the near one. The
// old "smallest root in (0,1)" rule shipped the far crossing (edge from (0,0,100): far root
// distToApex ~11.9), which forced a spurious cap and drifted V5's area +3.24% (see the
// geometry-math-advisor derivation and .superpowers/sdd/v5-stepA-report.md). We pin the edge whose
// outer endpoint is (0,0,100) — the advisor's edge22 — and assert both the point (~(41.87,89.79,
// 50.46), matching the advisor's independent (41.870,89.793,50.462)) and that its apex distance is
// ~1.03, NOT ~11.9. Every far edge on this fan must land its split within one radius of the apex.
func TestSplitOnFarEdgeSelectsNearApexV5(t *testing.T) {
	t.Parallel()
	b := importCorpusSolid(t, "simple/V5")
	fils := solvedFilsForCase(t, b, "simple/V5")
	fans, _ := classifyEndCorners(fils)
	if len(fans) != 1 {
		t.Fatalf("V5: got %d fans, want 1 (the valence-6 end)", len(fans))
	}
	fan := fans[0]
	for _, fe := range fan.farEdges {
		p, ok := splitOnFarEdge(fan, fe)
		if !ok {
			t.Fatalf("far edge %d reported no crossing on V5's valid fan", fe.edge)
		}
		if d := float64(p.DistanceTo(fan.apex)); d >= fan.radius {
			t.Errorf("far edge %d split %v is %.3f from apex (>= r=%.3f) — the far crossing, not the near-apex one",
				fe.edge, p, d, fan.radius)
		}
	}
	assertEdge22NearApex(t, fan)
}

// assertEdge22NearApex pins the advisor's worked example: the far edge running out to (0,0,100)
// must split at the near-apex crossing (41.87,89.79,50.46) ~1.03 from the apex, not the far one at
// ~11.9. Locating the edge by its outer endpoint (not a topology-dependent id) keeps the gate stable
// across re-imports.
func assertEdge22NearApex(t *testing.T, fan endCornerFan) {
	t.Helper()
	fe, ok := farEdgeFromOuter(fan, math.P3(0, 0, 100))
	if !ok {
		t.Fatal("V5 fan has no far edge running out to (0,0,100)")
	}
	p, _ := splitOnFarEdge(fan, fe)
	want := math.P3(41.8689325645522, 89.7882156390301, 50.464832740425756)
	if d := float64(p.DistanceTo(want)); d > 1e-2 {
		t.Errorf("edge22 split %v != near-apex %v (dist %.4g)", p, want, d)
	}
	if d := float64(p.DistanceTo(fan.apex)); d < 0.5 || d > 2 {
		t.Errorf("edge22 split apex-distance %.4f, want ~1.03 (near root), not ~11.9 (far root)", d)
	}
}

// farEdgeFromOuter returns the fan far edge whose non-apex endpoint is near p (fixtures are exact;
// tol is generous), disambiguating one interior edge from another by geometry rather than id.
func farEdgeFromOuter(fan endCornerFan, p math.Point3) (fanEdge, bool) {
	for _, fe := range fan.farEdges {
		if fe.from.DistanceTo(p) < 1e-6 || fe.to.DistanceTo(p) < 1e-6 {
			return fe, true
		}
	}
	return fanEdge{}, false
}

// TestSolveRunoutSpreadChainCloses is the weld-twice invariant gate: a hand-built 3-far-face fan
// (axis +x, r=2, apex origin) must assemble into a closed tA -> split(201) -> split(202) -> tB
// chain, with every interior far edge split at exactly the point its two bordering faces share.
func TestSolveRunoutSpreadChainCloses(t *testing.T) {
	t.Parallel()
	fan := endCornerFan{
		radius: 2, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0),
		ta: math.P3(0, 2, 0), tb: math.P3(0, -2, 0),
		fan: []fanFace{
			{face: 101, normal: math.V3(0, 1, 0), entryEdge: 0, exitEdge: 201},
			{face: 102, normal: math.V3(0, 0, 1), entryEdge: 201, exitEdge: 202},
			{face: 103, normal: math.V3(0, -1, 0), entryEdge: 202, exitEdge: 0},
		},
		farEdges: []fanEdge{
			{edge: 201, from: math.P3(0, 0, 0), to: math.P3(0, 7, 7), leftFace: 101, rightFace: 102},
			{edge: 202, from: math.P3(0, 0, 0), to: math.P3(0, -7, 7), leftFace: 102, rightFace: 103},
		},
	}
	sp, err := solveRunoutSpread(fan)
	if err != nil {
		t.Fatalf("solveRunoutSpread: %v", err)
	}
	if len(sp.splits) != 2 {
		t.Fatalf("splits = %d, want 2", len(sp.splits))
	}
	// The A-flank piece starts at ta; the B-flank piece ends at tb; consecutive pieces meet at splits.
	a := sp.pieces[101]
	b := sp.pieces[102]
	c := sp.pieces[103]
	if a.tIn.DistanceTo(fan.ta) > 1e-9 {
		t.Errorf("A-flank piece must start at ta, got %v", a.tIn)
	}
	if c.tOut.DistanceTo(fan.tb) > 1e-9 {
		t.Errorf("B-flank piece must end at tb, got %v", c.tOut)
	}
	if a.tOut.DistanceTo(sp.splits[201]) > 1e-9 {
		t.Errorf("face101.tOut must equal split 201")
	}
	// Full weld chain: a broken split-sharing (e.g. each face computing its own boundary point
	// instead of reading the shared sp.splits entry) would fail one of these even though the
	// first two assertions above pass.
	if b.tIn.DistanceTo(sp.splits[201]) > 1e-9 {
		t.Errorf("face102.tIn must equal split 201")
	}
	if b.tOut.DistanceTo(sp.splits[202]) > 1e-9 {
		t.Errorf("face102.tOut must equal split 202")
	}
	if c.tIn.DistanceTo(sp.splits[202]) > 1e-9 {
		t.Errorf("face103.tIn must equal split 202")
	}
}

// TestRunoutPieceIsArcOnCylinder is the arc-fit gate: every far-face piece carries a non-nil arc
// that passes through its tIn/tOut EXACTLY (that is what welds the pieces), the ellipse mid point
// the arc bulges through actually lies on the fillet cylinder (|dist-to-axis - r| ~ 0), AND that mid
// is angularly STRICTLY BETWEEN tIn and tOut about the fillet axis. The on-cylinder check alone is
// tautological (any ellipseMidPoint output is at radius r by construction); the in-span check is the
// one that actually distinguishes a correct bisector mid from a wrong-but-still-on-cylinder one — the
// old radial=ff.normal×axis derivation put face 101's mid at 90° (tIn=0°, tOut=45°), OUTSIDE the
// span, while passing the on-cylinder check.
func TestRunoutPieceIsArcOnCylinder(t *testing.T) {
	t.Parallel()
	fan := threeFaceFan()
	sp, err := solveRunoutSpread(fan)
	if err != nil {
		t.Fatalf("solve: %v", err)
	}
	uhat := probe.Unit(fan.axis)
	ref := probe.Unit(fan.center.VectorTo(fan.ta))
	for id, pc := range sp.pieces {
		if pc.curve == nil {
			t.Fatalf("face %d piece has no arc curve", id)
		}
		if d := pc.curve.PointAt(0).DistanceTo(pc.tIn); d > 1e-9 {
			t.Errorf("face %d arc start %v != tIn %v (dist %.3g)", id, pc.curve.PointAt(0), pc.tIn, d)
		}
		if d := pc.curve.PointAt(1).DistanceTo(pc.tOut); d > 1e-9 {
			t.Errorf("face %d arc end %v != tOut %v (dist %.3g)", id, pc.curve.PointAt(1), pc.tOut, d)
		}
		mid, ok := ellipseMidPoint(fan, pc.tIn, pc.tOut)
		if !ok {
			t.Fatalf("face %d ellipseMidPoint reported degenerate on a valid fan", id)
		}
		if d := stdmath.Abs(distToAxis(mid, fan.center, uhat) - fan.radius); d > 1e-9 {
			t.Errorf("face %d mid %v is off the cylinder (|dist-r| = %.3g)", id, mid, d)
		}
		assertMidBisectsSpan(t, fan, uhat, ref, id, pc.tIn, pc.tOut, mid)
	}
}

// assertMidBisectsSpan asserts mid sits angularly strictly between tIn and tOut about the fillet
// axis, on the minor-arc side (span < pi) — the property that distinguishes a correct bisector mid
// from an out-of-span one, which the mere on-cylinder check cannot catch.
func assertMidBisectsSpan(t *testing.T, fan endCornerFan, uhat, ref math.Vector3, id uint64, tIn, tOut, mid math.Point3) {
	t.Helper()
	angleIn := angleAbout(uhat, ref, fan.center, tIn)
	angleOut := probe.WrapPi(angleAbout(uhat, ref, fan.center, tOut) - angleIn)
	angleMid := probe.WrapPi(angleAbout(uhat, ref, fan.center, mid) - angleIn)
	if stdmath.Abs(angleOut) >= stdmath.Pi {
		t.Fatalf("face %d span %.3g rad is not the minor arc (>= pi)", id, angleOut)
	}
	if angleOut >= 0 {
		if !(angleMid > 1e-9 && angleMid < angleOut-1e-9) {
			t.Errorf("face %d mid angle %.3g not strictly between 0 and span %.3g", id, angleMid, angleOut)
		}
		return
	}
	if !(angleMid < -1e-9 && angleMid > angleOut+1e-9) {
		t.Errorf("face %d mid angle %.3g not strictly between span %.3g and 0", id, angleMid, angleOut)
	}
}

// TestMonotoneRejectsScrambled proves the non-self-intersection certificate actually fires: the
// valid fan (splits advance in angular order ta -> 201 -> 202 -> tb) solves cleanly, while swapping
// the two far edges' geometry folds the angular sequence back on itself and must be rejected — not
// silently welded into a self-intersecting cap.
func TestMonotoneRejectsScrambled(t *testing.T) {
	t.Parallel()
	if _, err := solveRunoutSpread(threeFaceFan()); err != nil {
		t.Fatalf("valid 3-face fan must solve, got: %v", err)
	}
	scrambled := threeFaceFan()
	// Swap the two far edges' target geometry so 201 now crosses toward -y+z and 202 toward +y+z:
	// the split points swap angular slots and the ta->splits->tb chain folds back on itself.
	scrambled.farEdges[0].to = math.P3(0, -7, 7)
	scrambled.farEdges[1].to = math.P3(0, 7, 7)
	if _, err := solveRunoutSpread(scrambled); err == nil {
		t.Fatal("scrambled (non-monotone) fan must be rejected by the angular-order certificate")
	}
}
