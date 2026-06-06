// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	gmath "oblikovati/math"
	"oblikovati/model/sketch"
)

// TestAutoDimensionCommandConstrainsSketch drives the Auto Dimension ribbon command
// end-to-end: an under-constrained rectangle becomes fully (well-)constrained.
func TestAutoDimensionCommandConstrainsSketch(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	sk.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	if sk.DegreesOfFreedom() == 0 {
		t.Fatal("rectangle should start under-constrained")
	}

	if err := s.Execute("Sketch.AutoDimension"); err != nil {
		t.Fatalf("Execute Sketch.AutoDimension: %v", err)
	}

	a := sk.AnalyzeConstraints()
	if a.DOF != 0 || a.Redundant != 0 {
		t.Fatalf("after Auto Dimension: DOF=%d Redundant=%d, want 0/0", a.DOF, a.Redundant)
	}
}

func TestAutoDimensionCommandRegistered(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if _, ok := s.Commands().ByID("Sketch.AutoDimension"); !ok {
		t.Error("Sketch.AutoDimension command not registered")
	}
}
