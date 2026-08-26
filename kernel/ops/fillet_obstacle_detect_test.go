// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// sampleEllipse returns n points of the T6 base ellipse (a=15,b=10) in the z=0 host plane.
func sampleEllipse(n int) []m.Point2 {
	pts := make([]m.Point2, n)
	for i := range n {
		t := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = m.P2(15*math.Cos(t), 10*math.Sin(t))
	}
	return pts
}

func TestRimCrossingsT6(t *testing.T) {
	rim := sampleEllipse(720)
	boundary := boundaryLine2{origin: m.P2(0, -7), dir: m.V2(1, 0)} // the receded boundary y=-7
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
	boundary := boundaryLine2{origin: m.P2(0, -10), dir: m.V2(1, 0)} // tangent to the ellipse bottom
	res := ResolutionForSize(50)
	if _, ok := obstacleNodes(rim, boundary, res); ok {
		t.Errorf("a rim tangent to the boundary must be rejected (no dip, no patch)")
	}
}

// diamondDipRim is a closed 4-vertex rim whose single vertex below y=0 sits at index 0,
// so the dip's two boundary crossings (segments [3,0] and [0,1]) bracket the array wrap.
// It exercises dipsPast's modular midpoint arithmetic in the c0.I > c1.I case.
func diamondDipRim() []m.Point2 {
	return []m.Point2{
		m.P2(0, -2), // index 0 — the dip, below the y=0 boundary (fillet side)
		m.P2(4, 2),  // index 1 — host side
		m.P2(0, 4),  // index 2 — host side
		m.P2(-4, 2), // index 3 — host side
	}
}

// TestDipsPast pins the fillet-band-side check for both the natural (c0.I < c1.I) and the
// wrap-around (c0.I > c1.I) orderings. side=+1 encodes "fillet band is the negative-signed
// side" (signedDist convention: host +ve, fillet -ve), so a genuine dip — the mid-arc sample
// on the negative side — must return true, and a rim whose sampled arc bulges to the host
// (positive) side must return false.
func TestDipsPast(t *testing.T) {
	res := ResolutionForSize(50)
	rim := sampleEllipse(720)

	// Non-wrap dip: y=-7 crossings (natural order) bracket the ellipse bottom (0,-10),
	// which is on the fillet side — a genuine dip.
	dip := boundaryLine2{origin: m.P2(0, -7), dir: m.V2(1, 0)}
	cd := rimCrossings(rim, dip, res)
	if len(cd) != 2 || cd[0].I >= cd[1].I {
		t.Fatalf("expected 2 ordered crossings for y=-7, got %d", len(cd))
	}
	if !dipsPast(rim, cd[0], cd[1], dip, 1) {
		t.Errorf("ellipse dipping below y=-7 must report dipsPast=true")
	}

	// Non-wrap bulge-away: y=+7 crossings; the forward arc between them passes over the top
	// (0,+10), on the host side — no dip into the fillet band below.
	top := boundaryLine2{origin: m.P2(0, 7), dir: m.V2(1, 0)}
	ct := rimCrossings(rim, top, res)
	if len(ct) != 2 {
		t.Fatalf("expected 2 crossings for y=+7, got %d", len(ct))
	}
	if dipsPast(rim, ct[0], ct[1], top, 1) {
		t.Errorf("ellipse arc bulging above y=+7 must report dipsPast=false")
	}

	// Wrap-around: the dip vertex is at index 0, bracketed by crossings on segments [3,0] and
	// [0,1]. rimCrossings returns them as [{I:0}, {I:3}]; the dip arc is the WRAP arc 3→0.
	diamond := diamondDipRim()
	b0 := boundaryLine2{origin: m.P2(0, 0), dir: m.V2(1, 0)}
	cw := rimCrossings(diamond, b0, res)
	if len(cw) != 2 || cw[0].I != 0 || cw[1].I != 3 {
		t.Fatalf("expected diamond crossings at I=0 and I=3, got %+v", cw)
	}
	// c0.I=3 > c1.I=0: forward arc 3→0 wraps through the dip vertex (0,-2) → true.
	if !dipsPast(diamond, cw[1], cw[0], b0, 1) {
		t.Errorf("wrap arc through the index-0 dip must report dipsPast=true")
	}
	// c0.I=0 < c1.I=3: forward arc 0→3 passes through the top (0,4), host side → false.
	if dipsPast(diamond, cw[0], cw[1], b0, 1) {
		t.Errorf("forward arc over the host-side top must report dipsPast=false")
	}
}

