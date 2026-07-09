// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	obmath "oblikovati.org/math"
)

// Two-point distance dimension orientation — Inventor's DimensionOrientationEnum (#1869): aligned
// measures the Euclidean distance, horizontal the X separation only, vertical the Y separation.

// TestDistanceOrientationMeasuresComponent: for the points (0,0) and (3,4) the aligned dim measures
// 5, the horizontal dim 3 (|Δx|), the vertical dim 4 (|Δy|) — proof each orientation constrains its
// own component. Each also reports its orientation.
func TestDistanceOrientationMeasuresComponent(t *testing.T) {
	for _, tc := range []struct {
		o    DistanceOrientation
		want float64
	}{
		{AlignedDistance, 5},
		{HorizontalDistance, 3},
		{VerticalDistance, 4},
	} {
		s := NewSketches().Add(XYPlane())
		a, b := s.NewPoint(obmath.P2(0, 0)), s.NewPoint(obmath.P2(3, 4))
		d, err := s.DimensionConstraints().AddDistanceOriented(a, b, "1 cm", tc.o)
		if err != nil {
			t.Fatalf("orientation %d: %v", tc.o, err)
		}
		if got := d.Measured(); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("orientation %d measured = %g, want %g", tc.o, got, tc.want)
		}
		if d.Orientation() != tc.o {
			t.Errorf("Orientation() = %d, want %d", d.Orientation(), tc.o)
		}
	}
}

// TestHorizontalDistanceSolvesXLeavesY: with point a grounded at the origin and b free, a
// horizontal "3 cm" dim drives |b.x − a.x| to 3 while leaving b.y at its initial value — the pair
// keeps a translational DOF along Y (AC #1869).
func TestHorizontalDistanceSolvesXLeavesY(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a, b := s.NewPoint(obmath.P2(0, 0)), s.NewPoint(obmath.P2(10, 7))
	s.GeometricConstraints().AddGround(a)
	if _, err := s.DimensionConstraints().AddDistanceOriented(a, b, "3 cm", HorizontalDistance); err != nil {
		t.Fatal(err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if dx := math.Abs(float64(b.X) - float64(a.X)); math.Abs(dx-3) > 1e-6 {
		t.Errorf("|Δx| after solve = %g, want 3", dx)
	}
	if math.Abs(float64(b.Y)-7) > 1e-6 {
		t.Errorf("b.y = %g, want 7 (Y left free/unmoved by a horizontal dim)", float64(b.Y))
	}
}

// TestDistanceDimensionRemovesOneDOF: any single distance dim (here horizontal) removes exactly one
// degree of freedom from the two otherwise-free points (4 → 3).
func TestDistanceDimensionRemovesOneDOF(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a, b := s.NewPoint(obmath.P2(0, 0)), s.NewPoint(obmath.P2(3, 4))
	before := s.DegreesOfFreedom()
	if _, err := s.DimensionConstraints().AddDistanceOriented(a, b, "3 cm", HorizontalDistance); err != nil {
		t.Fatal(err)
	}
	if after := s.DegreesOfFreedom(); after != before-1 {
		t.Errorf("DOF %d → %d, want one fewer (a horizontal dim is a single scalar constraint)", before, after)
	}
}

// TestDistanceOrientationSurvivesRoundTrip: a horizontal distance dim keeps its orientation across
// a serialize/restore (the default aligned serializes nothing, so a directed dim exercises the
// codec). #1869.
func TestDistanceOrientationSurvivesRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	a, b := s.NewPoint(obmath.P2(0, 0)), s.NewPoint(obmath.P2(3, 4))
	if _, err := s.DimensionConstraints().AddDistanceOriented(a, b, "3 cm", HorizontalDistance); err != nil {
		t.Fatal(err)
	}
	out := roundTrip(t, sc)
	dims := out.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("restored dimension count = %d, want 1", len(dims))
	}
	if got := dims[0].Orientation(); got != HorizontalDistance {
		t.Errorf("restored orientation = %d, want HorizontalDistance (%d)", got, HorizontalDistance)
	}
}

// TestDistanceOrientationUnknownName guards the name parser.
func TestDistanceOrientationUnknownName(t *testing.T) {
	if _, ok := ParseDistanceOrientation("diagonal"); ok {
		t.Error("unknown orientation name should not parse")
	}
	for _, n := range []string{"", "aligned", "horizontal", "vertical"} {
		if _, ok := ParseDistanceOrientation(n); !ok {
			t.Errorf("orientation %q should parse", n)
		}
	}
}
