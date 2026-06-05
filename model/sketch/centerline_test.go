// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

// squareSk draws a side×side square (a closed region) on the sketch.
func squareSk(s *Sketch, side float64) {
	a := s.Points().Add(gmath.P2(0, 0))
	b := s.Points().Add(gmath.P2(gmath.Scalar(side), 0))
	c := s.Points().Add(gmath.P2(gmath.Scalar(side), gmath.Scalar(side)))
	d := s.Points().Add(gmath.P2(0, gmath.Scalar(side)))
	s.Lines().Add(a, b)
	s.Lines().Add(b, c)
	s.Lines().Add(c, d)
	s.Lines().Add(d, a)
}

// A centerline (like any construction geometry) is excluded from profiles, so a midline drawn
// as a centerline does NOT split the region.
func TestCenterlineDoesNotCloseProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	squareSk(s, 4)
	if got := s.Profiles().Count(); got != 1 {
		t.Fatalf("square = %d profiles, want 1", got)
	}
	cl := s.Lines().AddByTwoPoints(gmath.P2(0, 2), gmath.P2(4, 2)) // midline
	cl.SetCenterline(true)
	if !cl.IsConstruction() {
		t.Error("a centerline must be construction geometry")
	}
	if got := s.Profiles().Count(); got != 1 {
		t.Errorf("centerline split the region into %d profiles, want 1 (it must not close paths)", got)
	}
}

// Control: the SAME midline as ordinary geometry DOES split the square into two regions —
// confirming the centerline exclusion above is what keeps it whole.
func TestNormalMidlineSplitsProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	squareSk(s, 4)
	s.Lines().AddByTwoPoints(gmath.P2(0, 2), gmath.P2(4, 2))
	if got := s.Profiles().Count(); got != 2 {
		t.Errorf("normal midline gave %d regions, want 2", got)
	}
}

// A centerline is reported by Centerlines() and yields a model-space axis for revolve/mirror.
func TestCenterlineAxisAndAccessor(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	cl := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(0, 4)) // vertical (Y) axis
	cl.SetCenterline(true)
	if len(s.Centerlines()) != 1 {
		t.Fatalf("Centerlines() = %d, want 1", len(s.Centerlines()))
	}
	o, dir := cl.Axis3D(XYPlane())
	if !o.IsEqualTo(gmath.P3(0, 0, 0), 1e-9) {
		t.Errorf("axis origin = %v, want (0,0,0)", o)
	}
	if float64(dir.X) != 0 || float64(dir.Y) != 4 || float64(dir.Z) != 0 {
		t.Errorf("axis dir = %v, want (0,4,0)", dir)
	}
}

// A centerline drives a sketch mirror (MirrorEntities already reflects across a *Line).
func TestMirrorAcrossCenterline(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	cl := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(0, 4)) // mirror about the Y axis (x=0)
	cl.SetCenterline(true)
	src := s.Lines().AddByTwoPoints(gmath.P2(2, 1), gmath.P2(3, 1))
	mirrored := s.MirrorEntities([]Entity{src}, cl)
	if len(mirrored) != 1 {
		t.Fatalf("mirror produced %d entities, want 1", len(mirrored))
	}
	ln := mirrored[0].(*Line)
	if float64(ln.A.Position().X) != -2 || float64(ln.B.Position().X) != -3 {
		t.Errorf("mirrored x = (%v,%v), want (-2,-3)", ln.A.Position().X, ln.B.Position().X)
	}
}

func TestCenterlineRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	cl := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(5, 0))
	cl.SetCenterline(true)
	out := roundTrip(t, sc)
	cls := out.Centerlines()
	if len(cls) != 1 || !cls[0].IsConstruction() {
		t.Errorf("after round trip: %d centerlines (want 1, construction)", len(cls))
	}
}
