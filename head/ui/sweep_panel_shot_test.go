//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/compdef"
)

// TestSweepPanelShot captures the reworked Sweep property panel (#1521) with a profile, path, and
// guide rail picked, so the Inventor-style flow — the Guide Rail slot, the Orientation/Taper/Twist
// behaviour rows, and the rail-only Profile Scaling combo — can be eyeballed. Skipped unless OBK_SHOT
// is set (it needs a window and saves a PNG; it asserts nothing).
func TestSweepPanelShot(t *testing.T) {
	if os.Getenv("OBK_SHOT") == "" {
		t.Skip("set OBK_SHOT to capture the Sweep panel PNG")
	}
	win, err := native.CreateWindow(1500, 900, "obk-sweep-panel-shot")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil

	s := sweepRailSession(t)
	for i := 0; i < 12; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "sweep-panel.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}

// sweepRailSession builds a part with a square profile, a straight path, and a guide rail, and starts
// the Sweep tool with all three picked — the state that shows every Inventor row, including Profile
// Scaling (rail-only).
func sweepRailSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "sweep-panel-shot.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sw := app.NewSweepTool()
	s.StartTool(sw)
	sw.Pick(s, app.ProfileHandle{Sketch: squareSketchAtZ(def, 0, 1), ProfileIndex: 0})
	sw.Pick(s, lineSketchOnXZ(def, 0, 0, 5))
	sw.ArmGuideRailPicking()
	sw.Pick(s, lineSketchOnXZ(def, 2, 0, 5))
	return s
}
