// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestActiveGripSnapNil: ActiveGripSnap is nil with no tool and with a different tool running.
func TestActiveGripSnapNil(t *testing.T) {
	s := assemblySession(t)
	if s.ActiveGripSnap() != nil {
		t.Error("ActiveGripSnap should be nil when no tool is running")
	}
	s.StartTool(NewAssemblyConstraintTool("Mate", 2, nil))
	if s.ActiveGripSnap() != nil {
		t.Error("ActiveGripSnap should be nil when a different tool is running")
	}
	s.StartTool(NewGripSnapTool())
	if s.ActiveGripSnap() == nil {
		t.Error("ActiveGripSnap should return the running grip-snap tool")
	}
}

// TestGripSnapToolPickProgression: the tool takes exactly two face picks (move + target) and only
// then enables commit; extra picks are ignored.
func TestGripSnapToolPickProgression(t *testing.T) {
	tool := NewGripSnapTool()
	if tool.Name() != "Grip Snap" {
		t.Errorf("Name() = %q, want Grip Snap", tool.Name())
	}
	if tool.CanCommit() {
		t.Error("no picks yet: CanCommit should be false")
	}
	tool.Pick(nil, FaceHandle{})
	if tool.CanCommit() {
		t.Error("one pick: CanCommit should be false")
	}
	tool.Pick(nil, FaceHandle{})
	if !tool.CanCommit() {
		t.Error("two picks: CanCommit should be true")
	}
	tool.Pick(nil, FaceHandle{}) // beyond two: ignored
	if len(tool.faces) != 2 {
		t.Errorf("collected %d faces, want 2 (extra picks ignored)", len(tool.faces))
	}
}

// TestGripSnapToolRejectsUnresolvedPick: committing with faces that do not resolve to placed
// components is a clean error, not a crash.
func TestGripSnapToolRejectsUnresolvedPick(t *testing.T) {
	s := assemblySession(t)
	tool := NewGripSnapTool()
	tool.Pick(s, FaceHandle{})
	tool.Pick(s, FaceHandle{})
	if err := tool.Commit(s); err == nil {
		t.Error("commit with unresolved faces should return an error")
	}
}

// TestGripSnapToolPreferOverride: the Constraint option round-trips through the index accessors, and
// the default is Auto (index 0).
func TestGripSnapToolPreferOverride(t *testing.T) {
	tool := NewGripSnapTool()
	if got := tool.PreferIndex(); got != 0 {
		t.Errorf("default PreferIndex = %d, want 0 (Auto)", got)
	}
	for i := range GripSnapPreferOptions() {
		tool.SetPreferIndex(i)
		if got := tool.PreferIndex(); got != i {
			t.Errorf("SetPreferIndex(%d) round-trips to %d", i, got)
		}
	}
	tool.SetPreferIndex(99) // out of range: ignored
	if got := tool.PreferIndex(); got != len(GripSnapPreferOptions())-1 {
		t.Errorf("out-of-range SetPreferIndex changed the selection to %d", got)
	}
}
