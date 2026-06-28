//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSweepOrientationScalingIndexRoundTrip locks the combo-index ↔ enum mappings the Sweep dialog's
// Orientation and Profile Scaling combos draw on (#1521): every enum value round-trips through its
// label index, and an unknown index falls back to the Inventor default.
func TestSweepOrientationScalingIndexRoundTrip(t *testing.T) {
	for _, o := range []types.SweepProfileOrientation{types.NormalToPath, types.ParallelToOriginalProfile} {
		if got := sweepOrientationFromIndex(sweepOrientationIndex(o)); got != o {
			t.Errorf("orientation %v did not round-trip: got %v", o, got)
		}
	}
	if sweepOrientationFromIndex(99) != types.NormalToPath {
		t.Error("unknown orientation index should default to Follow Path (NormalToPath)")
	}
	for _, sc := range []types.SweepProfileScaling{types.XYProfileScaling, types.XProfileScaling, types.NoProfileScaling} {
		if got := sweepScalingFromIndex(sweepScalingIndex(sc)); got != sc {
			t.Errorf("scaling %v did not round-trip: got %v", sc, got)
		}
	}
	if sweepScalingFromIndex(99) != types.XYProfileScaling {
		t.Error("unknown scaling index should default to X & Y")
	}
}

// sweepRigSession builds a part with a square profile (XY), a straight path up Z (XZ), and a guide
// rail offset +2 in X (XZ), then starts the Sweep tool with the profile and path picked — the rig the
// dialog draws on.
func sweepRigSession(t *testing.T) (*app.Session, *app.SweepTool, app.PathHandle) {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "sweep-dialog.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sw := app.NewSweepTool()
	s.StartTool(sw)
	sw.Pick(s, app.ProfileHandle{Sketch: squareSketchAtZ(def, 0, 1), ProfileIndex: 0})
	sw.Pick(s, lineSketchOnXZ(def, 0, 0, 5)) // path: (0,0,0)→(0,0,5)
	rail := lineSketchOnXZ(def, 2, 0, 5)     // rail: (2,0,0)→(2,0,5)
	return s, sw, rail
}

// lineSketchOnXZ adds a vertical line on the XZ plane from (x,y0) to (x,y1) and returns its path
// handle (model (x,0,y0)→(x,0,y1)).
func lineSketchOnXZ(def *compdef.PartComponentDefinition, x, y0, y1 float64) app.PathHandle {
	sk := def.Sketches().Add(sketch.XZPlane())
	a := sk.Points().Add(math.P2(math.Scalar(x), math.Scalar(y0)))
	b := sk.Points().Add(math.P2(math.Scalar(x), math.Scalar(y1)))
	sk.Lines().Add(a, b)
	return app.PathHandle{Sketch: sk, PathIndex: 0}
}

// TestInWindowSweepDialogRenders drives the reworked Sweep panel (#1521) through real frames so its
// whole draw path is exercised (and credited by the xvfb+lavapipe CI head job): Input Geometry with
// the optional Guide Rail (empty, armed, then filled), the Behavior section's orientation/taper/twist
// and the rail-only Profile Scaling row, and Output. It also best-effort drives the Orientation and
// Profile Scaling combos so their selection branches run. Skips cleanly with no display/Vulkan.
func TestInWindowSweepDialogRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = newIconCache(win)

	s, sw, rail := sweepRigSession(t)
	sweepUI.open = false // re-seed from the tool on the next frame

	panel := func() {
		win.BeginFrame()
		drawSweepDialog(s)
		win.EndFrame(0.1, 0.1, 0.12)
	}

	// No guide rail yet: Input Geometry shows the optional empty chip; Behavior hides Profile Scaling.
	panel()

	// Arm the rail selector → the chip reads "Selecting…"; then pick the rail → it fills and the
	// Profile Scaling row appears.
	sw.ArmGuideRailPicking()
	panel()
	sw.Pick(s, rail)
	if _, ok := sw.PickedGuideRail(); !ok {
		t.Fatal("rail pick did not land in the guide-rail slot")
	}
	panel()

	// Drive the guide-rail chip's × so drawSweepGuideRail's clear branch runs (the rail is filled).
	driveSweepGuideRailChip(win, sw, func() bool {
		_, ok := sw.PickedGuideRail()
		return !ok
	})
	if _, ok := sw.PickedGuideRail(); ok {
		t.Error("clicking the guide-rail × did not clear it")
	}

	// Now the chip is empty: clicking it arms rail picking (the arm branch).
	driveSweepGuideRailChip(win, sw, sw.GuideRailArmed)
	if !sw.GuideRailArmed() {
		t.Error("clicking the empty guide-rail chip did not arm picking")
	}
	sw.ClearGuideRail() // disarm before leaving

	// Best-effort: drive the Orientation combo to Parallel and the Profile Scaling combo off X & Y so
	// their selection branches run. (Re-pick the rail so Profile Scaling renders.)
	sw.ArmGuideRailPicking()
	sw.Pick(s, rail)
	driveSweepCombo(win, func() { drawSweepOrientation(sw) }, func() bool {
		return sw.Orientation() == types.ParallelToOriginalProfile
	})
	driveSweepCombo(win, func() { drawSweepScaling(sw) }, func() bool {
		return sw.Scaling() != types.XYProfileScaling
	})
}

