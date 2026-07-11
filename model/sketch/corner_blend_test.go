// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

// cornerLines builds two lines meeting at the origin: (0,0)→(4,0) and (0,0)→(0,4).
func cornerLines(s *Sketch) (*Line, *Line) {
	corner := s.newPoint(gmath.P2(0, 0))
	l1 := s.Lines().Add(corner, s.newPoint(gmath.P2(4, 0)))
	l2 := s.Lines().Add(corner, s.newPoint(gmath.P2(0, 4)))
	return l1, l2
}

func TestAddFilletInsertsTangentArc(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1, l2 := cornerLines(s)
	arc, err := s.AddFillet(l1, l2, 1)
	if err != nil {
		t.Fatalf("AddFillet: %v", err)
	}
	// The tangent arc of a right-angle corner with r=1 is centered at (1,1), radius 1.
	if c := arc.Center.Position(); math.Abs(float64(c.X)-1) > 1e-9 || math.Abs(float64(c.Y)-1) > 1e-9 {
		t.Errorf("arc center = %v, want (1,1)", c)
	}
	if r := float64(arc.Radius()); math.Abs(r-1) > 1e-9 {
		t.Errorf("arc radius = %v, want 1", r)
	}
	// l1 was trimmed back to its tangent point (1,0).
	if p := l1.A.Position(); math.Abs(float64(p.X)-1) > 1e-9 || math.Abs(float64(p.Y)) > 1e-9 {
		t.Errorf("l1 trimmed end = %v, want (1,0)", p)
	}
}

func TestAddChamferInsertsBevelLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1, l2 := cornerLines(s)
	bevel, err := s.AddChamfer(l1, l2, 1, 1)
	if err != nil {
		t.Fatalf("AddChamfer: %v", err)
	}
	// The bevel connects (1,0) and (0,1) ⇒ length √2.
	if got := float64(bevel.Length()); math.Abs(got-math.Sqrt2) > 1e-9 {
		t.Errorf("bevel length = %v, want √2", got)
	}
}

// TestFilletDropsOrphanCorner: the fillet consumes the shared corner vertex, so it no longer
// floats in the sketch as a free (+2 DOF) point (#69).
func TestFilletDropsOrphanCorner(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1, l2 := cornerLines(s)
	before := len(s.AllPoints())
	if _, err := s.AddFillet(l1, l2, 1); err != nil {
		t.Fatalf("AddFillet: %v", err)
	}
	// Net points: −1 corner (removed) +2 tangent +1 centre = +2.
	if got := len(s.AllPoints()); got != before+2 {
		t.Errorf("point count %d after fillet, want %d (orphan corner dropped, 2 tangents + centre added)", got, before+2)
	}
	for _, p := range s.AllPoints() {
		if p.Position().IsEqualTo(gmath.P2(0, 0), 1e-9) {
			t.Error("the orphaned corner vertex at (0,0) still lingers in the sketch")
		}
	}
}

// TestFilletKeepsCornerStillInUse: the fillet must NOT drop the shared corner when anything else
// still references it — a third edge, a geometric constraint, or a dimension. Only a truly
// orphaned vertex is removed (#69). Each keep-reason exercises a branch of pointReferenced.
func TestFilletKeepsCornerStillInUse(t *testing.T) {
	keepBy := map[string]func(s *Sketch, corner *Point){
		"third edge": func(s *Sketch, c *Point) { s.Lines().Add(c, s.newPoint(gmath.P2(-4, 0))) },
		"constraint": func(s *Sketch, c *Point) { s.GeometricConstraints().AddFix(c) },
		"dimension": func(s *Sketch, c *Point) {
			_, _ = s.DimensionConstraints().AddDistance(c, s.newPoint(gmath.P2(0, -3)), "3 cm")
		},
	}
	for reason, keep := range keepBy {
		s := NewSketches().Add(XYPlane())
		corner := s.newPoint(gmath.P2(0, 0))
		l1 := s.Lines().Add(corner, s.newPoint(gmath.P2(4, 0)))
		l2 := s.Lines().Add(corner, s.newPoint(gmath.P2(0, 4)))
		keep(s, corner)
		arc, err := s.AddFillet(l1, l2, 1)
		if err != nil {
			t.Fatalf("[%s] AddFillet: %v", reason, err)
		}
		found := false
		for _, p := range s.AllPoints() {
			if p == corner {
				found = true
			}
		}
		if !found {
			t.Errorf("[%s] corner still in use was wrongly removed", reason)
		}
		// The tangency records are structural: they cannot be deleted on their own.
		for _, ft := range arc.filletTangents {
			if ft.Deletable() {
				t.Errorf("[%s] fillet tangency constraint should be non-deletable", reason)
			}
		}
	}
}