// obliqueEllipseWrapDipRim returns n samples of an ellipse (semi-axes a,b, tilted by rot off the
// boundary's own frame) whose SHORT dip arc wraps through rim sample 0 — the exact #2007 U3 mechanism.
// Sample 0 (the curve's t=0 point, sampleHoleRim) sits at the tip (a,0) before rotation; the boundary
// used with this rim is a near-tangent line offset just INSIDE that tip along its outward normal, so
// only a short arc around index 0 dips past it while the rest of the ellipse stays host-side.
func obliqueEllipseWrapDipRim(n int, a, b, rot float64) []m.Point2 {
	pts := make([]m.Point2, n)
	for i := range n {
		t := 2 * math.Pi * float64(i) / float64(n)
		x0, y0 := a*math.Cos(t), b*math.Sin(t)
		pts[i] = m.P2(x0*math.Cos(rot)-y0*math.Sin(rot), x0*math.Sin(rot)+y0*math.Cos(rot))
	}
	return pts
}

// TestDipArcOrderRecoversU3WrapDip is the #2007 U3 kernel regression. rimCrossings always hands its two
// crossings back in ascending array-index order, and since sample 0 is an arbitrary curve reference with
// no relation to the boundary, that ascending arc can be EITHER of the two crossing-bounded arcs — for
// an axis-aligned/uniformly-sampled footprint it is (empirically) the true dip, but for this elongated
// (20:6) oblique (25°) ellipse the true dip is the SHORT arc that WRAPS through index 0, and ascending
// order instead selects the long, uniformly host-side bulge. dipsPast on that (wrong) ascending arc
// faithfully reports FALSE — reproducing U3's obstacle-path honest-reject verbatim (this is what
// production called before this fix: bandCrossings had no arc-order step at all). dipArcOrder, now
// called before dipsPast in bandCrossings, picks the SHORTER of the two candidate arcs — here the wrap
// arc — and dipsPast on THAT arc correctly reports TRUE.
func TestDipArcOrderRecoversU3WrapDip(t *testing.T) {
	const n = 64
	a, b, rot := 20.0, 6.0, 25*math.Pi/180
	rim := obliqueEllipseWrapDipRim(n, a, b, rot)

	normal := m.V2(math.Cos(rot), math.Sin(rot))
	tangent := m.V2(-math.Sin(rot), math.Cos(rot))
	tip := m.P2(a*math.Cos(rot), a*math.Sin(rot))
	boundary := boundaryLine2{origin: tip.TranslateBy(normal.Scale(-0.4)), dir: tangent}
	res := ResolutionForSize(50)

	cs := rimCrossings(rim, boundary, res)
	if len(cs) != 2 {
		t.Fatalf("fixture precondition: want 2 crossings bracketing the wrap dip, got %d", len(cs))
	}
	nodes := [2]crossing{cs[0], cs[1]} // ascending array-index order, exactly what obstacleNodes returns
	side := 1.0
	if boundary.signedDist(rim[0]) > 0 {
		side = -1 // keep rim[0] (inside the constructed dip) on the fillet side regardless of winding
	}

	if dipsPast(rim, nodes[0], nodes[1], boundary, side) {
		t.Fatalf("fixture precondition: the ascending arc must be the (wrong) bulge, not the dip")
	}
	c0, c1 := dipArcOrder(nodes, n)
	if !dipsPast(rim, c0, c1, boundary, side) {
		t.Errorf("dipArcOrder + dipsPast must recover the true wrap-through-index-0 dip (#2007 U3)")
	}
}

