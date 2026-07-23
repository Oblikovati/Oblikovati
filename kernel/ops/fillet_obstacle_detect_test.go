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
	for i := 0; i < n; i++ {
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
