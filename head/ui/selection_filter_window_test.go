//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestInWindowSelectionFilterWindowDraws opens the real window with the Selection Filter window
// visible and runs frames, exercising the per-kind checkbox rows, the drag-source/target wiring,
// and the Select All / Deselect All buttons without tripping Dear ImGui's Begin/End assertions
// (#1222). A mixed state (one kind disabled, the list reordered) makes every row branch draw.
func TestInWindowSelectionFilterWindowDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	s.OpenSelectionFilterWindow()
	defer s.CloseSelectionFilterWindow()

	st := s.SelectionFilterState()
	st.SetEnabled(app.SelectVertex, false) // a disabled row draws unticked
	st.Move(st.Rank(app.SelectFace), 0)    // a reordered list draws in the new order

	for range 3 {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	if !s.SelectionFilterWindowOpen() {
		t.Error("the Selection Filter window should stay open across frames")
	}
}