// driveSweepGuideRailChip renders the Guide Rail row alone in a fixed-position window and scans the
// row left-to-right, clicking until target() flips — hitting the chip body (arm) or its × (clear),
// so drawSweepGuideRail's arm/clear branches run. Best-effort: the surrounding renders cover the rest.
func driveSweepGuideRailChip(win *native.Window, sw *app.SweepTool, target func() bool) {
	frame := func() {
		win.BeginFrame()
		native.SetNextWindowPos(sweepComboX, sweepComboY)
		native.SetNextWindowSize(sweepComboW, sweepComboH)
		if native.Begin("##sweep-guiderail") {
			drawSweepGuideRail(sw)
		}
		native.End()
		win.EndFrame(0.1, 0.1, 0.12)
	}
	y := float32(sweepComboY + 34) // the chip row, below the title bar
	frame()
	for x := float32(sweepComboX + 8 + propertyLabelWidth); x <= sweepComboW-20 && !target(); x += 8 {
		native.InjectMousePos(x, y)
		frame()
		native.InjectMouseButton(native.MouseLeft, true)
		frame()
		native.InjectMouseButton(native.MouseLeft, false)
		frame()
	}
}

// sweepComboPos is the fixed on-screen rectangle a single combo is rendered into so its button and
// drop-down items land at deterministic pixels.
const (
	sweepComboX, sweepComboY = 8, 8
	sweepComboW, sweepComboH = 300, 200
)

// driveSweepCombo renders one combo (body) alone in a fixed-position window, opens it, and scans the
// drop-down for the option that flips changed() — covering the combo's selection branch. Best-effort:
// the surrounding renders cover the rest even if no pixel selects (ImGui metrics vary).
func driveSweepCombo(win *native.Window, body func(), changed func() bool) {
	frame := func() {
		win.BeginFrame()
		native.SetNextWindowPos(sweepComboX, sweepComboY)
		native.SetNextWindowSize(sweepComboW, sweepComboH)
		if native.Begin("##sweep-combo") {
			body()
		}
		native.End()
		win.EndFrame(0.1, 0.1, 0.12)
	}
	bx := float32(sweepComboX + 8 + propertyLabelWidth + 15) // the combo button (label column + a bit)
	by := float32(sweepComboY + 34)                          // the first content row, below the title bar
	openCombo := func() {
		native.InjectMousePos(bx, by)
		frame()
		native.InjectMouseButton(native.MouseLeft, true)
		frame()
		native.InjectMouseButton(native.MouseLeft, false)
		frame()
	}
	frame()
	openCombo()
	for iy := by + 18; iy <= by+150 && !changed(); iy += 5 {
		native.InjectMousePos(bx, iy)
		frame()
		native.InjectMouseButton(native.MouseLeft, true)
		frame()
		native.InjectMouseButton(native.MouseLeft, false)
		frame()
		if !changed() {
			openCombo() // the click likely closed the drop-down without selecting — reopen and try lower
		}
	}
}
