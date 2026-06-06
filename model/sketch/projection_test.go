// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati/math"
)

// movableVertex is a fake model vertex whose position can change (and can be "lost"),
// standing in for a topo vertex (M07).
type movableVertex struct {
	id   string
	pos  math.Point3
	lost bool
}

func (v *movableVertex) SourceID() string { return v.id }
func (v *movableVertex) Position() (math.Point3, bool) {
	if v.lost {
		return math.Point3{}, false
	}
	return v.pos, true
}

// movableEdge is a fake model edge yielding a sampled polyline (and can be "lost").
type movableEdge struct {
	id      string
	samples []math.Point3
	lost    bool
}

func (e *movableEdge) SourceID() string { return e.id }
func (e *movableEdge) SamplePoints() ([]math.Point3, bool) {
	if e.lost {
		return nil, false
	}
	return e.samples, true
}

// TestProjectedPointIsConstrainableAnchor proves a 2D sketch can be built on a projected
// point: a free point constrained coincident with the projected anchor snaps to it on
// solve, while the anchor stays fixed (it is not a free DOF).
func TestProjectedPointIsConstrainableAnchor(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pp := s.ProjectPoint(&movableVertex{id: "v1", pos: math.P3(4, 2, 0)}) // anchor at (4,2)
	free := s.Points().Add(math.P2(0, 0))

	if dof := s.DegreesOfFreedom(); dof != 2 {
		t.Fatalf("free DOF = %d, want 2 (the projected anchor is fixed)", dof)
	}
	s.GeometricConstraints().AddCoincident(free, pp.Anchor())
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve: %+v", res)
	}
	if !free.Position().IsEqualTo(math.P2(4, 2), tol) {
		t.Errorf("free point = %v, want snapped to the projected anchor (4,2)", free.Position())
	}
	if !pp.Position().IsEqualTo(math.P2(4, 2), tol) {
		t.Errorf("projected anchor should stay fixed, got %v", pp.Position())
	}
	if p, ok := s.PointByID(pp.EntityID()); !ok || p != pp.Anchor() {
		t.Error("PointByID should resolve the projected anchor by the entity id")
	}
}

// TestProjectedLostReferenceFreezes checks UpdateProjections breaks the link and freezes
// geometry when a projected source's reference is lost.
func TestProjectedLostReferenceFreezes(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	v := &movableVertex{id: "v1", pos: math.P3(2, 3, 5)}
	pp := s.ProjectPoint(v)
	e := &movableEdge{id: "e1", samples: []math.Point3{{X: 0}, {X: 1}}}
	pc := s.ProjectCurve(e)

	v.lost, e.lost = true, true
	s.UpdateProjections()
	if pp.Linked() || pc.Linked() {
		t.Error("lost references should break both projection links")
	}
	if !pp.Position().IsEqualTo(math.P2(2, 3), tol) || len(pc.Points()) != 2 {
		t.Error("lost references should freeze the last projected geometry")
	}
}

func TestProjectedPointUpdatesWhenSourceChanges(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	v := &movableVertex{id: "v1", pos: math.P3(2, 3, 5)}
	pp := s.ProjectPoint(v)

	if !pp.Position().IsEqualTo(math.P2(2, 3), tol) { // z dropped by XY projection
		t.Fatalf("initial projection = %v, want (2,3)", pp.Position())
	}
	if !pp.IsReference() || !pp.Linked() || pp.SourceID() != "v1" {
		t.Error("projected point should be linked reference geometry")
	}
	if s.EntityCount() != 1 {
		t.Error("projected point not added to the sketch")
	}
	if pp.EntityID() == 0 || s.Entities()[0].EntityID() != pp.EntityID() {
		t.Error("projected point entity id not registered")
	}

	// The source edge/vertex moves; re-projection tracks it.
	v.pos = math.P3(10, -4, 1)
	pp.Update()
	if !pp.Position().IsEqualTo(math.P2(10, -4), tol) {
		t.Errorf("after source move, projection = %v, want (10,-4)", pp.Position())
	}
}

func TestBreakLinkFreezesProjection(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	v := &movableVertex{id: "v1", pos: math.P3(1, 1, 0)}
	pp := s.ProjectPoint(v)

	pp.BreakLink()
	v.pos = math.P3(9, 9, 0)
	pp.Update() // no-op after break
	if !pp.Position().IsEqualTo(math.P2(1, 1), tol) {
		t.Errorf("broken-link projection moved to %v, want frozen (1,1)", pp.Position())
	}
	if pp.Linked() || pp.SourceID() != "" {
		t.Error("broken link still reports linked/source")
	}
}

func TestProjectCutEdgesProjectsEachSource(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	e := &movableEdge{id: "e1", samples: []math.Point3{math.P3(0, 0, 7), math.P3(2, 0, 7), math.P3(2, 2, 7)}}
	curves := s.ProjectCutEdges([]CurveSource{e})
	if len(curves) != 1 {
		t.Fatalf("ProjectCutEdges returned %d curves, want 1", len(curves))
	}
	pts := curves[0].Points()
	if len(pts) != 3 || !pts[2].IsEqualTo(math.P2(2, 2), tol) {
		t.Errorf("projected polyline = %v, want last point (2,2)", pts)
	}
	if curves[0].SourceID() != "e1" || !curves[0].IsReference() || curves[0].EntityID() == 0 {
		t.Error("projected curve metadata wrong")
	}

	// Source edge changes shape → re-projection follows.
	e.samples = []math.Point3{math.P3(0, 0, 7), math.P3(5, 0, 7)}
	curves[0].Update()
	if len(curves[0].Points()) != 2 {
		t.Errorf("after source change, polyline has %d points, want 2", len(curves[0].Points()))
	}

	curves[0].BreakLink()
	if curves[0].Linked() || curves[0].SourceID() != "" {
		t.Error("broken curve link still reports linked/source")
	}
}
