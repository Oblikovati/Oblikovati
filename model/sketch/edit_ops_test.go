// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

func TestMoveEntitiesTranslatesSharedPointsOnce(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	lines := s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	s.MoveEntities(lines, gmath.V2(10, 5))
	// Every vertex shifted by (10,5); the lower-left corner is now (10,5).
	min := gmath.P2(1e18, 1e18)
	for _, p := range s.AllPoints() {
		if p.X < min.X {
			min.X = p.X
		}
		if p.Y < min.Y {
			min.Y = p.Y
		}
	}
	if math.Abs(float64(min.X)-10) > 1e-9 || math.Abs(float64(min.Y)-5) > 1e-9 {
		t.Fatalf("min corner after move = %v, want (10,5)", min)
	}
}

func TestRotateEntitiesNinetyDegrees(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(1, 0), gmath.P2(2, 0))
	s.RotateEntities([]Entity{l}, gmath.P2(0, 0), math.Pi/2)
	// (1,0)→(0,1) and (2,0)→(0,2) under a +90° rotation about the origin.
	if p := l.A.Position(); math.Abs(float64(p.X)) > 1e-9 || math.Abs(float64(p.Y)-1) > 1e-9 {
		t.Errorf("A after rotate = %v, want (0,1)", p)
	}
	if p := l.B.Position(); math.Abs(float64(p.X)) > 1e-9 || math.Abs(float64(p.Y)-2) > 1e-9 {
		t.Errorf("B after rotate = %v, want (0,2)", p)
	}
}

func TestCopyEntitiesLeavesOriginalAndAddsCopy(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(0, 0), 1)
	copies := s.CopyEntities([]Entity{c}, gmath.V2(5, 0))
	if len(copies) != 1 {
		t.Fatalf("copies = %d, want 1", len(copies))
	}
	if s.Circles().Count() != 2 {
		t.Fatalf("circle count = %d, want 2 (original + copy)", s.Circles().Count())
	}
	cp := copies[0].(*Circle)
	if math.Abs(float64(cp.Center.X)-5) > 1e-9 {
		t.Errorf("copy center X = %v, want 5", cp.Center.X)
	}
	// Original untouched.
	if math.Abs(float64(c.Center.X)) > 1e-9 {
		t.Errorf("original moved to X=%v, want 0", c.Center.X)
	}
}

func TestCopyEntitiesClonesEveryEntityType(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents := []Entity{
		s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0)),
		s.Arcs().AddByCenterStartEnd(gmath.P2(0, 0), gmath.P2(1, 0), gmath.P2(0, 1), true),
		s.Ellipses().Add(gmath.P2(0, 0), gmath.V2(1, 0), 2, 1),
		s.EllipticalArcs().Add(gmath.P2(0, 0), gmath.V2(1, 0), 2, 1, 0, 1),
		s.Splines().AddByPoints([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)}, false),
		s.Points().Add(gmath.P2(5, 5)),
	}
	copies := s.CopyEntities(ents, gmath.V2(10, 0))
	if len(copies) != len(ents) {
		t.Fatalf("copied %d of %d entities", len(copies), len(ents))
	}
	// A rotated copy must rotate the ellipse's major axis (linear part applied).
	rc := s.cloneEntities([]Entity{ents[2]}, rotation(gmath.P2(0, 0), math.Pi/2))
	ax := rc[0].(*Ellipse).MajorAxis
	if math.Abs(float64(ax.X)) > 1e-9 || math.Abs(float64(ax.Y)-1) > 1e-9 {
		t.Errorf("rotated ellipse axis = %v, want ~(0,1)", ax)
	}
}

func TestMirrorEntitiesReflectsAcrossLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	mirror := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(0, 1)) // the Y axis
	dot := s.Points().Add(gmath.P2(3, 2))
	copies := s.MirrorEntities([]Entity{dot}, mirror)
	if len(copies) != 1 {
		t.Fatalf("mirror copies = %d, want 1", len(copies))
	}
	p := copies[0].(*Point).Position()
	if math.Abs(float64(p.X)+3) > 1e-9 || math.Abs(float64(p.Y)-2) > 1e-9 {
		t.Fatalf("mirrored point = %v, want (-3,2)", p)
	}
}
