// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestMiniToolbarLookupMissErrors covers the "no mini-toolbar" branches.
func TestMiniToolbarLookupMissErrors(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.UpdateMiniToolbarControls("ghost", nil); err == nil {
		t.Error("UpdateMiniToolbarControls(ghost) should error")
	}
	if err := s.RemoveMiniToolbar("ghost"); err == nil {
		t.Error("RemoveMiniToolbar(ghost) should error")
	}
	if err := s.CommitMiniToolbar("ghost", "tap"); err == nil {
		t.Error("CommitMiniToolbar(ghost) should error")
	}
}

// TestProgressLedgerLookupMissErrors covers the "no progress bar" branches.
func TestProgressLedgerLookupMissErrors(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if _, err := s.Progress().Update(404, 1, "x"); err == nil {
		t.Error("Update(404) should error")
	}
	if err := s.Progress().End(404); err == nil {
		t.Error("End(404) should error")
	}
	if err := s.CancelProgress(404); err == nil {
		t.Error("CancelProgress(404) should error")
	}
}

// TestSketchConstraintNeedTwoLinesErrors covers the "need two lines" guards for the
// perpendicular and collinear appliers when given no entities.
func TestSketchConstraintNeedTwoLinesErrors(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := applyPerpendicular(s, nil); err == nil {
		t.Error("applyPerpendicular(nil) should need two lines")
	}
	if err := applyCollinear(s, nil); err == nil {
		t.Error("applyCollinear(nil) should need two lines")
	}
}

// TestDrawingViewToolNameAndParams covers the view tools' Name/Params accessors (the
// base-view choice rows the property dialog reads).
func TestDrawingViewToolNameAndParams(t *testing.T) {
	t.Parallel()
	if NewBaseViewTool().Name() == "" {
		t.Error("BaseViewTool.Name() empty")
	}
	for _, p := range []ToolParams{
		NewProjectedViewTool().Params(),
		NewSliceViewTool().Params(),
		NewBreakoutViewTool().Params(),
	} {
		if len(p.Choices) == 0 {
			t.Error("view tool Params() returned no choices")
		}
	}
}
