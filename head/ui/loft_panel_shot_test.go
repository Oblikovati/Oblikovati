//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestLoftPanelShot captures the reworked Loft property panel (#1521) with three cross-sections
// picked, so the new ordered Sections list and the Curves/Conditions tabs can be eyeballed. Skipped
// unless OBK_SHOT is set (it needs a window and saves a PNG; it asserts nothing).
func TestLoftPanelShot(t *testing.T) {
	if os.Getenv("OBK_SHOT") == "" {
		t.Skip("set OBK_SHOT to capture the Loft panel PNG")
	}
	win, err := native.CreateWindow(1500, 900, "obk-loft-panel-shot")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil

	s := loftThreeSectionSession(t)
	for i := 0; i < 12; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "loft-panel.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}

// loftThreeSectionSession builds a part with three stacked square sketches and starts the Loft tool
// with all three picked as cross-sections — the state that exercises the multi-row Sections list.
func loftThreeSectionSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "loft-panel-shot.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	lf := app.NewLoftTool()
	s.StartTool(lf)
	for i, z := range []float64{0, 4, 8} {
		half := 2.0 - float64(i)*0.6 // shrinking squares so the loft is an obvious frustum stack
		lf.Pick(s, app.ProfileHandle{Sketch: squareSketchAtZ(def, z, half), ProfileIndex: 0})
	}
	return s
}

// squareSketchAtZ adds a centered square of the given half-size on a plane at height z.
func squareSketchAtZ(def *compdef.PartComponentDefinition, z, half float64) *sketch.Sketch {
	pl, _ := sketch.NewPlane(math.P3(0, 0, z), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	sk := def.Sketches().Add(pl)
	c0 := sk.Points().Add(math.P2(-half, -half))
	c1 := sk.Points().Add(math.P2(half, -half))
	c2 := sk.Points().Add(math.P2(half, half))
	c3 := sk.Points().Add(math.P2(-half, half))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return sk
}
