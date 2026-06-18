//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestInWindowHistoryBrowserDraws opens the real window with the History Browser visible over a
// part that has a couple of recorded edits, then runs frames — so a mismatched ImGui Begin/End
// or a bad table/selectable call would trip Dear ImGui's assertions. It also confirms the
// timeline the window reads matches the recorded steps.
func TestInWindowHistoryBrowserDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()
	s.EnsureActiveEditBaseline() // framedSession uses AddPart (no NewPart), so open the stream at the empty part

	if err := s.AddNumericUserParameter("w", "4 cm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	if err := s.AddNumericUserParameter("h", "3 cm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	id := s.ActiveDocument().ID()

	s.OpenHistoryBrowser()
	defer s.CloseHistoryBrowser()

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	tl, ok := s.DocumentHistoryView(id)
	if !ok || tl.Position != 2 || len(tl.Labels) != 2 {
		t.Fatalf("timeline = %+v ok=%v, want position 2 with 2 entries while the browser draws", tl, ok)
	}
}

// TestInWindowHistoryBrowserEmptyDraws opens the browser with no open documents and runs frames,
// covering the empty-state path without tripping ImGui assertions.
func TestInWindowHistoryBrowserEmptyDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := app.NewSession()
	s.OpenHistoryBrowser()
	defer s.CloseHistoryBrowser()

	for i := 0; i < 2; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// TestHistoryRowLabelMarksSavedAndUniqueID: a save checkpoint shows "*", and repeated step
// names get distinct widget ids (the "##pos" suffix) so the selectables do not collide.
func TestHistoryRowLabelMarksSavedAndUniqueID(t *testing.T) {
	tl := app.DocumentTimeline{Position: 2, Labels: []string{"Edit Parameters", "Edit Parameters"}, SavedDepths: []int{1}}

	saved := historyRowLabel("Edit Parameters", 1, tl)
	plain := historyRowLabel("Edit Parameters", 2, tl)
	if saved == plain {
		t.Fatalf("rows 1 and 2 produced identical labels %q (ids would collide)", saved)
	}
	if got := saved[:2]; got != "* " {
		t.Errorf("saved row prefix = %q, want %q", got, "* ")
	}
}
