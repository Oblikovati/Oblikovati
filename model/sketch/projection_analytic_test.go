// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// analyticEdge is a fake model edge that exposes its exact analytic curve (an [AnalyticCurveSource]),
// so a projection of it stays analytic instead of a sampled polyline (ADR-0055).
type analyticEdge struct {
	id    string
	curve geom.Curve3
}

func (e *analyticEdge) SourceID() string   { return e.id }
func (e *analyticEdge) SourceKind() string { return "edge" }
func (e *analyticEdge) SamplePoints() ([]math.Point3, bool) {
	return geom.SampleCurve3(e.curve, 16), true
}
func (e *analyticEdge) SourceCurve() (geom.Curve3, bool) { return e.curve, true }

// TestProjectedCircleStaysAnalytic: projecting a model circle parallel to the sketch plane yields an
// analytic geom.Circle2d of the same radius — not a 17-point polyline. This is the piston-rim case.
func TestProjectedCircleStaysAnalytic(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	circ, _ := geom.NewCircle(math.P3(2, 3, 5), math.V3(0, 0, 1), 4) // parallel to XY at z=5
	pc := s.ProjectCurve(&analyticEdge{id: "e1", curve: circ})

	c2, ok := pc.AnalyticCurve()
	if !ok {
		t.Fatal("projected circle is not analytic (fell back to a polyline)")
	}
	cc, isCircle := c2.(geom.Circle2d)
	if !isCircle {
		t.Fatalf("want geom.Circle2d, got %T", c2)
	}
	if cc.Center.DistanceTo(math.P2(2, 3)) > 1e-9 || stdmath.Abs(cc.Radius-4) > 1e-9 {
		t.Fatalf("projected circle = centre %v r %g, want (2,3) r 4", cc.Center, cc.Radius)
	}
}

// TestProjectedNonAnalyticFallsBack: a source that only yields sample points (no analytic curve)
// projects to a polyline, and AnalyticCurve reports false — the fallback path still works.
func TestProjectedNonAnalyticFallsBack(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pc := s.ProjectCurve(&movableEdge{id: "e2", samples: []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 0.3, 0), math.P3(2, -0.2, 0),
	}})
	if _, ok := pc.AnalyticCurve(); ok {
		t.Fatal("a sample-only source must not report an analytic curve")
	}
	if len(pc.Points()) == 0 {
		t.Fatal("fallback projection lost its polyline")
	}
}
