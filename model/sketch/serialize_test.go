// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// roundTrip serializes sc and rebuilds it in a fresh collection, returning the
// reopened first sketch — the real save→restore path the part recipe uses.
func roundTrip(t *testing.T, sc *Sketches) *Sketch {
	t.Helper()
	data, err := sc.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewSketches()
	if err := fresh.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != sc.Count() {
		t.Fatalf("sketch count after round trip = %d, want %d", fresh.Count(), sc.Count())
	}
	return fresh.Item(0)
}

// TestRectangleSharedPointsSurvive proves that a rectangle built from four lines
// sharing corner points round-trips as four lines and four points (not eight), i.e.
// the structural coincidence at each corner is preserved.
func TestRectangleSharedPointsSurvive(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	// Four corners as shared points.
	p00 := s.newPoint(math.P2(0, 0))
	p40 := s.newPoint(math.P2(4, 0))
	p43 := s.newPoint(math.P2(4, 3))
	p03 := s.newPoint(math.P2(0, 3))
	s.lines.Add(p00, p40)
	s.lines.Add(p40, p43)
	s.lines.Add(p43, p03)
	s.lines.Add(p03, p00)

	out := roundTrip(t, sc)
	if got := out.Lines().Count(); got != 4 {
		t.Errorf("line count = %d, want 4", got)
	}
	if got := len(out.AllPoints()); got != 4 {
		t.Errorf("point count = %d, want 4 (shared corners must not be duplicated)", got)
	}
	// The four lines must still form a closed loop: each corner point is shared by
	// exactly two lines, so a single closed profile is detected.
	if got := out.Profiles().Count(); got != 1 {
		t.Errorf("profile count = %d, want 1 closed loop", got)
	}
}

// TestEntityKindsSurvive round-trips one of every curve entity and checks counts +
// key geometry.
func TestEntityKindsSurvive(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	s.Circles().AddByCenterRadius(math.P2(5, 5), 2)
	s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), true)
	s.Ellipses().Add(math.P2(2, 2), math.V2(1, 0), 3, 1)
	s.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 1), math.P2(2, 0)}, false)
	s.Points().Add(math.P2(9, 9))

	out := roundTrip(t, sc)
	if out.Lines().Count() != 1 || out.Circles().Count() != 1 || out.Arcs().Count() != 1 ||
		out.Ellipses().Count() != 1 || out.Splines().Count() != 1 || out.Points().Count() != 1 {
		t.Errorf("entity counts after round trip: lines=%d circles=%d arcs=%d ellipses=%d splines=%d points=%d, want all 1",
			out.Lines().Count(), out.Circles().Count(), out.Arcs().Count(), out.Ellipses().Count(), out.Splines().Count(), out.Points().Count())
	}
	if got := out.Circles().Item(0).Radius; got != 2 {
		t.Errorf("circle radius = %v, want 2", got)
	}
}

// TestConstraintsSurvive round-trips a representative geometric constraint and a
// dimensional constraint and checks they are rebuilt.
func TestConstraintsSurvive(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(4, 1))
	s.GeometricConstraints().AddParallel(l1, l2)
	s.GeometricConstraints().AddCoincident(l1.A, l2.A)
	if _, err := s.DimensionConstraints().AddDistance(l1.A, l1.B, "40 mm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	out := roundTrip(t, sc)
	if got := out.GeometricConstraints().Count(); got != 2 {
		t.Errorf("geometric constraint count = %d, want 2", got)
	}
	if got := out.DimensionConstraints().Count(); got != 1 {
		t.Errorf("dimension count = %d, want 1", got)
	}
	dim := out.DimensionConstraints().Item(0)
	if dim.Kind() != DistanceDim || dim.Parameter().Expression() != "40 mm" {
		t.Errorf("restored dimension = kind %v expr %q, want distance 40 mm", dim.Kind(), dim.Parameter().Expression())
	}
}

// TestPlaneSurvives round-trips a non-origin plane.
func TestPlaneSurvives(t *testing.T) {
	sc := NewSketches()
	sc.Add(XZPlane())
	out := roundTrip(t, sc)
	got := out.Plane()
	want := XZPlane()
	if !got.Normal().AsVector().IsEqualTo(want.Normal().AsVector(), 1e-9) {
		t.Errorf("plane normal = %v, want %v", got.Normal(), want.Normal())
	}
}

// TestBlocksErrorRatherThanDropSilently guards the no-silent-loss rule for an as-yet
// unsupported feature.
func TestBlocksErrorRatherThanDropSilently(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	def := s.Blocks().DefineBlock("b")
	s.Blocks().Insert(def, math.Identity3())
	if _, err := sc.MarshalRecipe(); err == nil {
		t.Error("MarshalRecipe silently accepted a sketch with blocks; it must error")
	}
}
