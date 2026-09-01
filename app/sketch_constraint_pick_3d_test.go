// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// topDown800 is a camera looking straight down the −Z axis, so the centre pixel (400,400) casts a
// ray through the world origin — a marker anchored there is picked by a click at the centre.
func topDown800(s *Session) {
	cam := scene.NewCamera(800, 800)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
}

// TestSelectAndDeleteSketch3DConstraint is the #1998 pick/delete follow-up: clicking a 3D
// constraint marker selects it (and it highlights), and Delete removes the relation from the 3D
// sketch. A coincidence of two points at the origin puts its marker on the centre-pixel ray.
func TestSelectAndDeleteSketch3DConstraint(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	topDown800(s)
	l1 := sk.AddLine3D(math.P3(-1, 0, 0), math.P3(0, 0, 0))
	l2 := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	c := sketch.NewCoincident3D(l1.B, l2.A) // marker at (0,0,0), on the centre ray
	sk.GeometricConstraints3D().Add(c)
	s.SetShowSketchConstraints(true)

	if !s.SelectSketchConstraint3DAt(400, 400, 0) {
		t.Fatal("clicking the centre pixel over the (0,0,0) marker selected nothing")
	}
	got := s.SelectedSketchConstraints()
	if len(got) != 1 || got[0] != c {
		t.Fatalf("selection = %v, want the coincidence", got)
	}
	views := s.SketchConstraintGlyphs3D()
	if len(views) != 1 || !views[0].Selected {
		t.Errorf("picked marker not flagged Selected: %+v", views)
	}

	before := sk.GeometricConstraints3D().Count()
	if err := s.DeleteSelectedSketch3DConstraints(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if after := sk.GeometricConstraints3D().Count(); after != before-1 {
		t.Errorf("constraint count %d → %d, want one fewer after Delete", before, after)
	}
	if n := len(s.SelectedSketchConstraints()); n != 0 {
		t.Errorf("selection not cleared after delete: %d left", n)
	}
}

// TestPickSketchConstraint3DMissesEmptySpace: a click far from any marker selects nothing, so the
// constraint pick does not swallow a click meant for the geometry or empty space.
func TestPickSketchConstraint3DMissesEmptySpace(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	topDown800(s)
	l1 := sk.AddLine3D(math.P3(4, 4, 0), math.P3(5, 4, 0))
	l2 := sk.AddLine3D(math.P3(4, 5, 0), math.P3(5, 5, 0))
	sk.GeometricConstraints3D().Add(sketch.NewParallel3D(l1, l2)) // markers far from the origin
	s.SetShowSketchConstraints(true)

	if _, ok := s.PickSketchConstraint3DAt(400, 400); ok {
		t.Error("centre-pixel pick hit a marker that is off near (4.5,4/5), want a miss")
	}
}

// TestPickSketchConstraint3DEmptyWhenHidden: with Show Constraints off there are no markers to pick,
// even directly over where one would sit.
func TestPickSketchConstraint3DEmptyWhenHidden(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	topDown800(s)
	l1 := sk.AddLine3D(math.P3(-1, 0, 0), math.P3(0, 0, 0))
	l2 := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	sk.GeometricConstraints3D().Add(sketch.NewCoincident3D(l1.B, l2.A))
	// Show Constraints left OFF.
	if _, ok := s.PickSketchConstraint3DAt(400, 400); ok {
		t.Error("picked a marker while constraints are hidden")
	}
}
