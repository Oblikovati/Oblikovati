// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
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

// kindedVertex/kindedEdge are kinded fake sources, so the projection records a source kind for
// persistence (the unkinded movableVertex/Edge record "").
type kindedVertex struct{ movableVertex }

func (kindedVertex) SourceKind() string { return "vertex" }

type kindedEdge struct{ movableEdge }

func (kindedEdge) SourceKind() string { return "edge" }

// TestProjectedSourceDescriptorAndRebind covers the persistence seam: a projection records its
// source (kind, id); a restored-frozen projection keeps the descriptor and re-links + re-projects
// once a live source is rebound (#1268).
func TestProjectedSourceDescriptorAndRebind(t *testing.T) {
	live := NewSketches().Add(XYPlane())
	pp := live.ProjectPoint(&kindedVertex{movableVertex{id: "v1", pos: math.P3(1, 2, 0)}})
	if k, id := pp.SourceDescriptor(); k != "vertex" || id != "v1" {
		t.Errorf("point descriptor = (%q,%q), want (vertex,v1)", k, id)
	}
	pc := live.ProjectCurve(&kindedEdge{movableEdge{id: "e1", samples: []math.Point3{{X: 0}, {X: 1}}}})
	if k, id := pc.SourceDescriptor(); k != "edge" || id != "e1" {
		t.Errorf("curve descriptor = (%q,%q), want (edge,e1)", k, id)
	}

	s := NewSketches().Add(XYPlane())
	rp := s.RestoreProjectedPoint(ID(7), math.P2(1, 2), "vertex", "v1")
	if rp.Linked() {
		t.Error("a restored projection starts frozen (no live source yet)")
	}
	rp.Rebind(&movableVertex{id: "v1", pos: math.P3(5, 6, 0)})
	rp.Update()
	if !rp.Linked() || !rp.Position().IsEqualTo(math.P2(5, 6), tol) {
		t.Errorf("after rebind+update the point should track its source, got %v linked=%v", rp.Position(), rp.Linked())
	}

	rc := s.RestoreProjectedCurve(ID(8), []math.Point2{{X: 0}, {X: 1}}, "edge", "e1")
	if k, id := rc.SourceDescriptor(); k != "edge" || id != "e1" {
		t.Errorf("restored curve descriptor = (%q,%q), want (edge,e1)", k, id)
	}
	rc.Rebind(&movableEdge{id: "e1", samples: []math.Point3{{X: 0}, {X: 2}, {X: 4}}})
	rc.Update()
	if !rc.Linked() || len(rc.Points()) != 3 {
		t.Errorf("after rebind the curve should re-project, got %d points linked=%v", len(rc.Points()), rc.Linked())
	}
}

// TestProjectedAnchorIsPickable proves the projected anchor appears in AllPoints — the
// pick/snap/selection candidate set — so a coincident constraint can actually be picked to it
// (#1268; the anchor was previously omitted, so the user's click found nothing and no
// constraint was ever created).
func TestProjectedAnchorIsPickable(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pp := s.ProjectPoint(&movableVertex{id: "v1", pos: math.P3(4, 2, 0)})
	free := s.Points().Add(math.P2(0, 0))

	pts := s.AllPoints()
	if !containsPoint(pts, pp.Anchor()) {
		t.Error("AllPoints must include the projected anchor so it can be picked/snapped")
	}
	if !containsPoint(pts, free) {
		t.Error("AllPoints must still include free points")
	}
}

func containsPoint(pts []*Point, p *Point) bool {
	for _, q := range pts {
		if q == p {
			return true
		}
	}
	return false
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

// TestProjectedGeometrySerializeRoundTrip exercises the model-side serialize/restore of projected
// geometry (the codec that used to be missing, #1268): a projected point and curve survive a
// MarshalRecipe→ApplyRecipe, restoring frozen with their anchor id, geometry and source
// descriptor preserved (the host rebinds the live source separately).
func TestProjectedGeometrySerializeRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	pp := s.ProjectPoint(&kindedVertex{movableVertex{id: "v9", pos: math.P3(2, 3, 0)}})
	anchorID := pp.EntityID()
	s.ProjectCurve(&kindedEdge{movableEdge{id: "e9", samples: []math.Point3{{X: 0}, {X: 1, Y: 1}}}})

	out := roundTrip(t, sc)
	var rp *ProjectedPoint
	var rc *ProjectedCurve
	for _, e := range out.Entities() {
		switch v := e.(type) {
		case *ProjectedPoint:
			rp = v
		case *ProjectedCurve:
			rc = v
		}
	}
	if rp == nil || rc == nil {
		t.Fatalf("projected geometry lost on round trip: point=%v curve=%v", rp, rc)
	}
	if rp.EntityID() != anchorID {
		t.Errorf("restored anchor id = %d, want %d (constraints reference it)", rp.EntityID(), anchorID)
	}
	if !rp.Position().IsEqualTo(math.P2(2, 3), tol) {
		t.Errorf("restored projected point at %v, want frozen (2,3)", rp.Position())
	}
	if k, id := rp.SourceDescriptor(); k != "vertex" || id != "v9" {
		t.Errorf("restored point descriptor = (%q,%q), want (vertex,v9)", k, id)
	}
	if rp.Linked() {
		t.Error("restored projection should be frozen until the host rebinds it")
	}
	if k, id := rc.SourceDescriptor(); k != "edge" || id != "e9" {
		t.Errorf("restored curve descriptor = (%q,%q), want (edge,e9)", k, id)
	}
	if len(rc.Points()) != 2 {
		t.Errorf("restored curve has %d points, want 2", len(rc.Points()))
	}
}