// TestDipArcOrderUnchangedForRoundFootprint pins do-no-harm: for a uniformly-sampled, non-wrapping dip
// (T6's shape family — the existing TestDipsPast fixtures above), dipArcOrder must return the SAME
// (ascending) order rimCrossings already hands it, so every currently-green obstacle case is untouched.
func TestDipArcOrderUnchangedForRoundFootprint(t *testing.T) {
	rim := sampleEllipse(720)
	boundary := boundaryLine2{origin: m.P2(0, -7), dir: m.V2(1, 0)}
	res := ResolutionForSize(50)
	cs := rimCrossings(rim, boundary, res)
	if len(cs) != 2 {
		t.Fatalf("want 2 crossings, got %d", len(cs))
	}
	nodes := [2]crossing{cs[0], cs[1]}
	c0, c1 := dipArcOrder(nodes, len(rim))
	if c0 != nodes[0] || c1 != nodes[1] {
		t.Errorf("dipArcOrder swapped a round footprint's already-correct ascending arc: got c0=%+v c1=%+v, want unchanged nodes=%+v", c0, c1, nodes)
	}
	if !dipsPast(rim, c0, c1, boundary, 1) {
		t.Errorf("round footprint dip must still report true after dipArcOrder")
	}
}

// TestDipsPastExtremalIgnoresMisleadingMidSample pins dipsPast's OWN robustness, independent of arc
// selection: given a rim whose arithmetic INDEX-midpoint sample happens to sit on the wrong side (a
// local wobble/notch) even though the arc genuinely dips elsewhere, the deepest-excursion (extremal)
// scan finds the true dip and reports TRUE — the old single-sample midpoint test would have reported
// FALSE here. This specific failure mode cannot occur for a genuine ellipse/conic rim (a line meets a
// conic at ≤2 points, so a crossing-bounded arc is always uniformly signed — #2007 U3's actual defect
// was dipArcOrder's arc CHOICE, not this sample choice; see TestDipArcOrderRecoversU3WrapDip). This
// fixture is a hand-built polyline (not a pure conic sample) to exercise the extremal scan in isolation:
// the literal case dipsPast's docstring describes, and the robustness this slice adds against sampling
// noise or a future non-conic obstacle wall.
func TestDipsPastExtremalIgnoresMisleadingMidSample(t *testing.T) {
	// index:  0    1   2   3   4     5   6   7    8
	// y:     +10  -5  -5  -5  +0.5  -5  -5  +10  +10
	rim := []m.Point2{
		m.P2(0, 10), m.P2(1, -5), m.P2(2, -5), m.P2(3, -5), m.P2(4, 0.5),
		m.P2(5, -5), m.P2(6, -5), m.P2(7, 10), m.P2(8, 10),
	}
	boundary := boundaryLine2{origin: m.P2(0, 0), dir: m.V2(1, 0)}
	c0, c1 := crossing{I: 0}, crossing{I: 6}
	side := 1.0

	oldMidIndex := (c0.I + 1 + ((c1.I-c0.I+len(rim))%len(rim))/2) % len(rim)
	if oldMidIndex != 4 {
		t.Fatalf("fixture precondition: want the old index-midpoint to land on the misleading idx 4 notch, got %d", oldMidIndex)
	}
	if side*boundary.signedDist(rim[oldMidIndex]) < 0 {
		t.Fatalf("fixture precondition: the misleading sample must itself read host-side (positive)")
	}

	if !dipsPast(rim, c0, c1, boundary, side) {
		t.Errorf("extremal scan must find the deep dip (idx 1-3, 5-6) despite the misleading midpoint sample at idx 4")
	}
}

