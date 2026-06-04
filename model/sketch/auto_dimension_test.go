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
