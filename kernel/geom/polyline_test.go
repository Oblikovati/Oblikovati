// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

func TestPolylineEvaluationAndLength(t *testing.T) {
	t.Parallel()
	p, err := NewPolyline([]math.Point3{
		math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	approxScalar(t, p.Length(), 2, "Length")
	cases := []struct {
		t    float64
		want math.Point3
	}{
		{0, math.P3(0, 0, 0)},
		{0.25, math.P3(0.5, 0, 0)}, // mid of first segment (t∈[0,0.5])
		{0.5, math.P3(1, 0, 0)},    // shared vertex
		{0.75, math.P3(1, 0.5, 0)}, // mid of second segment
		{1, math.P3(1, 1, 0)},
	}
	for _, c := range cases {
		if got := p.PointAt(c.t); !got.IsEqualTo(c.want, eqScalar) {
			t.Errorf("PointAt(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestPolylineCopiesVertices(t *testing.T) {
	t.Parallel()
	src := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}
	p, _ := NewPolyline(src)
	src[0] = math.P3(99, 99, 99) // mutating the source must not affect the polyline
	if !p.PointAt(0).IsEqualTo(math.P3(0, 0, 0), eqScalar) {
		t.Error("polyline did not copy its vertices")
	}
}

func TestNewPolylineTooFewVerticesFails(t *testing.T) {
	t.Parallel()
	if _, err := NewPolyline([]math.Point3{math.P3(0, 0, 0)}); err == nil {
		t.Fatal("expected error for single vertex")
	}
}

func TestPolyline2dEvaluation(t *testing.T) {
	t.Parallel()
	p, err := NewPolyline2d([]math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(2, 2)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	approxScalar(t, p.Length(), 4, "Length2d")
	if got := p.PointAt(1); !got.IsEqualTo(math.P2(2, 2), eqScalar) {
		t.Errorf("PointAt(1) = %v, want {2 2}", got)
	}
}
