//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Box-select input + rubber-band rendering for the single viewport. A left-drag that
// starts on empty space sweeps a window/crossing selection box (app.Session owns the state
// machine + hit-test); a press on geometry falls through to the normal click select. The
// rectangle is drawn on top of the rendered viewport image (#916, follows the Inventor
// model chosen in the audit: left-drag is selection, never navigation).

// box-select rectangle colours (0..1 RGBA): a window select (drag left→right) is blue, a
// crossing select (drag right→left) is green — Inventor's convention. The fill is faint so
// the geometry stays visible through it.
var (
	boxWindowFill   = [4]float32{0.30, 0.55, 0.95, 0.18}
	boxWindowBorder = [4]float32{0.40, 0.65, 1.00, 0.90}
	boxCrossFill    = [4]float32{0.35, 0.80, 0.40, 0.18}
	boxCrossBorder  = [4]float32{0.45, 0.90, 0.50, 0.90}
)

// handleViewportSelection routes a left press/drag over the viewport: in the sketch editor a
// press on an unconstrained entity drags it; otherwise a drag begun on empty space runs
// box-select; anything else (a press on geometry, or no left input) falls through to the
// single-pick click handler. Replaces the bare handleViewportClick call so they never both
// fire on the same press.
func handleViewportSelection(s *app.Session) {
	if updateOrbitPivot(s) {
		return // Free Orbit: a no-drag click set the orbit pivot (#913 N9)
	}
	if s.ConstrainedOrbitActive() {
		return // Constrained Orbit owns the left-drag, which turntables (#913 N10)
	}
	if heldNavMode() != NavNone {
		return // a held F2/F3/F4 turns a left-drag into navigation, not selection (#911)
	}
	if updateZoomWindow(s) {
		return // the Zoom Window tool owns the left-drag while armed (#913 N16)
	}
	if s.SelectOtherActive() {
		if native.IsItemClicked(native.MouseLeft) {
			s.CommitSelectOther() // a click in the viewport accepts the highlighted candidate (#910)
		}
		return // the Select Other cycle owns viewport picking until it ends
	}
	if updateSketchPlacement(s) {
		return
	}
	if updateSketchDrag(s) {
		return
	}
	if updateCloudDrag(s) {
		return
	}
	if updateControlPointDrag(s) {
		return
	}
	if updateBoxSelect(s) {
		return
	}
	handleViewportClick(s)
}

