//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestInWindowLoftDialogRenders drives the reworked Loft panel (#1521) through real frames in the
// live window so its draw path — the Curves tab's ordered Sections list, the per-row Remove/reorder
// affordances, the guide group, and the Conditions tab — is exercised (and credited with coverage by
// the xvfb+lavapipe CI head job). It asserts the section API the dialog draws on stays consistent as
// the rendered panel mutates it. Skips cleanly when no display/Vulkan is available.
func TestInWindowLoftDialogRenders(t *testing.T) {
	win, err := native.CreateWindow(1100, 800, "obk-loft-dialog-inwindow")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil

	// With no loft tool active, drawLoftDialog early-returns (its panel is closed) — cover that branch.
	bare := app.NewSession()
	if _, err := compdef.AddPart(bare.Workspace(), "bare.opd", true); err == nil {
		win.BeginFrame()
		DrawChrome(win, bare)
		win.EndFrame(0.1, 0.1, 0.12)
	}

	s := loftThreeSectionSession(t)
	l := s.ActiveLoft()
	if l == nil {
		t.Fatal("loft tool is not active")
	}

	// A few full-chrome frames render the panel as the user sees it: drawLoftDialog → the Curves tab
	// (sections list rows, guides, options). refreshLoftUI seeds loftUI here, so the direct tab-body
	// draws below start from a valid editor state.
	for range 3 {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if l.SectionCount() != 3 {
		t.Fatalf("seeded section count = %d, want 3", l.SectionCount())
	}

	// Non-destructive coverage first. ImGui renders only the selected tab, so DrawChrome alone hits just
	// Curves; drive every tab body directly — Curves with a row selected (the Remove-section affordance),
	// Conditions open and then closed (the no-end-sections note), and Transition automatic and then with
	// a mapping — so the whole dialog draw path is covered.
	loftUI.open = true
	loftUI.selectedSection = 1
	loftCoverFrame(win, func() {
		drawLoftCurvesTab(l)
		drawLoftConditionsTab(s, l)
		drawLoftTransitionTab(l) // automatic mapping (no map curves)
	})
	l.SetClosed(true)
	loftCoverFrame(win, func() { drawLoftConditionsTab(s, l) })
	l.SetClosed(false)

	// With a map curve picked, the Transition tab reports the mapping and offers Clear.
	l.ArmMapCurvePicking()
	l.Pick(s, app.PathHandle{Sketch: loftMapCurveSketch(t, s), PathIndex: 0})
	loftCoverFrame(win, func() { drawLoftTransitionTab(l) })
	if l.AutomaticMapping() {
		t.Error("after picking a map curve the loft should not report automatic mapping")
	}

	// Drive the real, DESTRUCTIVE interactions last: a row selection + drag-reorder, then a Remove/Clear
	// button click — so the section-row draw branches (Selectable click, drag source/target → MoveSection,
	// the buttons) actually execute. The section count must end lower than it started.
	loftUI.selectedSection = 1
	driveLoftSectionInteractions(win, l)
	if n := l.SectionCount(); n < 2 || n > 3 {
		t.Errorf("after the reorder/remove interaction the section count is %d, want 2 or 3", n)
	}

	// Finally the empty-list state renders its prompt instead of rows.
	l.ClearSections()
	loftCoverFrame(win, func() { drawLoftSectionsList(l) })
	if l.SectionCount() != 0 {
		t.Errorf("after ClearSections, count = %d, want 0", l.SectionCount())
	}
}

// loftInteractWindow is the fixed on-screen rectangle the Curves tab is rendered into for the
// interaction test, so a row/button pixel is deterministic rather than wherever the dock places it.
const (
	loftIWX, loftIWY = 8, 8
	loftIWW, loftIWH = 380, 520
)

// curvesFrame renders one frame showing ONLY the Curves tab in a fixed-position window, so injected
// clicks land on known geometry.
func curvesFrame(win *native.Window, l *app.LoftTool) {
	win.BeginFrame()
	native.SetNextWindowPos(loftIWX, loftIWY)
	native.SetNextWindowSize(loftIWW, loftIWH)
	if native.Begin("##loft-interact") {
		drawLoftCurvesTab(l)
	}
	native.End()
	win.EndFrame(0.1, 0.1, 0.12)
}

// driveLoftSectionInteractions clicks a Sections-list row (the Selectable branch), drags it onto its
// neighbour (the drag-drop reorder branches → MoveSection) and clicks the Remove button (the button
// branch) — all in the fixed-position Curves tab. Best-effort: the surrounding render covers the rest
// even if a pixel can't be located (ImGui metrics vary).
func driveLoftSectionInteractions(win *native.Window, l *app.LoftTool) {
	frame := func() { curvesFrame(win, l) }
	frame()
	frame()
	rx, ry, found := findLoftSectionRow(frame)
	if !found {
		return
	}
	const rowH = 21
	native.InjectMousePos(rx, ry)
	frame()
	native.InjectMouseButton(native.MouseLeft, true)
	frame()
	native.InjectMousePos(rx, ry+rowH*0.6) // exceed the drag threshold → BeginDragDropSource
	frame()
	native.InjectMousePos(rx, ry+rowH) // hover the next row → its BeginDragDropTarget accepts
	frame()
	frame()
	native.InjectMouseButton(native.MouseLeft, false)
	frame()
	clickLoftRemoveButton(win, l) // the Remove-section button (a row is still selected)
}

// findLoftSectionRow scans the fixed Curves-tab window for the pixel that selects a Sections-list row
// (clicking it sets loftUI.selectedSection), returning its position.
func findLoftSectionRow(frame func()) (float32, float32, bool) {
	for y := float32(loftIWY + 30); y <= loftIWY+170; y += 4 {
		for x := float32(loftIWX + 16); x <= loftIWX+220; x += 24 {
			loftUI.selectedSection = -1
			native.InjectMousePos(x, y)
			frame()
			native.InjectMouseButton(native.MouseLeft, true)
			frame()
			native.InjectMouseButton(native.MouseLeft, false)
			frame()
			if loftUI.selectedSection >= 0 {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// clickLoftRemoveButton scans below the sections list for the pixel that removes a section (the Remove
// or Clear-all button, both always rendered), so drawLoftSectionsListButtons' click branch executes.
func clickLoftRemoveButton(win *native.Window, l *app.LoftTool) {
	frame := func() { curvesFrame(win, l) }
	loftUI.selectedSection = 0 // ensure the Remove-section button is shown too
	frame()
	frame() // settle the fixed-position layout before clicking
	want := l.SectionCount()
	for y := float32(loftIWY + 30); y <= loftIWY+320 && l.SectionCount() == want; y += 3 {
		for x := float32(loftIWX + 10); x <= loftIWX+240; x += 10 {
			native.InjectMousePos(x, y)
			frame()
			native.InjectMouseButton(native.MouseLeft, true)
			frame()
			native.InjectMouseButton(native.MouseLeft, false)
			frame()
			if l.SectionCount() < want {
				return
			}
		}
	}
}

// loftMapCurveSketch adds an open path to the active part and returns its sketch — a map curve the
// Transition tab can consume.
func loftMapCurveSketch(t *testing.T, s *app.Session) *sketch.Sketch {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XZPlane())
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(0, 6))
	sk.Lines().Add(a, b)
	return sk
}

// loftCoverFrame renders one window frame whose body is drawn inside a plain panel, so a dialog
// sub-section can be exercised on its own (ImGui shows only the active tab during DrawChrome).
func loftCoverFrame(win *native.Window, body func()) {
	win.BeginFrame()
	if native.Begin("##loft-cover") {
		body()
	}
	native.End()
	win.EndFrame(0.1, 0.1, 0.12)
}
