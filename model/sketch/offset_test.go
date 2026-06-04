// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

func TestOffsetLineIsParallelAtDistance(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	off, err := s.OffsetEntity(l, 1)
	if err != nil {
		t.Fatalf("OffsetEntity: %v", err)
	}
	ol := off.(*Line)
	// Left normal of +X is +Y ⇒ the offset line sits at y = 1.
	if math.Abs(float64(ol.A.Y)-1) > 1e-9 || math.Abs(float64(ol.B.Y)-1) > 1e-9 {
		t.Fatalf("offset line at y = %v/%v, want 1", ol.A.Y, ol.B.Y)
	}
}

func TestOffsetCircleGrowsRadius(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(0, 0), 1)
	off, err := s.OffsetEntity(c, 0.5)
	if err != nil {
		t.Fatalf("OffsetEntity: %v", err)
	}
	if r := float64(off.(*Circle).Radius); math.Abs(r-1.5) > 1e-9 {
		t.Fatalf("offset circle radius = %v, want 1.5", r)
	}
}

func TestOffsetCircleRejectsCollapse(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(0, 0), 1)
	if _, err := s.OffsetEntity(c, -1); err == nil {
		t.Error("offsetting a circle to radius 0 should error")
	}
}

func TestSketchImageRoundTrips(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	s.Images().Add("pkg://trace.png", gmath.P2(1, 2), 8, 6, 0.5, 0.75)

	out := roundTrip(t, sc)
	if out.Images().Count() != 1 {
		t.Fatalf("images after round trip = %d, want 1", out.Images().Count())
	}
	img := out.Images().Item(0)
	if img.Ref != "pkg://trace.png" || float64(img.Width) != 8 || float64(img.Height) != 6 || img.Opacity != 0.75 {
		t.Fatalf("image = %+v, want trace.png 8×6 opacity 0.75", img)
	}
}
