// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

func TestAutoDimensionReachesZeroDOF(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3)) // 8 DOF
	if s.DegreesOfFreedom() == 0 {
		t.Fatal("rectangle should start under-constrained")
	}
	added := s.AutoDimension()
	if added == 0 {
		t.Fatal("AutoDimension added nothing")
	}
	if dof := s.DegreesOfFreedom(); dof != 0 {
		t.Fatalf("after AutoDimension DOF = %d, want 0", dof)
	}
}

func TestAutoDimensionIdempotentWhenFull(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	s.AutoDimension()
	if again := s.AutoDimension(); again != 0 {
		t.Fatalf("second AutoDimension added %d, want 0", again)
	}
}

// The improvement over grounding whole entities: AutoDimension leaves the sketch
// WELL-constrained (0 DOF, no redundancy), never over-constrained.
func TestAutoDimensionIsWellConstrainedNotRedundant(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	s.AutoDimension()
	a := s.AnalyzeConstraints()
	if a.DOF != 0 || a.Redundant != 0 || a.Status != WellConstrained {
		t.Fatalf("after AutoDimension: DOF=%d Redundant=%d Status=%v, want 0/0/WellConstrained", a.DOF, a.Redundant, a.Status)
	}
}

// AutoDimension prefers real, editable dimensions (not only grounds).
func TestAutoDimensionAddsRealDimensions(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	s.AutoDimension()
	if s.DimensionConstraints().Count() == 0 {
		t.Error("AutoDimension added no dimensions (should place length dims, not only grounds)")
	}
}

// A free circle (center + radius = 3 DOF) auto-dimensions to a well-constrained sketch
// including a radius dimension.
func TestAutoDimensionCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Circles().AddByCenterRadius(gmath.P2(1, 2), 2)
	s.AutoDimension()
	a := s.AnalyzeConstraints()
	if a.DOF != 0 || a.Redundant != 0 {
		t.Fatalf("circle AutoDimension: DOF=%d Redundant=%d, want 0/0", a.DOF, a.Redundant)
	}
	if s.DimensionConstraints().Count() == 0 {
		t.Error("expected a radius dimension on the circle")
	}
}

// Dimensions are placed at current values, so auto-dimensioning then solving does not move
// the geometry.
func TestAutoDimensionPreservesGeometry(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	s.AutoDimension()
	s.Solve()
	if !pointAt(s, gmath.P2(4, 3)) {
		t.Error("AutoDimension + Solve moved the geometry; (4,3) corner is gone")
	}
}

// pointAt reports whether any constrainable point sits at q.
func pointAt(s *Sketch, q gmath.Point2) bool {
	for _, p := range s.uniqueEntityPoints() {
		if math.Abs(float64(p.X)-float64(q.X)) < 1e-6 && math.Abs(float64(p.Y)-float64(q.Y)) < 1e-6 {
			return true
		}
	}
	return false
}

func TestOffsetChainStaysConnected(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// An L: (0,0)→(4,0)→(4,3), sharing the corner (4,0).
	corner := s.newPoint(gmath.P2(4, 0))
	l1 := s.Lines().Add(s.newPoint(gmath.P2(0, 0)), corner)
	l2 := s.Lines().Add(corner, s.newPoint(gmath.P2(4, 3)))
	off := s.OffsetChain([]*Line{l1, l2}, 1)
	if len(off) != 2 {
		t.Fatalf("offset chain = %d lines, want 2", len(off))
	}
	// The two offset lines share their mitred corner exactly.
	if off[0].B != off[1].A {
		t.Fatal("offset chain is not connected at the miter")
	}
	// Offsetting the horizontal then the vertical inward miters at (3,1).
	if p := off[0].B.Position(); math.Abs(float64(p.X)-3) > 1e-9 || math.Abs(float64(p.Y)-1) > 1e-9 {
		t.Fatalf("miter corner = %v, want (3,1)", p)
	}
}
