// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

func TestSamplePolyline3DShapes(t *testing.T) {
	s := NewSketches3D().Add()
	zAxis, err := gmath.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("axis: %v", err)
	}

	p := s.AddPoint3D(gmath.P3(1, 2, 3))
	if pts := SamplePolyline3D(p, 16); len(pts) != 1 || pts[0] != gmath.P3(1, 2, 3) {
		t.Errorf("point sample = %v, want its single position", pts)
	}

	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	if pts := SamplePolyline3D(l, 16); len(pts) != 2 || pts[0] != gmath.P3(0, 0, 0) || pts[1] != gmath.P3(1, 0, 0) {
		t.Errorf("line sample = %v, want its two endpoints", pts)
	}

	c := s.AddCircle3D(gmath.P3(0, 0, 0), zAxis, 2)
	pts := SamplePolyline3D(c, 32)
	if len(pts) != 33 {
		t.Fatalf("circle sample count = %d, want segments+1 = 33", len(pts))
	}
	for i, q := range pts {
		r := float64(gmath.P3(0, 0, 0).DistanceTo(q))
		if stdmath.Abs(r-2) > 1e-9 {
			t.Fatalf("circle sample %d at radius %g, want 2", i, r)
		}
	}

	h := s.AddHelix3D(gmath.P3(0, 0, 0), zAxis, 1, 0.5, 0, 4, false)
	hp := SamplePolyline3D(h, 16)
	if len(hp) != 4*16+1 {
		t.Errorf("helix sample count = %d, want turns*segments+1 = 65", len(hp))
	}
	if rise := float64(hp[len(hp)-1].Z - hp[0].Z); stdmath.Abs(rise-2) > 1e-9 {
		t.Errorf("helix rise = %g, want pitch*turns = 2", rise)
	}
}

func TestSamplePolyline3DUnknownKindIsNil(t *testing.T) {
	s := NewSketches3D().Add()
	// An included reference curve has no analytic kernel curve of its own — the
	// sampler must report nil rather than guessing.
	inc := s.IncludeCurve3D(&fakeCurveSource{id: "e", pts: []gmath.Point3{{X: 0}, {X: 1}}})
	if pts := SamplePolyline3D(inc, 8); pts != nil {
		t.Errorf("unsupported entity sampled to %v, want nil", pts)
	}
}
