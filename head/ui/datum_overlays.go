//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// datumOverlaySession is what drawing the datum overlays needs of the session: the active
// document's datums, the edit scope that hides ones created later, the selection to highlight,
// and whether Create 2D Sketch is revealing the origin frame to pick a host. A consumer-side
// interface rather than the whole *app.Session, per audit I5 (the arrowSession pattern).
type datumOverlaySession interface {
	ActiveWorkGeometry() (*feature.WorkGeometry, bool)
	EditScopeHides(seq uint64) bool
	SelectedWorkPlane() *feature.WorkPlane
	RevealSketchHostDatums() bool
	Selection() *app.Selection
}

var _ datumOverlaySession = (*app.Session)(nil)

// datumOverlays builds the work-plane, work-axis and work-point overlays for the active part or
// assembly's origin frame and user datums. Each kind draws only when View ▸ Object visibility
// shows it and the datum's own Visible flag is set — the rule the app-side pickers mirror, so
// that what is not drawn is not clickable.
func datumOverlays(s datumOverlaySession, cam scene.Camera, hovered *feature.WorkPlane, vis wire.ObjectVisibilityView) []renderer.DrawItem {
	wg, ok := s.ActiveWorkGeometry() // a part OR an assembly's origin frame + datums (#769 parity)
	if !ok {
		return nil
	}
	hidden := s.EditScopeHides // hide datums created after the node being edited (issue #132)
	var items []renderer.DrawItem
	if vis.WorkPlanes {
		items = append(items, planesOverlay(wg.WorkPlanes(), s.SelectedWorkPlane(), hovered, hidden, s.RevealSketchHostDatums())...)
	}
	if vis.WorkAxes {
		items = append(items, axesOverlay(wg.WorkAxes(), selectedWorkAxis(s.Selection()), hidden)...)
	}
	if vis.WorkPoints { // the origin Center Point and user work points (#2016)
		items = append(items, pointsDatumOverlay(wg.WorkPoints(), selectedWorkPoint(s.Selection()), hidden, datumPointPixels*cam.WorldPerPixel())...)
	}
	return items
}
