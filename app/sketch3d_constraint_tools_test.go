// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// sketch3DSession builds a session editing a fresh 3D sketch with the standard
// commands registered — the fixture every 3D constraint-tool test starts from.
func sketch3DSession(t *testing.T) (*Session, *sketch.Sketch3D) {
	t.Helper()
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	sk, err := s.CreateSketch3D()
	if err != nil {
		t.Fatalf("create 3D sketch: %v", err)
	}
	return s, sk
}

// TestSketch3DTangentToolAppliesViaPicks drives the ribbon command + pick flow end to
// end: activate Sketch3D.Tangent, pick a line and an arc, and the constraint applies
// and re-solves the sketch into a tangent join.
func TestSketch3DTangentToolAppliesViaPicks(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	a := sk.AddArc3D(math.P3(1, 1, 0), math.P3(1.3, 0.2, 0), math.P3(2, 1, 0), false)

	if err := s.Execute("Sketch3D.Tangent"); err != nil {
		t.Fatalf("Sketch3D.Tangent: %v", err)
	}
	s.feedPick(SketchEntityHandle{Entity: l})
	if s.ActiveTool() == nil {
		t.Fatal("tool should stay active after one pick")
	}
	s.feedPick(SketchEntityHandle{Entity: a}) // second pick → auto-applies + deactivates
	if s.ActiveTool() != nil {
		t.Error("tool should deactivate after applying the constraint")
	}
	if sk.GeometricConstraints3D().Count() != 1 {
		t.Fatalf("constraints = %d, want 1", sk.GeometricConstraints3D().Count())
	}
	if d := float64(l.B.Position().DistanceTo(a.Start.Position())); d > 1e-6 {
		t.Errorf("join endpoints %g apart after tool applied, want coincident", d)
	}
}

// TestSketch3DSmoothToolNeedsSpline keeps gathering picks for two analytic curves
// (not ready) and applies once a spline is picked.
func TestSketch3DSmoothToolNeedsSpline(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	l1 := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	l2 := sk.AddLine3D(math.P3(1.2, 0, 0), math.P3(2, 1, 0))
	sp := sk.AddSpline3D([]math.Point3{{X: 1.1, Y: 0.1, Z: 0}, {X: 2, Y: 1, Z: 0.5}, {X: 3, Y: 0, Z: 1}}, false, true)

	if err := s.Execute("Sketch3D.Smooth"); err != nil {
		t.Fatalf("Sketch3D.Smooth: %v", err)
	}
	s.feedPick(SketchEntityHandle{Entity: l1})
	s.feedPick(SketchEntityHandle{Entity: l2})
	if s.ActiveTool() == nil {
		t.Fatal("two analytic curves must not satisfy Smooth (needs a spline)")
	}
	s.feedPick(SketchEntityHandle{Entity: sp})
	if s.ActiveTool() != nil {
		t.Error("tool should deactivate after the spline pick completes the constraint")
	}
	if sk.GeometricConstraints3D().Count() != 1 {
		t.Fatalf("constraints = %d, want 1", sk.GeometricConstraints3D().Count())
	}
	if _, ok := sk.GeometricConstraints3D().Item(0).(*sketch.Smooth3D); !ok {
		t.Errorf("applied constraint = %T, want *Smooth3D", sk.GeometricConstraints3D().Item(0))
	}
}

// TestSketch3DHelicalToolApplies picks a helix and a circle (any order) and ties them.
func TestSketch3DHelicalToolApplies(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	zAxis, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("axis: %v", err)
	}
	circle := sk.AddCircle3D(math.P3(0, 0, 0), zAxis, 5)
	helix := sk.AddHelix3D(math.P3(0.4, 0.1, 0), zAxis, 4, 1, 0, 6, false)

	if err := s.Execute("Sketch3D.Helical"); err != nil {
		t.Fatalf("Sketch3D.Helical: %v", err)
	}
	s.feedPick(SketchEntityHandle{Entity: circle}) // circle first: order must not matter
	s.feedPick(SketchEntityHandle{Entity: helix})
	if s.ActiveTool() != nil {
		t.Error("tool should deactivate after applying the constraint")
	}
	if sk.GeometricConstraints3D().Count() != 1 {
		t.Fatalf("constraints = %d, want 1", sk.GeometricConstraints3D().Count())
	}
	if d := float64(helix.Origin.Position().DistanceTo(circle.Center.Position())); d > 1e-6 {
		t.Errorf("helix origin %g from circle center after solve, want coincident", d)
	}
	if dr := stdmath.Abs(float64(helix.StartRadius - circle.Radius)); dr > 1e-6 {
		t.Errorf("start radius differs from circle radius by %g after solve", dr)
	}
}

// TestSketch3DSplineFitToolApplies attaches a point to a fit spline via picks.
func TestSketch3DSplineFitToolApplies(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	sp := sk.AddSpline3D([]math.Point3{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0}, {X: 2, Y: 0, Z: 1}}, false, true)
	p := sk.AddPoint3D(math.P3(1.1, 0.9, 0.1))

	if err := s.Execute("Sketch3D.SplineFit"); err != nil {
		t.Fatalf("Sketch3D.SplineFit: %v", err)
	}
	s.feedPick(SketchEntityHandle{Entity: sp})
	s.feedPick(SketchEntityHandle{Entity: p})
	if s.ActiveTool() != nil {
		t.Error("tool should deactivate after applying the constraint")
	}
	if d := float64(sp.Points[1].Position().DistanceTo(p.Position())); d > 1e-6 {
		t.Errorf("point %g from its fit point after solve, want attached", d)
	}
}

// TestSketch3DConstrainToolRejectsWrongType guards the hover/accept filter: the
// helical tool must not record a line pick.
func TestSketch3DConstrainToolRejectsWrongType(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	if err := s.Execute("Sketch3D.Helical"); err != nil {
		t.Fatalf("Sketch3D.Helical: %v", err)
	}
	tool, ok := s.ActiveTool().Tool().(*ConstraintTool)
	if !ok {
		t.Fatalf("active tool = %T, want *ConstraintTool", s.ActiveTool().Tool())
	}
	if tool.Accepts(l) {
		t.Error("the helical tool should not accept a line")
	}
	s.feedPick(SketchEntityHandle{Entity: l})
	if len(tool.Picked()) != 0 {
		t.Errorf("rejected pick was recorded: %d picks", len(tool.Picked()))
	}
}

// TestSketch3DConstrainCommandsDisabledOutsideSketch guards the enable predicate: the
// Constrain commands are contextual to the 3D-sketch environment.
func TestSketch3DConstrainCommandsDisabledOutsideSketch(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	for _, id := range []string{"Sketch3D.Tangent", "Sketch3D.Smooth", "Sketch3D.Helical", "Sketch3D.SplineFit"} {
		cmd, ok := s.Commands().ByID(id)
		if !ok {
			t.Fatalf("command %s is not registered", id)
		}
		if cmd.IsEnabled(s) {
			t.Errorf("%s enabled outside the 3D-sketch environment", id)
		}
	}
}
