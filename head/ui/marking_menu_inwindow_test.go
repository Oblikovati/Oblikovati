//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
)

// TestInWindowMarkingMenuRepeatAndStyles renders the right-click menu in both styles with an idle
// Repeat entry present, capturing each to a PNG. It is a smoke guard that the #915 entries draw
// through the real cgo path without crashing, and the captures back the live visual confirmation.
func TestInWindowMarkingMenuRepeatAndStyles(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register standard commands: %v", err)
	}
	// Run a no-op command so the idle "Repeat <command>" entry has a target, leaving no tool active.
	if err := s.Commands().Add(app.NewCommand("Test.Demo", "Demo Command", "Test", func(*app.Session) error { return nil })); err != nil {
		t.Fatalf("add demo command: %v", err)
	}
	if err := s.Execute("Test.Demo"); err != nil {
		t.Fatalf("execute demo: %v", err)
	}
	if _, _, ok := s.RepeatMenuEntry(); !ok {
		t.Fatal("expected an idle Repeat entry after running a command")
	}

	render := func() {
		for i := 0; i < 8; i++ {
			win.BeginFrame()
			DrawChrome(win, s)
			win.EndFrame(0.1, 0.1, 0.1)
		}
	}
	save := func(name string) {
		if err := win.SaveWindowPNG(filepath.Join(outDir(), name)); err != nil {
			t.Logf("SaveWindowPNG(%s): %v", name, err)
		}
	}

	// Radial marking menu (default) with the Repeat entry on top.
	openMarkingMenuOnFirstFrame = true
	render()
	save("mm-radial-repeat.png")

	// Switch to the classic linear menu — the same popup re-renders in the other style.
	s.SetClassicContextMenu(true)
	openMarkingMenuOnFirstFrame = true
	render()
	save("mm-classic.png")
}

// outDir returns a writable directory for the capture PNGs (OBK_SHOT_DIR or /tmp).
func outDir() string {
	if d := os.Getenv("OBK_SHOT_DIR"); d != "" {
		return d
	}
	return "/tmp"
}
