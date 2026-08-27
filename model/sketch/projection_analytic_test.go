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

// TestProjectedCircleSerializesAnalytic: a projected circle persists as a compact analytic
// descriptor (3 floats) with NO coords polyline, and round-trips back to an analytic circle
// (ADR-0055 — the fix for the 34-float-per-curve document bloat).
func TestProjectedCircleSerializesAnalytic(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	circ, _ := geom.NewCircle(math.P3(2, 3, 5), math.V3(0, 0, 1), 4)
	s.ProjectCurve(&analyticEdge{id: "c9", curve: circ})

	data, err := sc.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	var ed EntityData
	for _, e := range data[0].Entities {
		if e.Source == "c9" {
			ed = e
		}
	}
	if ed.ProjShape != "circle" || len(ed.ProjParams) != 3 {
		t.Fatalf("serialized projShape=%q params=%v, want circle [2 3 4]", ed.ProjShape, ed.ProjParams)
	}
	if len(ed.Coords) != 0 {
		t.Fatalf("serialized a %d-float coords polyline; an analytic curve stores none", len(ed.Coords))
	}

	out := roundTrip(t, sc)
	var rc *ProjectedCurve
	for _, e := range out.Entities() {
		if c, ok := e.(*ProjectedCurve); ok {
			rc = c
		}
	}
	if rc == nil {
		t.Fatal("projected curve lost on round trip")
	}
	c2, ok := rc.AnalyticCurve()
	if !ok {
		t.Fatal("restored projected curve is not analytic")
	}
	if cc, isCircle := c2.(geom.Circle2d); !isCircle || stdmath.Abs(cc.Radius-4) > 1e-9 {
		t.Fatalf("restored curve = %T, want Circle2d r 4", c2)
	}
}

// TestProjectedCircleOffsetsAnalytic: offsetting an analytic projected circle yields a concentric
// analytic circle — not a faceted polyline. This is the offset robustness the analytic
// representation buys (ADR-0055): segment polylines break offset; analytic curves offset cleanly.
func TestProjectedCircleOffsetsAnalytic(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	circ, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	pc := s.ProjectCurve(&analyticEdge{id: "e5", curve: circ})

	e, err := s.OffsetEntity(pc, 1)
	if err != nil {
		t.Fatalf("offset: %v", err)
	}
	c, ok := e.(*Circle)
	if !ok {
		t.Fatalf("offset of a projected circle = %T, want an analytic *Circle (not a faceted chain)", e)
	}
	if r := float64(c.Radius); stdmath.Abs(r-6) > 1e-9 && stdmath.Abs(r-4) > 1e-9 {
		t.Fatalf("offset circle radius = %g, want 4 or 6 (concentric)", r)
	}
}

// TestProjectedArcOffsetsAnalytic: offsetting an analytic projected arc yields a concentric analytic
// arc of the same span — the offset path reads the geom.Curve2 directly (ADR-0055, no projectedShape).
func TestProjectedArcOffsetsAnalytic(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc, _ := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 5, 0, stdmath.Pi/2)
	pc := s.ProjectCurve(&analyticEdge{id: "a5", curve: arc})

	got, err := s.OffsetEntity(pc, -1) // inner offset → radius 4
	if err != nil {
		t.Fatalf("offset projected arc: %v", err)
	}
	a, ok := got.(*Arc)
	if !ok {
		t.Fatalf("offset of a projected arc = %T, want a real *Arc (not a faceted chain)", got)
	}
	if r := float64(a.Radius()); stdmath.Abs(r-4) > 1e-9 {
		t.Fatalf("offset arc radius = %g, want 4 (5 + -1)", r)
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
