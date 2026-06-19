// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/model/sketch"
)

// TestEnterExitSketchEmitsSketchEdit: entering and leaving a 2D sketch each emit a
// SketchEditChanged carrying the sketch identity, with Entered distinguishing them (#148).
func TestEnterExitSketchEmitsSketchEdit(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())

	var got []SketchEditChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e SketchEditChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	s.EnterSketch(sk)
	s.ExitSketch()

	if len(got) != 2 {
		t.Fatalf("emitted %d sketch-edit events, want 2 (enter, exit)", len(got))
	}
	if !got[0].Entered || got[1].Entered {
		t.Errorf("entered flags = %v/%v, want true then false", got[0].Entered, got[1].Entered)
	}
	if got[0].Sketch != sk.Seq() || got[0].Sketch == 0 {
		t.Errorf("sketch id = %d, want the sketch's seq %d", got[0].Sketch, sk.Seq())
	}
}
