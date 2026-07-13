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
