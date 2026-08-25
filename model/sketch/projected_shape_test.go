// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// sampleArc3 returns n+1 points along a circular arc — the kind of coarse polyline the projection
// samples an arc edge into (a coplanar/parallel projection lands them exactly on the circle).
func sampleArcPts(center gmath.Point2, r, start, sweep float64, n int) []gmath.Point2 {
	pts := make([]gmath.Point2, n+1)
	for i := range pts {
		a := start + sweep*float64(i)/float64(n)
		pts[i] = gmath.P2(center.X+gmath.Scalar(r*stdmath.Cos(a)), center.Y+gmath.Scalar(r*stdmath.Sin(a)))
	}
	return pts
}

func TestFitProjectedShapeLine(t *testing.T) {
	pts := []gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 2), gmath.P2(3, 3)}
	if s := fitProjectedShape(pts); s.kind != shapeLine {
		t.Fatalf("collinear points → kind %d, want shapeLine", s.kind)
	}
}

func TestFitProjectedShapeArc(t *testing.T) {
	pts := sampleArcPts(gmath.P2(1, 2), 3, 0, stdmath.Pi/2, 16) // quarter arc, 16 facets
	s := fitProjectedShape(pts)
	if s.kind != shapeArc {
		t.Fatalf("arc-sampled points → kind %d, want shapeArc", s.kind)
	}
	if stdmath.Abs(s.radius-3) > 1e-6 || !s.center.IsEqualTo(gmath.P2(1, 2), 1e-6) {
		t.Errorf("arc fit center=%v radius=%.4f, want (1,2) r=3", s.center, s.radius)
	}
	if s.sweep <= 0 { // CCW quarter
		t.Errorf("arc sweep=%.4f, want a positive (CCW) quarter", s.sweep)
	}
}

func TestFitProjectedShapeCircle(t *testing.T) {
	pts := sampleArcPts(gmath.P2(0, 0), 2, 0, 2*stdmath.Pi, 24) // closed
	if s := fitProjectedShape(pts); s.kind != shapeCircle {
		t.Fatalf("closed circle-sampled points → kind %d, want shapeCircle", s.kind)
	}
}

func TestFitProjectedShapeEllipseFallsBackToPolyline(t *testing.T) {
	// an oblique arc projects to an ellipse: squash the y of a circle so points are NOT on a circle
	pts := sampleArcPts(gmath.P2(0, 0), 3, 0, stdmath.Pi, 16)
	for i := range pts {
		pts[i] = gmath.P2(pts[i].X, pts[i].Y*0.5)
	}
	if s := fitProjectedShape(pts); s.kind != shapeNone {
		t.Fatalf("ellipse points → kind %d, want shapeNone (polyline fallback)", s.kind)
	}
}

func TestOffsetProjectedArcMakesAnArc(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pts := sampleArcPts(gmath.P2(0, 0), 5, 0, stdmath.Pi/2, 16)
	pc := s.RestoreProjectedCurve(nextID(), pts, "edge", "E1")
	if pc.shape.kind != shapeArc {
		t.Fatalf("restored projected arc shape = %d, want shapeArc", pc.shape.kind)
	}
	arcsBefore := s.Arcs().Count()
	got, err := s.OffsetEntity(pc, -1) // inner offset → radius 4
	if err != nil {
		t.Fatalf("OffsetEntity(projected arc): %v", err)
	}
	if _, ok := got.(*Arc); !ok {
		t.Fatalf("offset of a projected arc = %T, want a real *Arc (not a polyline)", got)
	}
	if s.Arcs().Count() != arcsBefore+1 {
		t.Errorf("offset created %d arcs, want 1", s.Arcs().Count()-arcsBefore)
	}
	if r := float64(s.Arcs().Item(arcsBefore).Radius()); stdmath.Abs(r-4) > 1e-6 {
		t.Errorf("offset arc radius = %.4f, want 4 (5 + -1)", r)
	}
}

func TestOffsetProjectedCircleMakesACircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pts := sampleArcPts(gmath.P2(0, 0), 5, 0, 2*stdmath.Pi, 24)
	pc := s.RestoreProjectedCurve(nextID(), pts, "edge", "E1")
	got, err := s.OffsetEntity(pc, -1)
	if err != nil {
		t.Fatalf("OffsetEntity(projected circle): %v", err)
	}
	c, ok := got.(*Circle)
	if !ok {
		t.Fatalf("offset of a projected circle = %T, want a real *Circle", got)
	}
	if stdmath.Abs(float64(c.Radius)-4) > 1e-6 {
		t.Errorf("offset circle radius = %.4f, want 4", float64(c.Radius))
	}
}
