//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/compdef"
)

// TestInWindowSketchRibbonVisualHold is a manual visual check: it opens the real window
// in the sketch environment and holds it for ~15s so a human (or a screenshot tool) can
// inspect the contextual Sketch ribbon. Skipped unless OBK_VISUAL_HOLD=1 — it renders
// nothing assertable and exists only for eyeballing layout work.
func TestInWindowSketchRibbonVisualHold(t *testing.T) {
	if os.Getenv("OBK_VISUAL_HOLD") == "" {
		t.Skip("manual visual check only (set OBK_VISUAL_HOLD=1)")
	}
	win, err := native.CreateWindow(1800, 600, "obk-sketch-ribbon-visual")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil
	s := sketchEnvSession(t)
	for i := 0; i < 6000; i++ { // long enough to screenshot even uncapped (~30s+)
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// sketchEnvSession builds a part session with the standard commands registered and the
// sketch environment entered via the pre-selected-XY-plane path (no interactive pick).
func sketchEnvSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "sketch-ribbon-visual.opd", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	var origin app.BrowserNode
	for _, c := range app.BuildBrowser(s).Children {
		if c.Kind == "origin" {
			origin = c
		}
	}
	if len(origin.Children) == 0 {
		t.Fatal("browser tree has no Origin folder with planes")
	}
	s.SelectBrowserNode(origin.Children[0]) // XY Plane
	if err := s.Execute("Sketch.Create2D"); err != nil {
		t.Fatalf("Create 2D Sketch: %v", err)
	}
	if !s.InSketch() {
		t.Fatal("session did not enter the sketch environment")
	}
	return s
}
