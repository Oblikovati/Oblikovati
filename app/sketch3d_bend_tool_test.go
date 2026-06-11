// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSketch3DBendToolAppliesViaPicks drives the ribbon command + pick flow: set the
// radius, pick two chained lines, and the corner becomes a tangent arc with its
// maintaining constraint.
func TestSketch3DBendToolAppliesViaPicks(t *testing.T) {
	s, sk := sketch3DSession(t)
	l1 := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	l2 := sk.AddLine3D(math.P3(1, 0, 0), math.P3(1, 1, 0))

	if err := s.Execute("Sketch3D.Bend"); err != nil {
		t.Fatalf("Sketch3D.Bend: %v", err)
	}
	tool, ok := s.ActiveTool().Tool().(*Bend3DTool)
	if !ok {
		t.Fatalf("active tool = %T, want *Bend3DTool", s.ActiveTool().Tool())
	}
	tool.Params().Floats[0].Set(0.25)
	s.feedPick(SketchEntityHandle{Entity: l1})
	if s.ActiveTool() == nil {
		t.Fatal("tool should stay active after one pick")
	}
	s.feedPick(SketchEntityHandle{Entity: l2}) // second pick → bend applies

	if got := sk.GeometricConstraints3D().Count(); got != 1 {
		t.Fatalf("constraints after bend = %d, want the maintaining bend", got)
	}
	if _, ok := sk.GeometricConstraints3D().Item(0).(*sketch.Bend3D); !ok {
		t.Errorf("constraint = %T, want *sketch.Bend3D", sk.GeometricConstraints3D().Item(0))
	}
	if got := l1.B.Position(); float64(got.DistanceTo(math.P3(0.75, 0, 0))) > 1e-9 {
		t.Errorf("line1 trimmed end = %v, want (0.75,0,0)", got)
	}
	if paths := sk.Paths3D(); len(paths) != 1 {
		t.Errorf("paths after bend = %d, want 1 connected chain", len(paths))
	}
}

// TestSketch3DBendToolRejectsNonLines guards the accept filter and repeats.
func TestSketch3DBendToolRejectsNonLines(t *testing.T) {
	s, sk := sketch3DSession(t)
	p := sk.AddPoint3D(math.P3(0, 0, 0))
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	if err := s.Execute("Sketch3D.Bend"); err != nil {
		t.Fatalf("Sketch3D.Bend: %v", err)
	}
	tool := s.ActiveTool().Tool().(*Bend3DTool)
	if tool.Accepts(p) {
		t.Error("the bend tool should not accept a point")
	}
	s.feedPick(SketchEntityHandle{Entity: p})
	s.feedPick(SketchEntityHandle{Entity: l})
	s.feedPick(SketchEntityHandle{Entity: l}) // repeat ignored
	if got := len(tool.Picked()); got != 1 {
		t.Errorf("picked = %d entities, want only the line once", got)
	}
}
