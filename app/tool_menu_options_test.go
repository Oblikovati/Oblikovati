// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestActiveToolMenuOptionsReflectOffsetToggles: the Offset tool surfaces Loop Select and Constrain
// Offset as checkable right-click options, both on by default, and toggling one through the session
// flips exactly that option.
func TestActiveToolMenuOptionsReflectOffsetToggles(t *testing.T) {
	s, _ := emptyPartSession(t)
	tool := NewSketchOffsetTool(0.5)
	s.StartTool(tool)

	opts := s.ActiveToolMenuOptions()
	if len(opts) != 2 {
		t.Fatalf("offset tool exposed %d menu options, want 2", len(opts))
	}
	if opts[0].Label != "Loop Select" || !opts[0].Checked {
		t.Errorf("option[0] = %+v, want Loop Select checked", opts[0])
	}
	if opts[1].Label != "Constrain Offset" || !opts[1].Checked {
		t.Errorf("option[1] = %+v, want Constrain Offset checked", opts[1])
	}

	s.ToggleActiveToolMenuOption("Loop Select")
	if tool.LoopSelect() {
		t.Error("Loop Select still on after toggle")
	}
	if !tool.ConstrainOffset() {
		t.Error("toggling Loop Select must not affect Constrain Offset")
	}

	s.ToggleActiveToolMenuOption("Constrain Offset")
	if tool.ConstrainOffset() {
		t.Error("Constrain Offset still on after toggle")
	}
}

// TestActiveToolMenuOptionsNoneWhenIdle: with no tool active, and for a tool that offers none, the
// session reports no options (so the right-click menu shows nothing extra).
func TestActiveToolMenuOptionsNoneWhenIdle(t *testing.T) {
	s, _ := emptyPartSession(t)
	if opts := s.ActiveToolMenuOptions(); opts != nil {
		t.Errorf("idle session exposed %d tool menu options, want none", len(opts))
	}
	s.ToggleActiveToolMenuOption("Loop Select") // must not panic with no active tool
}
