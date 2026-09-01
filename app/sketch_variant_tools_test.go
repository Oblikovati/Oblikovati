// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// Three Point Rectangle: a base edge then a width, producing four lines (non-axis-aligned).
func TestThreePointRectangleTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewThreePointRectangleTool())
	s.Click(60, 100)  // first corner
	s.Click(140, 100) // end of the base edge
	s.Click(140, 140) // width — auto-commits on the third click
	if s.ActiveTool() != nil {
		t.Fatal("three-point rectangle should deactivate after three clicks")
	}
	if sk.Lines().Count() != 4 {
		t.Fatalf("three-point rectangle made %d lines, want 4", sk.Lines().Count())
	}
}

// Two Point Center Rectangle: a center then a corner, producing four edges plus the two
// construction diagonals that pin the centre (#2014).
func TestCenterRectangleTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewCenterRectangleTool())
	s.Click(100, 100) // center
	s.Click(140, 140) // corner — auto-commits on the second click
	if s.ActiveTool() != nil {
		t.Fatal("center rectangle should deactivate after two clicks")
	}
	if sk.Lines().Count() != 6 {
		t.Fatalf("center rectangle made %d lines, want 6 (4 edges + 2 construction diagonals)", sk.Lines().Count())
	}
	edges, diagonals := 0, 0
	for i := 0; i < sk.Lines().Count(); i++ {
		if sk.Lines().Item(i).IsConstruction() {
			diagonals++
			continue
		}
		edges++
	}
	if edges != 4 || diagonals != 2 {
		t.Errorf("edges = %d, construction diagonals = %d, want 4 and 2", edges, diagonals)
	}
}

// Three Point Circle: the circle through three clicked points.
func TestThreePointCircleTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewThreePointCircleTool())
	s.Click(60, 100)
	s.Click(140, 100)
	s.Click(100, 60) // auto-commits on the third click
	if sk.Circles().Count() != 1 {
		t.Fatalf("three-point circle added %d circles, want 1", sk.Circles().Count())
	}
	if r := sk.Circles().Item(0).Radius; r <= 0 {
		t.Errorf("circle radius = %v, want > 0", r)
	}
}

// Three Point Circle rejects three collinear points (no circumcircle).
func TestThreePointCircleToolRejectsCollinear(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewThreePointCircleTool())
	s.Click(40, 100)
	s.Click(100, 100)
	s.Click(160, 100)
	if err := s.OK(); err == nil {
		t.Error("collinear three-point circle should fail to commit")
	}
	if sk.Circles().Count() != 0 {
		t.Errorf("collinear circle added %d circles, want 0", sk.Circles().Count())
	}
}

// Center Point Arc: center, start, end produces one arc.
func TestCenterPointArcTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewCenterPointArcTool())
	s.Click(100, 100) // center
	s.Click(140, 100) // start
	s.Click(100, 140) // end — auto-commits on the third click
	if sk.Arcs().Count() != 1 {
		t.Fatalf("center-point arc added %d arcs, want 1", sk.Arcs().Count())
	}
}

// Center Point Arc Slot: center, start, end produces the slot's lines and arc caps.
func TestCenterPointArcSlotTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewCenterPointArcSlotTool(1))
	s.Click(100, 100)
	s.Click(140, 100)
	s.Click(100, 140) // auto-commits on the third click
	if s.ActiveTool() != nil {
		t.Fatal("center-point arc slot should deactivate after three clicks")
	}
	if sk.Arcs().Count() < 2 {
		t.Fatalf("arc slot made %d arcs, want >=2 (the two caps plus the slot arcs)", sk.Arcs().Count())
	}
}

// Three Point Arc Slot: start, a point on the arc, end produces the slot.
func TestThreePointArcSlotTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewThreePointArcSlotTool(1))
	s.Click(60, 100)
	s.Click(100, 60)
	s.Click(140, 100) // auto-commits on the third click
	if sk.Arcs().Count() < 2 {
		t.Fatalf("three-point arc slot made %d arcs, want >=2", sk.Arcs().Count())
	}
}

// Three Point Arc Slot rejects three collinear centre points.
func TestThreePointArcSlotToolRejectsCollinear(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewThreePointArcSlotTool(1))
	s.Click(40, 100)
	s.Click(100, 100)
	s.Click(160, 100)
	if err := s.OK(); err == nil {
		t.Error("collinear three-point arc slot should fail to commit")
	}
	if sk.Arcs().Count() != 0 {
		t.Errorf("collinear arc slot added %d arcs, want 0", sk.Arcs().Count())
	}
}

// Control Vertex Spline: the clicked points are control vertices; OK finishes the curve.
func TestControlVertexSplineTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewControlVertexSplineTool())
	s.Click(60, 100)
	s.Click(100, 120)
	s.Click(140, 100)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if sk.Splines().Count() != 1 {
		t.Fatalf("control-vertex spline added %d splines, want 1", sk.Splines().Count())
	}
}
