// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/sketch"
)

// TestLineToolInfersHorizontalOnCommit: the interactive e2e of M06-F10 (#625)
// — clicking a nearly horizontal line applies the inferred horizontal on
// commit, and the solve squares it up.
func TestLineToolInfersHorizontalOnCommit(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)

	tool := NewLineTool()
	s.StartTool(tool)
	// Pixels map 1:1 to sketch units with the test camera; a 0.2-unit rise
	// over 30 units is well inside the 3° inference tolerance.
	s.Click(60, 100)
	s.Click(90, 100.2)
	_ = s.PressKey(KeyEvent{Key: "Escape"}) // finish the chain (#2024)
	if sk.Lines().Count() != 1 {
		t.Fatalf("lines = %d, want the committed line", sk.Lines().Count())
	}
	if got := sk.GeometricConstraints().Count(); got != 1 {
		t.Fatalf("constraints = %d, want the inferred horizontal", got)
	}
	sk.Solve()
	l := sk.Lines().Item(0)
	if dy := float64(l.B.Position().Y - l.A.Position().Y); math.Abs(dy) > 1e-9 {
		t.Errorf("solved Δy = %v, want 0", dy)
	}
}

// TestSessionInferenceOptionsToggle: the session preference is honored by the
// tool and survives a set/get round-trip.
func TestSessionInferenceOptionsToggle(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)

	opts := s.SketchInferenceOptions()
	if !opts.InferEnabled || opts.Priority != types.PriorityHorizontalVertical {
		t.Fatalf("defaults = %+v, want inference on with horizontal/vertical priority", opts)
	}
	opts.InferEnabled = false
	s.SetSketchInferenceOptions(opts)

	tool := NewLineTool()
	s.StartTool(tool)
	s.Click(60, 100)
	s.Click(90, 100.2)
	if got := sk.GeometricConstraints().Count(); got != 0 {
		t.Errorf("constraints with inference off = %d, want 0", got)
	}
}
