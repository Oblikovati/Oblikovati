// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// perpDistToAxis is the perpendicular distance from p to the line through a point on the axis with unit
// direction u — the surface test for a cylinder (== r on-surface).
func perpDistToAxis(p, axisPt math.Point3, u math.Vector3) float64 {
	w := axisPt.VectorTo(p)
	perp := w.Sub(u.Scale(w.Dot(u)))
	return perp.Length()
}

// p9Cylinders is the exact P9 rolling-ball pair: the r1 fillet on the x=5 arm and the r0.5 fillet on the
// y=0 arm of a 5-cube, both tangent to the shared top plane z=5 (outward normal +z).
func p9Cylinders() (math.Vector3, miterCyl, miterCyl) {
	nS := math.V3(0, 0, 1)
	c0 := miterCyl{cen: math.P3(4, 0, 4), axis: math.V3(0, 1, 0), nF: math.V3(1, 0, 0), r: 1}
	c1 := miterCyl{cen: math.P3(5, 0.5, 4.5), axis: math.V3(1, 0, 0), nF: math.V3(0, -1, 0), r: 0.5}
	return nS, c0, c1
}

// TestAsymmetricMiterSeamLiesOnBothCylinders is the core watertightness invariant: every sampled seam
// point must lie on BOTH unequal fillet cylinders (that is what lets the two arm faces weld along one
// shared polyline). It also pins the endpoints against OCCT's exact P9 seam: sTop=(4,0.5,5) where both
// fillets meet the shared plane, sBot=(4.866…,0,4.5) where the tighter r0.5 fillet runs out on its own
// outer face y=0.
func TestAsymmetricMiterSeamLiesOnBothCylinders(t *testing.T) {
	nS, c0, c1 := p9Cylinders()
	seam, err := sampleAsymmetricMiterSeam(nS, c0, c1)
	if err != nil {
		t.Fatalf("sampleAsymmetricMiterSeam: %v", err)
	}
	if len(seam) < 5 {
		t.Fatalf("seam has %d points, want ≥5 (a chorded arc)", len(seam))
	}
	const tol = 1e-9
	for i, p := range seam {
		if d := perpDistToAxis(p, c0.cen, c0.axis); stdmath.Abs(d-c0.r) > tol {
			t.Fatalf("seam[%d]=%v is %.12f from axis0, want r0=%g (off the r1 cylinder)", i, p, d, c0.r)
		}
		if d := perpDistToAxis(p, c1.cen, c1.axis); stdmath.Abs(d-c1.r) > tol {
			t.Fatalf("seam[%d]=%v is %.12f from axis1, want r1=%g (off the r0.5 cylinder)", i, p, d, c1.r)
		}
	}
	if got := seam[0]; !got.IsEqualTo(math.P3(4, 0.5, 5), 1e-9) {
		t.Fatalf("sTop=%v, want (4,0.5,5)", got)
	}
	want := math.P3(4+stdmath.Sqrt(0.75), 0, 4.5)
	if got := seam[len(seam)-1]; !got.IsEqualTo(want, 1e-9) {
		t.Fatalf("sBot=%v, want %v (r0.5 fillet run-out on y=0)", got, want)
	}
}

// TestAsymmetricMiterTerminusPicksTighterArm pins the run-out rule: the seam ends where the SMALLER-wedge
// fillet (r0.5) first reaches its outer face, not where the larger one (r1) does — the candidate that
// keeps the other arm's contact direction inside its rolling-ball wedge.
func TestAsymmetricMiterTerminusPicksTighterArm(t *testing.T) {
	nS, c0, c1 := p9Cylinders()
	sBot, ok := asymMiterTerminus(nS, c0, c1)
	if !ok {
		t.Fatal("asymMiterTerminus found no valid terminus")
	}
	// sBot must lie on the r0.5 arm's outer face y=0 (its run-out), NOT the r1 arm's outer face x=5.
	if stdmath.Abs(sBot.Y) > 1e-9 {
		t.Fatalf("sBot=%v not on y=0 (the tighter r0.5 arm's outer face)", sBot)
	}
	if stdmath.Abs(sBot.X-5) < 1e-6 {
		t.Fatalf("sBot=%v ran out on the r1 arm's x=5 face — should be the tighter r0.5 arm", sBot)
	}
}

// TestAsymmetricMiterSeamIsOrderInvariant confirms swapping the two arms yields the SAME seam point set
// (reversed) — the seam is a property of the cylinder pair, not of which pick is arm0.
func TestAsymmetricMiterSeamIsOrderInvariant(t *testing.T) {
	nS, c0, c1 := p9Cylinders()
	s01, err := sampleAsymmetricMiterSeam(nS, c0, c1)
	if err != nil {
		t.Fatalf("seam(c0,c1): %v", err)
	}
	s10, err := sampleAsymmetricMiterSeam(nS, c1, c0)
	if err != nil {
		t.Fatalf("seam(c1,c0): %v", err)
	}
	// Endpoints are the same physical points regardless of arm order (sTop shared; sBot on y=0).
	if !s01[0].IsEqualTo(s10[0], 1e-9) {
		t.Fatalf("sTop differs by arm order: %v vs %v", s01[0], s10[0])
	}
	if !s01[len(s01)-1].IsEqualTo(s10[len(s10)-1], 1e-9) {
		t.Fatalf("sBot differs by arm order: %v vs %v", s01[len(s01)-1], s10[len(s10)-1])
	}
}
