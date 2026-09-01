// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestBuildStatusIdle(t *testing.T) {
	t.Parallel()
	sb := BuildStatus(NewSession())
	if sb.ToolActive || sb.Prompt != "Ready" || sb.SelectionCount != 0 {
		t.Fatalf("idle status = %+v, want Ready / inactive / 0 selected", sb)
	}
	if sb.InSketch {
		t.Error("idle status should not report InSketch")
	}
}

// TestBuildStatusReportsSketchAndRelax checks the status model surfaces the sketch + Relax
// Mode state the command-window control row needs to draw its toggle (#791).
func TestBuildStatusReportsSketchAndRelax(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	_ = sk
	s.SetRelaxMode(true)
	sb := BuildStatus(s)
	if !sb.InSketch {
		t.Error("editing a 2D sketch should report InSketch")
	}
	if !sb.RelaxMode {
		t.Error("Relax Mode on should be reported in the status model")
	}
}

func TestBuildStatusGuidesExtrude(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)

	sb := BuildStatus(s)
	if !sb.ToolActive || sb.ToolName != "Extrude" || sb.CanCommit {
		t.Fatalf("started status = %+v, want active Extrude, not yet committable", sb)
	}
	if sb.Prompt != "Select a region to extrude (Ctrl+click to add more)" {
		t.Errorf("initial prompt = %q, want select-region guidance", sb.Prompt)
	}

	s.Click(120, 90) // synthetic click picks the profile
	ext.SetDistance(5)
	sb = BuildStatus(s)
	if !sb.CanCommit || sb.Prompt != "Set the extrude options and click OK" {
		t.Errorf("after profile + distance: %+v, want committable OK prompt", sb)
	}
}

// promptlessTool implements Tool but not Prompted, to cover the generic fallback prompt.
type promptlessTool struct{ commit bool }

func (promptlessTool) Name() string              { return "Fake" }
func (promptlessTool) Start(*Session)            {}
func (promptlessTool) Pick(*Session, Selectable) {}
func (t promptlessTool) CanCommit() bool         { return t.commit }
func (promptlessTool) Commit(*Session) error     { return nil }
func (promptlessTool) Cancel(*Session)           {}

func TestBuildStatusGenericPromptFallback(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.StartTool(promptlessTool{})
	if got := BuildStatus(s).Prompt; got != "Select or specify input for Fake" {
		t.Errorf("incomplete fallback prompt = %q", got)
	}

	s2 := NewSession()
	s2.StartTool(promptlessTool{commit: true})
	if got := BuildStatus(s2).Prompt; got != "Click OK to finish, or Cancel (Esc)" {
		t.Errorf("committable fallback prompt = %q", got)
	}
}
