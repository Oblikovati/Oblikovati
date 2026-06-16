//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"flag"
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

// TestRunBoxSelectCaptures runs the full box-select capture (opens the real window, installs
// the picker, injects a drag, saves the window mid-drag) end to end, so run + captureBoxSelect
// are exercised. The m16shot binary supports only ONE real window per test process (a second
// native.CreateWindow crashes on GLFW/Vulkan teardown), so this is the single window test and
// it covers the box-select path added in #916.
func TestRunBoxSelectCaptures(t *testing.T) {
	out := filepath.Join(t.TempDir(), "box.png")
	if err := run(opts{box: true, boxselect: true, frames: 2, out: out}); err != nil {
		t.Fatalf("run boxselect: %v", err)
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		t.Errorf("box-select capture PNG not written: %v", err)
	}
}

// TestParseOpts checks the flag parser maps command-line args onto the capture config.
func TestParseOpts(t *testing.T) {
	oldArgs, oldFlags := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldFlags }()
	flag.CommandLine = flag.NewFlagSet("m16shot", flag.ContinueOnError)
	os.Args = []string{"m16shot", "-box", "-boxselect", "-frames", "3", "-out", "/tmp/parseopts.png"}

	o := parseOpts()
	if !o.box || !o.boxselect || o.frames != 3 || o.out != "/tmp/parseopts.png" {
		t.Errorf("parseOpts = %+v, want box+boxselect, frames 3, the given out path", o)
	}
}

// TestEnterDimensionedRectangle exercises the sketch-capture scene setup (no window): it must
// create a sketch with the four rectangle edges, add a dimension, and enter the sketch env.
func TestEnterDimensionedRectangle(t *testing.T) {
	s := app.NewSession()
	if err := enterDimensionedRectangle(s); err != nil {
		t.Fatalf("enterDimensionedRectangle: %v", err)
	}
	sk := s.ActiveSketch()
	if sk == nil || !s.InSketch() {
		t.Fatal("should be editing the new sketch")
	}
	if sk.Lines().Count() != 4 {
		t.Errorf("rectangle should have 4 lines, got %d", sk.Lines().Count())
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
