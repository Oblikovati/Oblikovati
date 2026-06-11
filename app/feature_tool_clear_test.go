// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// The property panels' selector chips clear one pick each (⊗). These lock the clears:
// the pick empties, the tool stops being committable, and the breadcrumb name follows.

func TestRevolveClearProfileEmptiesSelection(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	rv := NewRevolveTool()
	s.StartTool(rv)
	rv.Pick(s, profile)
	if !rv.CanCommit() {
		t.Fatal("a picked profile should make the revolve committable")
	}
	if rv.SourceSketchName() == "" {
		t.Error("SourceSketchName() = \"\" with a picked profile, want the sketch's name")
	}
	rv.ClearProfile()
	if _, ok := rv.PickedProfile(); ok || rv.CanCommit() {
		t.Error("after ClearProfile the revolve must have no profile and not be committable")
	}
	if name := rv.SourceSketchName(); name != "" {
		t.Errorf("SourceSketchName() = %q after clearing, want \"\"", name)
	}
}

func TestSweepClearsProfileAndPathIndependently(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	sw := NewSweepTool()
	s.StartTool(sw)
	sw.Pick(s, profile)
	sw.Pick(s, PathHandle{Sketch: profile.Sketch, PathIndex: 0})
	if !sw.CanCommit() {
		t.Fatal("profile + path should make the sweep committable")
	}
	sw.ClearPath()
	if _, ok := sw.PickedPath(); ok {
		t.Error("after ClearPath the sweep must have no path")
	}
	if _, ok := sw.PickedProfile(); !ok {
		t.Error("ClearPath must not drop the profile pick")
	}
	sw.ClearProfile()
	if _, ok := sw.PickedProfile(); ok || sw.SourceSketchName() != "" {
		t.Error("after ClearProfile the sweep must have no profile and no source sketch name")
	}
}

func TestHoleClearFaceEmptiesSelection(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	h := NewHoleTool()
	s.StartTool(h)
	h.Pick(s, FaceHandle{})
	if _, ok := h.PickedFace(); !ok {
		t.Fatal("picking a face handle should set the placement face")
	}
	h.ClearFace()
	if _, ok := h.PickedFace(); ok || h.CanCommit() {
		t.Error("after ClearFace the hole must have no face and not be committable")
	}
}
