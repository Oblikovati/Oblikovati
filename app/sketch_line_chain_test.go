// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2024: the line tool chained only when the command line armed it; the viewport ran a
// single two-point mode, so every edge of a profile cost a fresh tool activation and the
// segments did not share endpoints. These drive the viewport click path.

// TestLineToolChainsSegments is the headline regression: four clicks make three connected
// segments without re-activating the tool.
func TestLineToolChainsSegments(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewLineTool())
	for _, p := range [][2]float64{{40, 40}, {80, 40}, {80, 80}, {40, 80}} {
		s.Click(p[0], p[1])
		if s.ActiveTool() == nil {
			t.Fatalf("the tool deactivated mid-chain at %v — it is not chaining", p)
		}
	}
	_ = s.PressKey(KeyEvent{Key: "Enter"})

	if got := sk.Lines().Count(); got != 3 {
		t.Fatalf("four clicks gave %d lines, want 3 connected segments", got)
	}
}

// TestChainedSegmentsShareEndpoints is why chaining matters beyond click count: consecutive
// segments must share one Point, not merely touch at coincident coordinates. Independent
// endpoints look identical on screen but drag apart and never solve as a closed profile.
func TestChainedSegmentsShareEndpoints(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(80, 40)
	s.Click(80, 80)
	_ = s.PressKey(KeyEvent{Key: "Enter"})

	if sk.Lines().Count() != 2 {
		t.Fatalf("got %d lines, want 2", sk.Lines().Count())
	}
	first, second := sk.Lines().Item(0), sk.Lines().Item(1)
	if first.B != second.A {
		t.Errorf("segments do not share their joining point (%p vs %p) — they only touch",
			first.B, second.A)
	}
}

// TestEscapeEndsChainKeepingSegments: Escape is how CAD users say "done" mid-polyline, and it
// must not throw away the geometry they have already placed.
func TestEscapeEndsChainKeepingSegments(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(80, 40)
	s.Click(80, 80)
	_ = s.PressKey(KeyEvent{Key: "Escape"})

	if got := sk.Lines().Count(); got != 2 {
		t.Errorf("Escape mid-chain left %d lines, want the 2 already drawn", got)
	}
	if s.ActiveTool() != nil {
		t.Error("Escape should end the tool")
	}
}

// TestEscapeBeforeAnySegmentStillCancels keeps Escape's ordinary meaning where there is
// nothing to keep: one click has placed no segment, so it abandons.
func TestEscapeBeforeAnySegmentStillCancels(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	_ = s.PressKey(KeyEvent{Key: "Escape"})

	if got := sk.Lines().Count(); got != 0 {
		t.Errorf("Escape after one click created %d lines, want 0", got)
	}
	if s.ActiveTool() != nil {
		t.Error("Escape should end the tool")
	}
}

// TestChainPreviewFollowsTheLastPoint: the rubber band anchored to the FIRST point, so every
// segment after the first previewed from the wrong end (or not at all).
func TestChainPreviewFollowsTheLastPoint(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewLineTool()
	s.StartTool(tool)
	s.Click(40, 40)
	s.Click(80, 40)

	last, ok := tool.PendingReferencePoint()
	if !ok {
		t.Fatal("no pending reference point mid-chain")
	}
	cursor := math.P2(last.X+1, last.Y+1)
	r, ok := tool.PendingRecipe(s, cursor, nil)
	if !ok {
		t.Fatal("no preview mid-chain — the rubber band vanishes after the first segment")
	}
	if got := r.Points[0]; got.DistanceTo(last) > 1e-9 {
		t.Errorf("preview starts at %v, want the last placed point %v", got, last)
	}
}

// TestChainedLineIsOneUndoStep: finishing a chain records a single edit, so Ctrl+Z removes the
// polyline the user drew rather than unpicking it segment by segment.
//
// Uses CreateSketch rather than the sketchSession helper: CreateSketch is what establishes the
// undo baseline, and without it Undo reports "nothing to undo" for ANY tool (verified against a
// plain circle commit), which would make this test pass or fail for the wrong reason.
func TestChainedLineIsOneUndoStep(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(80, 40)
	s.Click(80, 80)
	_ = s.PressKey(KeyEvent{Key: "Escape"})
	if sk.Lines().Count() != 2 {
		t.Fatalf("setup: got %d lines, want 2", sk.Lines().Count())
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if got := s.ActiveSketch().Lines().Count(); got != 0 {
		t.Errorf("one undo left %d lines, want 0 — the chain was not a single edit", got)
	}
}
