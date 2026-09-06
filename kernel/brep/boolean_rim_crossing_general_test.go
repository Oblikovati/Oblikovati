// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A rod driven obliquely through a cylinder's wall and out through its top RIM — so the exit crossing
// straddles the rim instead of landing inside a cap — must weld through the general pipeline. It is the
// case the hand-written rim-crossing recognizer was built for (#1724 slice 2), and the general path
// reaches it once the wall crossing is CLIPPED to the band rather than refused for straddling a rim, and
// the cap's bite is allowed to end on the cap's own CIRCULAR boundary (ADR-0061 stage 4).
//
// The result is four faces: the notched holed wall, the bitten top cap, the whole bottom cap, and the
// tunnel.
func TestRimCrossingCutWeldsThroughTheGeneralPath(t *testing.T) {
	t.Parallel()
	s := math.Scalar(1 / stdmath.Sqrt2)
	target, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder target: %v", err)
	}
	tool, err := SolidCylinder(math.P3(-5.6, 0, 2), math.V3(s, 0, s), 0.9, 16)
	if err != nil {
		t.Fatalf("SolidCylinder tool: %v", err)
	}
	res, err := Boolean(Difference, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Difference) on the rim crossing: %v", err)
	}
	assertWatertight(t, res)
	if got := len(res.Faces()); got != 4 {
		t.Errorf("rim-crossing cut has %d faces, want 4 (notched wall + bitten top cap + bottom cap + tunnel)", got)
	}
}

// An open section imprint that ends on a face's CIRCULAR boundary must have its crossings solved against
// that circle, not skipped for being non-straight: skipped, the imprint entered the arrangement as a
// chord dangling inside the face, bounded nothing, and the face came back whole.
func TestOpenImprintCrossesACurvedFrameEdge(t *testing.T) {
	t.Parallel()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	rim, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	// A circle of radius 1 centred at (2.5, 0): it crosses the r=3 rim in exactly two points.
	sec, err := geom.NewCircle(math.P3(2.5, 0, 0), math.V3(0, 0, 1), 1)
	if err != nil {
		t.Fatalf("NewCircle section: %v", err)
	}
	pc, ok := toPlaneConic(sec, pl)
	if !ok {
		t.Fatalf("toPlaneConic declined a circle in its own plane")
	}
	lo, hi := rim.Domain()
	pts, tangent, got := conicEdgeCrossingPoints(pc, loopEdge{curve: rim, t0: lo, t1: hi}, pl, geom.ResolutionForSize(6))
	if !got || tangent {
		t.Fatalf("conicEdgeCrossingPoints: ok=%v tangent=%v, want ok and no graze", got, tangent)
	}
	if len(pts) != 2 {
		t.Fatalf("got %d crossings of a r=1 circle at x=2.5 with the r=3 rim, want 2", len(pts))
	}
	for _, p := range pts {
		p3 := to3D(pl, p)
		if r := stdmath.Hypot(float64(p3.X), float64(p3.Y)); stdmath.Abs(r-3) > 1e-9 {
			t.Errorf("crossing at radius %.12f, want 3 (it must lie ON the rim)", r)
		}
		if d := stdmath.Hypot(float64(p3.X)-2.5, float64(p3.Y)); stdmath.Abs(d-1) > 1e-9 {
			t.Errorf("crossing at distance %.12f from the section centre, want 1", d)
		}
	}
}
