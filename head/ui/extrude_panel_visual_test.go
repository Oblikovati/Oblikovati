//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestInWindowExtrudePanelVisualHold is a manual visual check: it opens the real window
// with the Extrude tool active over a picked square profile and holds it so a human (or
// a screenshot tool) can inspect the Extrusion property panel. Skipped unless
// OBK_VISUAL_HOLD=1 — it renders nothing assertable and exists only for layout work.
func TestInWindowExtrudePanelVisualHold(t *testing.T) {
	if os.Getenv("OBK_VISUAL_HOLD") == "" {
		t.Skip("manual visual check only (set OBK_VISUAL_HOLD=1)")
	}
	win, err := native.CreateWindow(1500, 900, "obk-extrude-panel-visual")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil
	s, profile := extrudeReadySession(t)
	ext := app.NewExtrudeTool()
	s.StartTool(ext)
	if os.Getenv("OBK_VISUAL_EMPTY") == "" { // EMPTY=1 shows the required/empty selector state
		ext.Pick(s, profile)
	}
	for i := 0; i < 6000; i++ { // long enough to screenshot even uncapped (~30s+)
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// extrudeReadySession builds a part session holding a 2×2 closed square sketch — the
// profile the Extrude panel extrudes — with the standard commands registered.
func extrudeReadySession(t *testing.T) (*app.Session, app.ProfileHandle) {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "extrude-panel-visual.obk", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(2, 0))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(0, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return s, app.ProfileHandle{Sketch: sk, ProfileIndex: 0}
}
