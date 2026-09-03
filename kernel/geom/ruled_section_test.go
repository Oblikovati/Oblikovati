// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func testCylinderY(t *testing.T) Cylinder {
	t.Helper()
	cyl, err := NewCylinder(math.P3(0, -2, 1), math.V3(0, 1, 0), 0.1)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	return cyl
}

// tiltedSection is the ellipse a plane tilted `deg` off the cylinder's cross-section cuts at axial
// station y0 — the oblique rim of the multipoint-disk pin (#3459).
func tiltedSection(t *testing.T, cyl Cylinder, y0, deg float64) EllipseFull {
	t.Helper()
	a := deg * stdmath.Pi / 180
	pl, err := NewPlane(math.P3(0, y0, 1), math.V3(stdmath.Sin(a), stdmath.Cos(a), 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	curves, handled := IntersectSurfacesAnalytic(pl, cyl, ResolutionForSize(1))
	if !handled || len(curves) != 1 {
		t.Fatalf("plane∩cylinder: handled=%v n=%d", handled, len(curves))
	}
	e, ok := curves[0].(EllipseFull)
	if !ok {
		t.Fatalf("section is %T, want EllipseFull", curves[0])
	}
	return e
}

func TestRuledFrameOfCylinderAndCone(t *testing.T) {
	cyl := testCylinderY(t)
	fr, ok := RuledFrameOf(cyl)
	if !ok || fr.RadSlope != 0 || fr.RadConst != 0.1 {
		t.Fatalf("cylinder frame = %+v ok=%v", fr, ok)
	}
	if p := fr.Ruling(0.7).PointAt(0.3); p.DistanceTo(cyl.PointAt(0.7, 0.3)) > 1e-12 {
		t.Errorf("ruling point %v off the cylinder point %v", p, cyl.PointAt(0.7, 0.3))
	}
	cone, err := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6)
	if err != nil {
		t.Fatalf("NewCone: %v", err)
	}
	fr, ok = RuledFrameOf(cone)
	if !ok || stdmath.Abs(fr.RadSlope-stdmath.Tan(stdmath.Pi/6)) > 1e-15 || fr.RadConst != 0 {
		t.Fatalf("cone frame = %+v ok=%v", fr, ok)
	}
	if p := fr.Ruling(1.1).PointAt(2); stdmath.Abs(fr.Radius(float64(fr.Base.VectorTo(p).Dot(fr.Axis)))-stdmath.Hypot(float64(p.X), float64(p.Y))) > 1e-12 {
		t.Errorf("cone ruling leaves the cone at %v", p)
	}
	if _, ok := RuledFrameOf(Plane{}); ok {
		t.Errorf("a plane is not a ruled frame")
	}
}

func TestSectionPlaneOfArcsAndConics(t *testing.T) {
	cyl := testCylinderY(t)
	e := tiltedSection(t, cyl, -1.5, 7.5)
	arc, err := NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius, -stdmath.Pi/2, stdmath.Pi)
	if err != nil {
		t.Fatalf("NewEllipticalArc: %v", err)
	}
	for _, c := range []Curve3{e, arc} {
		pl, ok := SectionPlane(c)
		if !ok {
			t.Fatalf("SectionPlane(%T) declined", c)
		}
		for _, tp := range []float64{0, 0.3, 0.77} {
			if d := stdmath.Abs(float64(pl.Normal().Dot(pl.Origin.VectorTo(c.PointAt(tp))))); d > 1e-14 {
				t.Errorf("%T point at %g is %g off its section plane", c, tp, d)
			}
		}
	}
	if _, ok := SectionPlane(NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))); ok {
		t.Errorf("a straight curve has no single section plane")
	}
}

func TestCurveParamAtInvertsEveryFrameEdgeKind(t *testing.T) {
	cyl := testCylinderY(t)
	e := tiltedSection(t, cyl, -1.5, 7.5)
	arc, _ := NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius, -stdmath.Pi/2, stdmath.Pi)
	circ, _ := NewArc3d(math.P3(0, -0.4, 1), math.V3(0, 1, 0), math.V3(1, 0, 0), 0.1, 2.5, 2.0)
	seg := NewLineSegment(math.P3(0.1, -1, 1), math.P3(0.1, -0.5, 1))
	for _, c := range []Curve3{e, arc, circ, seg} {
		for _, want := range []float64{0, 0.25, 0.6, 0.999} {
			got, ok := CurveParamAt(c, c.PointAt(want))
			if !ok || stdmath.Abs(got-want) > 1e-12 {
				t.Errorf("%T: CurveParamAt(PointAt(%g)) = %g ok=%v", c, want, got, ok)
			}
		}
	}
}

