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

	s := loftThreeSectionSession(t)
	l := s.ActiveLoft()
	if l == nil {
		t.Fatal("loft tool is not active")
	}

	// A few full-chrome frames render the panel as the user sees it: drawLoftDialog → the Curves tab
	// (sections list rows, guides, options). refreshLoftUI seeds loftUI here, so the direct tab-body
	// draws below start from a valid editor state.
	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if l.SectionCount() != 3 {
		t.Fatalf("seeded section count = %d, want 3", l.SectionCount())
	}

	// Drive a real drag-reorder + row selection through the docked panel, so the section-row draw
	// branches — the Selectable click and the drag source/target that call MoveSection — execute.
	driveLoftSectionReorder(win, s)

	// ImGui renders only the selected tab, so DrawChrome alone hits just Curves. Drive every tab body
	// directly — Curves with a row selected (the Remove-section affordance), Conditions open and then
	// closed (the no-end-sections note), and Transition automatic and then with a mapping — so the
	// whole dialog draw path is covered.
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

	// Removing the selected row through the API the dialog button calls keeps the list consistent.
	l.RemoveSection(1)
	if l.SectionCount() != 2 {
		t.Errorf("after RemoveSection, count = %d, want 2", l.SectionCount())
	}

	// The empty-list state renders its prompt instead of rows.
	l.ClearSections()
	loftCoverFrame(win, func() { drawLoftSectionsList(l) })
	if l.SectionCount() != 0 {
		t.Errorf("after ClearSections, count = %d, want 0", l.SectionCount())
	}
}

// driveLoftSectionReorder finds a Sections-list row in the docked panel and drags it onto the next
// row, exercising the Selectable-click and drag-drop reorder branches of drawLoftSectionRows /
// reorderLoftSectionRow. Best-effort: if the row pixel can't be located (ImGui metrics vary), the
// surrounding render still covers the rest of the draw path.
func driveLoftSectionReorder(win *native.Window, s *app.Session) {
	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	frame()
	frame()
	rx, ry, found := findLoftSectionRow(frame)
	if !found {
		return
	}
	// Press the row, cross the drag threshold, then hover the next row before releasing so ImGui starts
	// the drag-drop and the target accepts it (reorderLoftSectionRow's source + target → MoveSection).
	const rowH = 21
	native.InjectMousePos(rx, ry)
	frame()
	native.InjectMouseButton(native.MouseLeft, true)
	frame()
	native.InjectMousePos(rx, ry+rowH*0.5)
	frame()
	native.InjectMousePos(rx, ry+rowH)
	frame()
	frame()
	native.InjectMouseButton(native.MouseLeft, false)
	frame()

	// Right-click the row to open its context menu (the Remove-section path's BeginPopupContextItem),
	// then dismiss it by clicking empty space below the list.
	native.InjectMousePos(rx, ry)
	frame()
	native.InjectMouseButton(native.MouseRight, true)
	frame()
	native.InjectMouseButton(native.MouseRight, false)
	frame()
	native.InjectMousePos(rx, ry+rowH*5)
	native.InjectMouseButton(native.MouseLeft, true)
	frame()
	native.InjectMouseButton(native.MouseLeft, false)
	frame()
}

// findLoftSectionRow scans the docked property panel for a pixel that selects a Sections-list row
// (clicking it sets loftUI.selectedSection), returning its position. frame renders one DrawChrome frame.
func findLoftSectionRow(frame func()) (float32, float32, bool) {
	for y := float32(150); y <= 320; y += 6 {
		for x := float32(80); x <= 260; x += 30 {
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
