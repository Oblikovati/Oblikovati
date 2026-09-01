// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSketchDistanceDimensionExtraction proves the model-dimension host resolver's core mapping: a
// part sketch's distance dimension is resolved to a ModelDimension with the parameter name, the
// solved value, and the two world endpoints via the sketch plane (#1991).
func TestSketchDistanceDimensionExtraction(t *testing.T) {
	t.Parallel()
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(3, 0)) // starts 3 cm apart
	if _, err := s.DimensionConstraints().AddDistance(a, b, "5 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	s.Solve() // drives the distance to 5 cm

	dims := sketchDistanceDimensions(s)
	if len(dims) != 1 {
		t.Fatalf("extracted %d dimensions, want 1", len(dims))
	}
	md := dims[0]
	if md.Name == "" {
		t.Error("extracted dimension has no parameter name")
	}
	if stdmath.Abs(md.Value-5) > 1e-6 {
		t.Errorf("value = %v cm, want 5 (the solved parameter)", md.Value)
	}
	// XY-plane: sketch (u,v) maps to world (x,y,0). The two endpoints lie on the X axis (y=z=0) and
	// span 5 cm (the solver may recentre both points, so assert the span, not absolute positions).
	if stdmath.Abs(float64(md.A.Y)) > 1e-6 || stdmath.Abs(float64(md.A.Z)) > 1e-6 || stdmath.Abs(float64(md.B.Y)) > 1e-6 {
		t.Errorf("endpoints off the X axis: %v–%v", md.A, md.B)
	}
	if span := stdmath.Abs(float64(md.B.X - md.A.X)); stdmath.Abs(span-5) > 1e-6 {
		t.Errorf("endpoint span = %v cm, want 5", span)
	}
}