func TestSectionCrossingCandidatesOnCylinder(t *testing.T) {
	cyl := testCylinderY(t)
	rim := tiltedSection(t, cyl, -1.5, 7.5)
	tool := tiltedSection(t, cyl, -1.5, -30) // a second oblique section through the same station crosses the rim twice
	pts, coincident := SectionCrossingCandidates(cyl, rim, tool)
	if coincident || len(pts) != 2 {
		t.Fatalf("two crossing sections: %d points coincident=%v", len(pts), coincident)
	}
	for _, p := range pts {
		if _, tr := CurveParamAt(rim, p); !tr {
			t.Errorf("crossing %v not on the rim", p)
		}
		if d := stdmath.Hypot(float64(p.X), float64(p.Z)-1) - 0.1; stdmath.Abs(d) > 1e-13 {
			t.Errorf("crossing %v is %g off the cylinder", p, d)
		}
	}
	ruling := NewLineSegment(cyl.PointAt(1.2, -0.5), cyl.PointAt(1.2, 0.5))
	pts, _ = SectionCrossingCandidates(cyl, ruling, rim)
	if len(pts) != 1 {
		t.Fatalf("ruling × section: %d points, want 1", len(pts))
	}
	if d := stdmath.Abs(float64(rim.Normal.AsVector().Dot(rim.Center.VectorTo(pts[0])))); d > 1e-14 {
		t.Errorf("ruling pierce %v is %g off the section plane", pts[0], d)
	}
	same := tiltedSection(t, cyl, -1.5, 7.5)
	if _, coincident := SectionCrossingCandidates(cyl, rim, same); !coincident {
		t.Errorf("the same section twice must report coincident")
	}
	if pts, _ := SectionCrossingCandidates(cyl, ruling, NewLineSegment(cyl.PointAt(2, 0), cyl.PointAt(2, 1))); len(pts) != 0 {
		t.Errorf("two rulings never cross, got %d", len(pts))
	}
}

func TestAxialExtentOfTiltedRim(t *testing.T) {
	cyl := testCylinderY(t)
	rim := tiltedSection(t, cyl, -1.5, 7.5)
	axis := math.V3(0, 1, 0)
	lo, hi, ok := AxialExtent(rim, 0, 1, cyl.Origin, axis)
	want := 0.1 * stdmath.Tan(7.5*stdmath.Pi/180) // the rim rises ±r·tanα about its station
	if !ok || stdmath.Abs((hi-lo)/2-want) > 1e-14 || stdmath.Abs((hi+lo)/2-0.5) > 1e-14 {
		t.Errorf("tilted rim extent = [%g, %g] ok=%v, want ±%g about 0.5", lo, hi, ok, want)
	}
	// A quarter arc that starts at the station and climbs to the top holds no interior stationary point.
	arc, _ := NewEllipticalArc(rim.Center, rim.Normal.AsVector(), rim.MajorAxis.AsVector(), rim.MajorRadius, rim.MinorRadius, 0, stdmath.Pi/2)
	alo, ahi, _ := AxialExtent(arc, 0, 1, cyl.Origin, axis)
	v0 := float64(cyl.Origin.VectorTo(arc.PointAt(0)).Dot(axis))
	v1 := float64(cyl.Origin.VectorTo(arc.PointAt(1)).Dot(axis))
	if stdmath.Abs(alo-stdmath.Min(v0, v1)) > 1e-14 || stdmath.Abs(ahi-stdmath.Max(v0, v1)) > 1e-14 {
		t.Errorf("quarter-arc extent = [%g, %g], want its endpoints [%g, %g]", alo, ahi, stdmath.Min(v0, v1), stdmath.Max(v0, v1))
	}
	seg := NewLineSegment(math.P3(0.1, -1, 1), math.P3(0.1, -0.5, 1))
	if slo, shi, ok := AxialExtent(seg, 0, 1, cyl.Origin, axis); !ok || slo != 1 || shi != 1.5 {
		t.Errorf("segment extent = [%g, %g] ok=%v", slo, shi, ok)
	}
}
