// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestAxialWindowParamsClipsAnUnboundedSection pins the inverse of AxialExtent (ADR-0062): a plane
// cutting a ruled wall parallel to its axis sections it in a curve that runs to infinity, and a chart
// samples an imprint over its own domain — an infinite domain samples to nothing.
func TestAxialWindowParamsClipsAnUnboundedSection(t *testing.T) {
	t.Parallel()
	axis := math.V3(0, 0, 1)
	origin := math.P3(0, 0, 0)
	// An infinite line along +z through (1,2,·): its axial coordinate is affine and monotone.
	line, err := NewLine(math.P3(1, 2, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("line: %v", err)
	}
	spans, ok := AxialWindowParams(line, origin, axis, 3, 8)
	if !ok || len(spans) != 1 {
		t.Fatalf("a monotone section clips to ONE span, got ok=%v spans=%v", ok, spans)
	}
	for _, want := range []float64{3, 8} {
		found := false
		for _, e := range []float64{spans[0][0], spans[0][1]} {
			if stdmath.Abs(float64(origin.VectorTo(line.PointAt(e)).Dot(axis))-want) < 1e-9 {
				found = true
			}
		}
		if !found {
			t.Errorf("no clipped end sits at v=%g; got v=%g..%g", want,
				float64(origin.VectorTo(line.PointAt(spans[0][0])).Dot(axis)),
				float64(origin.VectorTo(line.PointAt(spans[0][1])).Dot(axis)))
		}
	}
	// Every point strictly inside the span must be inside the window.
	mid := (spans[0][0] + spans[0][1]) / 2
	if v := float64(origin.VectorTo(line.PointAt(mid)).Dot(axis)); v < 3 || v > 8 {
		t.Errorf("the span's midpoint sits at v=%g, outside [3, 8]", v)
	}
}

// TestAxialWindowParamsTurnsAtAConicVertex: a hyperbola arm's axial coordinate is NOT monotone — it
// turns at the vertex — so the window is inverted on each monotone piece, not on the whole curve.
func TestAxialWindowParamsTurnsAtAConicVertex(t *testing.T) {
	t.Parallel()
	axis := math.V3(0, 0, 1)
	apex := math.P3(0, 0, 0)
	cone, err := NewCone(apex, math.V3(0, 0, 1), stdmath.Atan(0.3))
	if err != nil {
		t.Fatalf("cone: %v", err)
	}
	pl, err := NewPlane(math.P3(1.2, 0, 0), math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	curves, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(20))
	if !handled || len(curves) != 1 {
		t.Fatalf("an axis-parallel plane sections a cone in one hyperbola arm; got handled=%v n=%d", handled, len(curves))
	}
	arm := curves[0]
	if lo, hi := arm.Domain(); !stdmath.IsInf(lo, 0) || !stdmath.IsInf(hi, 0) {
		t.Skipf("the section is already bounded [%g %g]: this case no longer exercises the clip", lo, hi)
	}
	spans, ok := AxialWindowParams(arm, apex, axis, 4.5, 13)
	if !ok {
		t.Fatal("the clip declined a hyperbola arm")
	}
	for _, sp := range spans {
		for _, s := range []float64{0, 0.25, 0.5, 0.75, 1} {
			p := arm.PointAt(sp[0] + s*(sp[1]-sp[0]))
			v := float64(apex.VectorTo(p).Dot(axis))
			if v < 4.5-1e-6 || v > 13+1e-6 {
				t.Errorf("a clipped point sits at v=%g, outside [4.5, 13]", v)
			}
		}
	}
	// The arm's vertex is at v≈4 — below the window — so the window is reached on BOTH sides of it and
	// the clip must report two spans, not one that swallows the vertex.
	if len(spans) != 2 {
		t.Errorf("the window sits above the arm's vertex, so it is reached on both sides: got %d span(s)", len(spans))
	}
}