// TestFilletedSlopedCornerReachesDOF0 is the #69 regression: a fillet on a SLOPED corner (a
// tapered flange's root) reaches DOF 0 with only the caller's edge directions + a radius
// dimension, because AddFillet now drops the orphan corner and pins tangency non-degenerately.
func TestFilletedSlopedCornerReachesDOF0(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()

	// Fixed 8% reference so the sloped edge's direction is pinned independent of its endpoints.
	rA, rB := s.Points().Add(gmath.P2(-5, 0)), s.Points().Add(gmath.P2(5, 0.8))
	ref := s.Lines().Add(rA, rB)
	g.AddFix(rA)
	g.AddFix(rB)

	vFar := s.Points().Add(gmath.P2(0, 0))    // web-front bottom
	corner := s.Points().Add(gmath.P2(0, 5))  // root corner
	sFar := s.Points().Add(gmath.P2(8, 5.64)) // flange toe, on the 8% slope through the corner
	v := s.Lines().Add(vFar, corner)
	sLine := s.Lines().Add(corner, sFar)

	arc, err := s.AddFillet(v, sLine, 1)
	if err != nil {
		t.Fatalf("AddFillet: %v", err)
	}
	g.AddFix(vFar)
	g.AddFix(sFar)
	g.AddVertical(vFar, v.B)  // v trimmed to vFar->t1
	g.AddParallel(sLine, ref) // sLine trimmed to t2->sFar, pinned to the 8% slope
	if _, err := s.DimensionConstraints().AddRadius(arc, "1 cm"); err != nil {
		t.Fatalf("AddRadius: %v", err)
	}

	if dof := s.DegreesOfFreedom(); dof != 0 {
		t.Errorf("filleted sloped corner DOF = %d, want 0", dof)
	}
	r := s.Solve()
	if !r.Converged {
		t.Fatalf("filleted sloped corner did not converge (residual %g)", r.Residual)
	}
	// Tangent to both edges at unit radius, and the flange slope held at 8%.
	if got := float64(arc.Radius()); math.Abs(got-1) > 1e-6 {
		t.Errorf("arc radius %v, want 1", got)
	}
	if dv := perpFromCenter(arc, v); math.Abs(dv-1) > 1e-6 {
		t.Errorf("arc not tangent to the vertical edge (perp dist %v)", dv)
	}
	if ds := perpFromCenter(arc, sLine); math.Abs(ds-1) > 1e-6 {
		t.Errorf("arc not tangent to the sloped edge (perp dist %v)", ds)
	}
	d := sLine.A.Position().VectorTo(sLine.B.Position())
	if slope := float64(d.Y) / float64(d.X); math.Abs(slope-0.08) > 1e-6 {
		t.Errorf("flange slope %v, want 0.08", slope)
	}
}

// perpFromCenter is the perpendicular distance from an arc's centre to a line's supporting line.
func perpFromCenter(arc *Arc, l *Line) float64 {
	dir := l.A.Position().VectorTo(l.B.Position())
	length := dir.Length()
	if length == 0 {
		return 0
	}
	return math.Abs(float64(dir.Cross(l.A.Position().VectorTo(arc.Center.Position())))) / float64(length)
}

func TestAddFilletRejectsParallelLines(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	shared := s.newPoint(gmath.P2(0, 0))
	l1 := s.Lines().Add(shared, s.newPoint(gmath.P2(4, 0)))
	l2 := s.Lines().Add(shared, s.newPoint(gmath.P2(-4, 0))) // anti-parallel through the corner
	if _, err := s.AddFillet(l1, l2, 1); err == nil {
		t.Error("fillet of (anti)parallel lines should error")
	}
}

func TestAddFilletRejectsDisjointLines(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	l2 := s.Lines().AddByTwoPoints(gmath.P2(5, 5), gmath.P2(6, 6))
	if _, err := s.AddFillet(l1, l2, 1); err == nil {
		t.Error("fillet of lines with no shared corner should error")
	}
}
