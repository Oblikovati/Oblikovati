//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
)

// boxSession builds a session with the demo box (an active part with one body), the state the
// apply* helpers operate on.
func boxSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	buildBox(s)
	if len(s.VisibleBodies()) == 0 {
		t.Fatal("buildBox produced no visible body")
	}
	return s
}

// TestApplyHelpers exercises the scene-mutation helpers against a session (no window): edge /
// ground colors, style assignment, surface + image overlays, named views, and the style panel.
func TestApplyHelpers(t *testing.T) {
	s := boxSession(t)

	applyEdgeColor(s, "255,40,40")
	if c := s.DisplaySettings(0).EdgeColor; c.R != 255 || c.G != 40 || c.B != 40 {
		t.Errorf("edge color = %+v, want (255,40,40)", c)
	}

	applyGround(s, "40,90,200")
	if c := s.DisplaySettings(0).GroundPlane.Color; c.B != 200 || !s.ShadowSettings().GroundShadows {
		t.Errorf("ground color/shadows wrong: %+v shadows=%v", c, s.ShadowSettings().GroundShadows)
	}

	applyStyle(s, "Brass")
	key := string(s.VisibleBodies()[0].ReferenceKey())
	if name, ok := s.BodyColorStyle(key); !ok || name != "Brass" {
		t.Errorf("body style = (%q, %v), want Brass", name, ok)
	}

	applyOverlay(s)
	applyImageOverlay(s)
	if len(s.Graphics().Groups()) < 2 {
		t.Errorf("overlay + image should add two client-graphics groups, got %d", len(s.Graphics().Groups()))
	}

	applyNamedViews(s)
	if len(s.NamedViews()) != 2 || !s.NamedViewsPanelOpen() {
		t.Errorf("named views = %d, panel open = %v; want 2 + open", len(s.NamedViews()), s.NamedViewsPanelOpen())
	}

	applyStylesPanel(s)
	if !s.ColorStylesPanelOpen() {
		t.Error("applyStylesPanel should open the Color Styles panel")
	}
}

// TestApplySetup runs the full setup dispatch (no window) and checks the requested mutations
// took effect.
func TestApplySetup(t *testing.T) {
	s := boxSession(t)
	o := opts{orient: "iso", edge: "10,20,30", ground: "1,2,3", overlay: true, style: "Steel", dialog: true, image: true}
	if err := applySetup(s, o); err != nil {
		t.Fatalf("applySetup: %v", err)
	}
	if s.DisplaySettings(0).EdgeColor.G != 20 || !s.DisplaySettingsOpen() {
		t.Error("applySetup did not apply edge color / open the dialog")
	}
}

// TestRunCaptures runs the full capture (opens the real window, renders a couple frames, saves
// the viewport) end to end, so run + renderAndCapture are exercised.
func TestRunCaptures(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.png")
	if err := run(opts{box: true, frames: 2, out: out}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		t.Errorf("capture PNG not written: %v", err)
	}
}

// TestInjectBoxDrag exercises the box-select drag-injection sequence with a counting frame
// callback (no window): it must press, drag through several positions, and render a frame at
// each step. captureBoxSelect's window orchestration is verified manually (the live PNG) —
// the m16shot binary supports only one real window per test process.
func TestInjectBoxDrag(t *testing.T) {
	frames := 0
	injectBoxDrag(func() { frames++ })
	if frames < 6 {
		t.Errorf("injectBoxDrag rendered %d frames, want at least 6 (press + drag steps)", frames)
	}
}

// TestWriteCheckerPNG writes a checker PNG and checks it lands on disk.
func TestWriteCheckerPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checker.png")
	if err := writeCheckerPNG(path, 16); err != nil {
		t.Fatalf("writeCheckerPNG: %v", err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		t.Errorf("checker PNG not written: %v", err)
	}
}
