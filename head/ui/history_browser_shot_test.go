//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
	"oblikovati.org/scene"
)

// TestShotHistoryBrowser renders the History Browser over TWO documents — each with several
// edits and a save checkpoint — and writes a PNG of the whole window, so the side-by-side
// timelines and the "* saved" markers can be eyeballed headlessly. Skipped unless OBK_SHOT is
// set, since it writes files and is for manual inspection.
func TestShotHistoryBrowser(t *testing.T) {
	if os.Getenv("OBK_SHOT") == "" {
		t.Skip("set OBK_SHOT to capture the History Browser PNG")
	}
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	dir := t.TempDir()
	s := app.NewSessionWithStore(persistence.NewPackageStore())
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)

	bracket := shotDocWithHistory(t, s, filepath.Join(dir, "bracket.obk"), []string{"width", "height", "thickness"})
	plate := shotDocWithHistory(t, s, filepath.Join(dir, "plate.obk"), []string{"length", "holes"})
	historyBrowserSelection[bracket] = true
	historyBrowserSelection[plate] = true

	s.OpenHistoryBrowser()
	for i := 0; i < 4; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.12, 0.12, 0.14)
	}
	if err := win.SaveWindowPNG("/tmp/oblikovati-history-browser.png"); err != nil {
		t.Fatalf("SaveWindowPNG: %v", err)
	}
	t.Log("wrote /tmp/oblikovati-history-browser.png")
}

// shotDocWithHistory adds a part, records one edit per name, saves (a checkpoint), then records
// one more unsaved edit, returning the document id. It leaves the document active.
func shotDocWithHistory(t *testing.T, s *app.Session, path string, names []string) doc.ID {
	t.Helper()
	d, err := compdef.AddPart(s.Workspace(), path, true)
	if err != nil {
		t.Fatalf("AddPart %s: %v", path, err)
	}
	s.EnsureActiveEditBaseline()
	for _, n := range names {
		if err := s.AddNumericUserParameter(n, "4 cm"); err != nil {
			t.Fatalf("add %s: %v", n, err)
		}
	}
	if err := s.SaveDocumentAs(d, path); err != nil {
		t.Fatalf("save %s: %v", path, err)
	}
	if err := s.AddNumericUserParameter("draft", "2 deg"); err != nil {
		t.Fatalf("post-save edit: %v", err)
	}
	return d.ID()
}
