//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The property panels EPIC #2053 added, rendered in a real window so their draw paths actually
// execute: every branch a user can reach — each mode of a mode-switching panel, and the rows
// that only appear once an input is gathered. Guards that they render without panicking, and
// gives head/ui the coverage a pure native.* draw function otherwise has none of.
// Skips cleanly where no display/Vulkan is available.
func TestEpic2053DialogsRender(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = newIconCache(win) // dialogs draw icon toggles directly

	frame := func(draw func()) {
		win.BeginFrame()
		draw()
		win.EndFrame(0.1, 0.1, 0.1)
	}

	t.Run("unwrap", func(t *testing.T) {
		s := app.NewSession()
		frame(func() { drawUnwrapDialog(s) }) // no tool: the panel must not draw
		s.StartTool(app.NewUnwrapTool())
		frame(func() { drawUnwrapDialog(s) })
	})

	t.Run("simplify", func(t *testing.T) {
		s := app.NewSession()
		frame(func() { drawSimplifyDialog(s) })
		sp := app.NewSimplifyTool()
		s.StartTool(sp)
		frame(func() { drawSimplifyDialog(s) })
		sp.SetFillVoids(true) // the toggle's other state
		frame(func() { drawSimplifyDialog(s) })
	})

	t.Run("anglePlane", func(t *testing.T) {
		s := partSessionForDialogs(t)
		frame(func() { drawAnglePlaneDialog(s) })
		s.StartTool(app.NewAngleWorkPlaneTool())
		frame(func() { drawAnglePlaneDialog(s) }) // seeds the angle field
		anglePlaneUI.angleDeg = 30
		frame(func() { drawAnglePlaneDialog(s) })
	})

	t.Run("modelTolerance", func(t *testing.T) {
		s := partSessionForDialogs(t)
		frame(func() { drawModelToleranceDialog(s) })
		s.StartTool(app.NewModelFrameTool()) // frame mode: characteristic + tolerance + datums
		frame(func() { drawModelToleranceDialog(s) })
		s.CancelTool()
		s.StartTool(app.NewModelDatumTool()) // datum mode: the label row instead
		frame(func() { drawModelToleranceDialog(s) })
	})

	t.Run("freeformCage", func(t *testing.T) {
		s := partSessionForDialogs(t)
		frame(func() { drawFreeformCageDialog(s) })
		s.StartTool(app.NewFreeformCageEditTool())
		frame(func() { drawFreeformCageDialog(s) })
		cageEditUI.level, cageEditUI.sharpness = 2, 0.5
		frame(func() { drawFreeformCageDialog(s) })
	})

	t.Run("chamferModes", func(t *testing.T) {
		s := app.NewSession()
		c := app.NewChamferTool()
		s.StartTool(c)
		for i := range app.ChamferTypeNames() { // distance / two distances / distance and angle
			c.SetChamferTypeIndex(i)
			chamferUI.typeIndex = i
			frame(func() { drawChamferDialog(s) })
		}
	})

	t.Run("replaceFacePlaneTarget", func(t *testing.T) {
		s := partSessionForDialogs(t)
		r := app.NewReplaceFaceTool()
		s.StartTool(r)
		frame(func() { drawReplaceFaceDialog(s) })
		r.SetPickingTarget(true)
		r.Pick(s, app.WorkPlaneHandle{Plane: originPlaneForDialogs(t, s)})
		frame(func() { drawReplaceFaceDialog(s) }) // the chip now reads "1 Plane"
	})

	t.Run("thickenOptions", func(t *testing.T) {
		s := app.NewSession()
		th := app.NewThickenTool()
		s.StartTool(th)
		frame(func() { drawThickenDialog(s) })
		th.SetDirectionIndex(2)
		th.SetOperationIndex(3)
		th.SetAsSurface(true)
		frame(func() { drawThickenDialog(s) })
	})
}

// partSessionForDialogs is a session with an active part, which the datum and freeform panels
// need to resolve anything at all.
func partSessionForDialogs(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "epic2053-dialogs.opd", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	return s
}

// originPlaneForDialogs returns the active part's XY origin plane.
func originPlaneForDialogs(t *testing.T, s *app.Session) *feature.WorkPlane {
	t.Helper()
	part := activePart(s)
	if part == nil {
		t.Fatal("no active part")
	}
	planes := part.OriginPlanes()
	if len(planes) == 0 {
		t.Fatal("the part has no origin planes")
	}
	return planes[0]
}