// updateControlPointDrag advances interactive NURBS control-point editing and reports whether it
// consumed this frame's left input. While the Edit Control Points tool is active, a left press on
// a control-net handle begins the drag, the cursor slides it (the surface re-evaluates live), and
// release commits the edit (M36-F03). Mirrors updateCloudDrag.
func updateControlPointDrag(s *app.Session) bool {
	if !s.CVEditActive() {
		return false
	}
	if s.CVDragActive() {
		lx, ly := viewportCursor()
		if native.MouseDown(native.MouseLeft) {
			s.UpdateCVDrag(lx, ly)
		} else {
			s.CommitCVDrag()
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) {
		return false
	}
	lx, ly := viewportCursor()
	return s.BeginCVDrag(lx, ly)
}

// updateCloudDrag advances the interactive Move of a point cloud and reports whether it consumed
// this frame's left input. While the Move tool is active, a left press begins the drag, the cursor
// translates the cloud (datums built on it follow), and release commits — mirroring the sketch
// drag-to-move (#645).
func updateCloudDrag(s *app.Session) bool {
	if !s.CloudMoveActive() {
		return false
	}
	if s.CloudDragActive() {
		lx, ly := viewportCursor()
		if native.MouseDown(native.MouseLeft) {
			s.UpdateCloudDrag(lx, ly)
		} else {
			s.CommitCloudDrag()
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) {
		return false
	}
	lx, ly := viewportCursor()
	return s.BeginCloudDrag(lx, ly)
}

// placementInputSession is the session surface drag-to-create drives (audit I5, the
// arrowSession pattern): the sketch-environment gate plus the placement state machine.
type placementInputSession interface {
	InSketch() bool
	PlacementActive() bool
	BeginPlacement(px, py float64) bool
	UpdatePlacement(px, py float64)
	EndPlacement(px, py float64)
}

var _ placementInputSession = (*app.Session)(nil)

// updateSketchPlacement drives drag-to-create for sketch geometry tools and reports whether it
// consumed this frame's left input: a press places the first point, the cursor rubber-bands the
// shape, and release places the second point and commits it (#2014). A press and release without
// movement falls back to the click-click flow, so both gestures work and produce the same
// geometry. It runs before updateSketchDrag, which only moves already-committed entities and
// stands down while a tool is active anyway.
func updateSketchPlacement(s placementInputSession) bool {
	if !s.InSketch() {
		return false
	}
	lx, ly := viewportCursor()
	if s.PlacementActive() {
		return advanceSketchPlacement(s, lx, ly)
	}
	if !native.IsItemClicked(native.MouseLeft) {
		return false
	}
	return s.BeginPlacement(lx, ly)
}

// advanceSketchPlacement tracks the held cursor and finishes the placement on release.
func advanceSketchPlacement(s placementInputSession, lx, ly float64) bool {
	if native.MouseDown(native.MouseLeft) {
		s.UpdatePlacement(lx, ly)
		return true
	}
	s.EndPlacement(lx, ly)
	return true
}

// updateSketchDrag advances direct drag-to-move of sketch entities and reports whether it
// consumed this frame's left input. A press on an unconstrained entity (no modifier, no active
// tool) begins the drag; the cursor moves it; release commits. A Shift/Ctrl press is a
// selection-extend, not a drag, so it falls through.
func updateSketchDrag(s *app.Session) bool {
	if !s.InSketch() {
		return false
	}
	if s.EntityDragActive() {
		lx, ly := viewportCursor()
		if native.MouseDown(native.MouseLeft) {
			s.UpdateEntityDrag(lx, ly)
		} else {
			s.CommitEntityDrag()
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) || native.KeyShift() || native.KeyCtrl() {
		return false
	}
	lx, ly := viewportCursor()
	return s.BeginEntityDrag(lx, ly)
}

// updateBoxSelect advances the box-select state machine and reports whether it consumed this
// frame's left input. It begins a box only on a fresh left press over empty space, tracks the
// cursor while the button is held, and commits on release. In the sketch editor it selects 2D
// entities; in the model env it selects bodies (#909). updateSketchDrag runs first, so a press on
// a draggable sketch entity moves it rather than starting a box.
func updateBoxSelect(s *app.Session) bool {
	if s.BoxSelectActive() {
		lx, ly := viewportCursor()
		if native.MouseDown(native.MouseLeft) {
			s.UpdateBoxSelect(lx, ly)
		} else {
			s.CommitBoxSelect(viewportSelectMods())
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) {
		return false
	}
	if s.ActiveToolConsumesClicks() {
		return false // a click tool (sketch geometry, point-cloud Crop Box) owns the press, not box-select
	}
	lx, ly := viewportCursor()
	if _, hit := s.PickAt(lx, ly, app.NewSelectionFilter()); hit {
		return false // pressed on geometry → normal click select (drag-to-move is future work)
	}
	s.BeginBoxSelect(lx, ly)
	return s.BoxSelectActive() // false when no RegionPicker is installed → fall through to click
}

// updateZoomWindow drives the Zoom Window rubber band while the tool is armed: a fresh left press
// anchors the box, the cursor sweeps it, and release zooms the view to fit it (#913 N16). It consumes
// all left input while armed so the drag never also selects. Esc disarms via cancelViewportTool.
func updateZoomWindow(s *app.Session) bool {
	if !s.ZoomWindowArmed() {
		return false
	}
	lx, ly := viewportCursor()
	if s.ZoomWindowDragging() {
		if native.MouseDown(native.MouseLeft) {
			s.UpdateZoomWindow(lx, ly)
		} else {
			s.CommitZoomWindow()
		}
		return true
	}
	if native.IsItemClicked(native.MouseLeft) {
		s.BeginZoomWindow(lx, ly)
	}
	return true
}

// drawZoomWindowRect draws the in-progress Zoom Window rubber band over the viewport image, in a
// neutral white (distinct from box-select's blue/green window/crossing colours). No-op when idle.
func drawZoomWindowRect(s *app.Session, bx, by float32) {
	if !s.ZoomWindowDragging() {
		return
	}
	x0, y0, x1, y1 := s.ZoomWindowRect()
	sx0, sy0 := bx+float32(x0), by+float32(y0)
	sx1, sy1 := bx+float32(x1), by+float32(y1)
	fill := [4]float32{0.85, 0.85, 0.90, 0.15}
	border := [4]float32{0.95, 0.95, 1.00, 0.95}
	native.DrawQuadFilled(sx0, sy0, sx1, sy0, sx1, sy1, sx0, sy1, fill)
	native.DrawLine(sx0, sy0, sx1, sy0, border, 1.2)
	native.DrawLine(sx1, sy0, sx1, sy1, border, 1.2)
	native.DrawLine(sx1, sy1, sx0, sy1, border, 1.2)
	native.DrawLine(sx0, sy1, sx0, sy0, border, 1.2)
}

// orbitPivotClickSlop is the max drag (pixels) under which an F4 left press counts as a click
// (set-pivot) rather than an orbit drag.
const orbitPivotClickSlop = 4

// orbitPivot tracks one F4 left press: armed once pressed, moved once it drags past the slop.
var orbitPivot struct {
	armed bool
	moved bool
}

// updateOrbitPivot sets a new orbit pivot when the user clicks (presses then releases without
// dragging past the slop) in the viewport while Free Orbit (F4) is held — Inventor's set-pivot
// (#913 N9). A drag orbits instead (ApplyNavigation handles the rotation), so only a no-drag click
// sets the pivot. Reports whether it consumed this frame's left input (so selection stands down).
func updateOrbitPivot(s *app.Session) bool {
	if heldNavMode() != NavOrbit {
		orbitPivot.armed = false
		return false
	}
	if native.IsItemClicked(native.MouseLeft) {
		orbitPivot.armed, orbitPivot.moved = true, false
	}
	if !orbitPivot.armed {
		return false
	}
	if native.MouseDown(native.MouseLeft) {
		dx, dy := native.MouseDelta()
		if dx*dx+dy*dy > orbitPivotClickSlop*orbitPivotClickSlop {
			orbitPivot.moved = true
		}
		return true // the drag (if any) orbits via ApplyNavigation; selection stands down
	}
	if !orbitPivot.moved { // released without dragging → a click sets the pivot
		cx, cy := viewportCursor()
		s.SetOrbitPivot(cx, cy)
	}
	orbitPivot.armed = false
	return true
}

// drawOrbitRing draws the Free-Orbit ring centred on the viewport while F4 (Free Orbit) is held, so
// the user sees the zones a drag selects: inside = free orbit, rim = axis-locked, outside = roll
// (#913 N5–N8). bx,by is the viewport image's top-left in screen pixels; pw,ph its pixel size.
func drawOrbitRing(bx, by float32, pw, ph int) {
	if heldNavMode() != NavOrbit {
		return
	}
	cx, cy := bx+float32(pw)/2, by+float32(ph)/2
	radius := float32(orbitRingRadius(float64(pw), float64(ph)))
	col := [4]float32{0.85, 0.85, 0.95, 0.55}
	const seg = 48
	prevx, prevy := cx+radius, cy
	for i := 1; i <= seg; i++ {
		a := 2 * stdmath.Pi * float64(i) / seg
		x := cx + radius*float32(stdmath.Cos(a))
		y := cy + radius*float32(stdmath.Sin(a))
		native.DrawLine(prevx, prevy, x, y, col, 1.4)
		prevx, prevy = x, y
	}
}

// viewportSelectMods reads the Shift/Ctrl modifiers held this frame (Shift adds, Ctrl
// inverts the box's hits).
func viewportSelectMods() app.Modifier {
	var m app.Modifier
	if native.KeyShift() {
		m |= app.ShiftMod
	}
	if native.KeyCtrl() {
		m |= app.CtrlMod
	}
	return m
}

// drawBoxSelectRect draws the in-progress rubber-band rectangle over the viewport image.
// bx,by is the viewport image's top-left in screen pixels (the box stores viewport-local
// pixels). A no-op when no box-select drag is active.
func drawBoxSelectRect(s *app.Session, bx, by float32) {
	if !s.BoxSelectActive() {
		return
	}
	x0, y0, x1, y1, crossing := s.BoxSelectRect()
	sx0, sy0 := bx+float32(x0), by+float32(y0)
	sx1, sy1 := bx+float32(x1), by+float32(y1)
	fill, border := boxWindowFill, boxWindowBorder
	if crossing {
		fill, border = boxCrossFill, boxCrossBorder
	}
	native.DrawQuadFilled(sx0, sy0, sx1, sy0, sx1, sy1, sx0, sy1, fill)
	native.DrawLine(sx0, sy0, sx1, sy0, border, 1.2)
	native.DrawLine(sx1, sy0, sx1, sy1, border, 1.2)
	native.DrawLine(sx1, sy1, sx0, sy1, border, 1.2)
	native.DrawLine(sx0, sy1, sx0, sy0, border, 1.2)
}