// TestAnalyticNodeSolvesOnTheRimCurveNotItsChords is the node-solver regression. rimCrossings works on
// the obstacleRimSamples-chord polyline, so the crossing it returns is the CHORD's zero of the signed
// distance — a full sagitta inside the CURVE's. This fixture is the exact shape of U4's boss-B mouth: a
// radius-12 circle crossed by a band line 10 from its centre, whose crossings are therefore x = ±√44 in
// closed form. The test asserts BOTH directions of the claim: that the raw polyline crossing really is
// ~1e-2 off (so the premise cannot silently evaporate), and that analyticNode lands on √44 at the
// parametric floor.
func TestAnalyticNodeSolvesOnTheRimCurveNotItsChords(t *testing.T) {
	rim, err := geom.NewCircle(m.P3(0, 0, 0), m.V3(0, 0, 1), 12)
	if err != nil {
		t.Fatalf("fixture circle: %v", err)
	}
	flat := func(p m.Point3) m.Point2 { return m.P2(p.X, p.Y) }
	boundary := boundaryLine2{origin: m.P2(0, -10), dir: m.V2(1, 0)} // the band line y = −10
	pts := make([]m.Point2, obstacleRimSamples)
	for i := range pts {
		pts[i] = flat(rim.PointAt(float64(i) / obstacleRimSamples))
	}
	cs := rimCrossings(pts, boundary, ResolutionForSize(50))
	if len(cs) != 2 {
		t.Fatalf("fixture: circle ∩ y=−10 must cross twice, got %d", len(cs))
	}
	want := math.Sqrt(44)
	for _, c := range cs {
		sampled := math.Abs(math.Abs(c.P.X) - want)
		if sampled < 1e-3 {
			t.Fatalf("fixture precondition: the SAMPLED crossing x=%.12f is already within %.3e of ±√44 — the sagitta this solver removes is gone, so the test proves nothing", c.P.X, sampled)
		}
		got := analyticNode(c, rim, obstacleRimSamples, flat, boundary)
		if got.I != c.I {
			t.Errorf("analyticNode moved the polyline index %d -> %d; the dip-range convention downstream is expressed in indices", c.I, got.I)
		}
		if d := math.Abs(math.Abs(got.P.X) - want); d > 1e-12 {
			t.Errorf("analyticNode x=%.15f is %.3e off the closed form ±√44=%.15f (sampled was %.3e off) — the node is not on the rim curve", got.P.X, d, want, sampled)
		}
	}
}

// TestAnalyticNodeKeepsTheChordWhenTheCurveDoesNotStraddle pins the solver's honest-reject: when the rim
// curve does not strictly change sign across the sample's own parameter bracket, the chord's crossing is
// kept unchanged rather than bisected to some other root.
func TestAnalyticNodeKeepsTheChordWhenTheCurveDoesNotStraddle(t *testing.T) {
	rim, err := geom.NewCircle(m.P3(0, 0, 0), m.V3(0, 0, 1), 12)
	if err != nil {
		t.Fatalf("fixture circle: %v", err)
	}
	flat := func(p m.Point3) m.Point2 { return m.P2(p.X, p.Y) }
	// A band line far outside the circle: no bracket of the rim curve straddles it.
	boundary := boundaryLine2{origin: m.P2(0, -50), dir: m.V2(1, 0)}
	c := crossing{I: 3, P: m.P2(1, 2)}
	if got := analyticNode(c, rim, obstacleRimSamples, flat, boundary); got != c {
		t.Errorf("analyticNode invented a node %+v where the curve never crosses; want the chord's %+v unchanged", got, c)
	}
}

// TestOtherHostDetectionOnlyPairsDualHost pins the coupled-station guard's pairing: only a qualifying==2
// (dual-host) edge has an "other" host whose boss can already be setting the ball back at a node.
func TestOtherHostDetectionOnlyPairsDualHost(t *testing.T) {
	one := []obstacleDetection{{hostIsA: true}}
	if got := otherHostDetection(one, 0); got != nil {
		t.Errorf("a single-host edge must have no other detection, got %+v", got)
	}
	two := []obstacleDetection{{hostIsA: true}, {hostIsA: false}}
	if got := otherHostDetection(two, 0); got == nil || got.hostIsA {
		t.Errorf("otherHostDetection(dual, 0) must return host B's detection, got %+v", got)
	}
	if got := otherHostDetection(two, 1); got == nil || !got.hostIsA {
		t.Errorf("otherHostDetection(dual, 1) must return host A's detection, got %+v", got)
	}
	if coupledNodeStation(nil, edgeFillet{}, m.P3(0, 0, 0)) {
		t.Error("with no other host, no node station can be coupled")
	}
}
