// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

func TestSketchDimensionsNilOutsideSketch(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if got := s.SketchDimensions(); got != nil {
		t.Errorf("SketchDimensions outside a sketch = %v, want nil", got)
	}
}

func TestDistanceDimensionView(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(3, 0)) // 3 cm = 30 mm
	if _, err := sk.DimensionConstraints().AddDistance(a, b, "30 mm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	views := s.SketchDimensions()
	if len(views) != 1 {
		t.Fatalf("got %d dimension views, want 1", len(views))
	}
	v := views[0]
	if !strings.Contains(v.Label, "30") {
		t.Errorf("distance label = %q, want it to show 30 mm", v.Label)
	}
	if len(v.Segments) != 7 { // two witness lines + the dimension line + two arrowheads
		t.Errorf("distance dimension has %d segments, want 7", len(v.Segments))
	}
}

func TestRadiusAndDiameterViewsDiffer(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	if _, err := sk.DimensionConstraints().AddRadius(c, "20 mm"); err != nil {
		t.Fatalf("AddRadius: %v", err)
	}
	views := s.SketchDimensions()
	if len(views) != 1 || !strings.HasPrefix(views[0].Label, radiusPrefix) {
		t.Fatalf("radius view = %+v, want one labeled with %q", views, radiusPrefix)
	}
	// A radius leader is one segment; the diameter line spans the circle (still one
	// segment) but is labeled differently — verify the prefix distinguishes them.
	if _, err := sk.DimensionConstraints().AddDiameter(c, "40 mm"); err != nil {
		t.Fatalf("AddDiameter: %v", err)
	}
	views = s.SketchDimensions()
	var sawDiameter bool
	for _, v := range views {
		if strings.HasPrefix(v.Label, diameterPrefix) {
			sawDiameter = true
		}
	}
	if !sawDiameter {
		t.Errorf("expected a diameter-prefixed (%q) label among %d views", diameterPrefix, len(views))
	}
}

func TestAngleDimensionViewHasArc(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 4))
	if _, err := sk.DimensionConstraints().AddAngle(l1, l2, "90 deg"); err != nil {
		t.Fatalf("AddAngle: %v", err)
	}
	views := s.SketchDimensions()
	if len(views) != 1 {
		t.Fatalf("got %d angle views, want 1", len(views))
	}
	if len(views[0].Segments) != angleArcSegments {
		t.Errorf("angle arc has %d segments, want %d", len(views[0].Segments), angleArcSegments)
	}
	if !strings.Contains(views[0].Label, "deg") {
		t.Errorf("angle label = %q, want degrees", views[0].Label)
	}
}

func TestParallelLinesAngleViewSkipped(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(4, 1)) // parallel: no vertex
	if _, err := sk.DimensionConstraints().AddAngle(l1, l2, "0 deg"); err != nil {
		t.Fatalf("AddAngle: %v", err)
	}
	if got := s.SketchDimensions(); len(got) != 0 {
		t.Errorf("parallel-line angle should be skipped (no vertex), got %d views", len(got))
	}
}
