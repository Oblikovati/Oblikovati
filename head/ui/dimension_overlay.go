//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"bytes"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/renderer"
	"github.com/Oblikovati/oblikovati/scene"
)

// Dimension display: the lines (extension/dimension/arc leaders) are 3D line items in
// the viewport draw list — like the grid and sketch geometry — while the value text is
// 2D, drawn over the rendered image at the label anchor's projected pixel. Double-
// clicking a label re-opens the dimension's value editor (drawDimensionPopup). The
// geometry + labels come pre-computed from app.Session.SketchDimensions (headless).

// dimensionLines accumulates every dimension's segments (mapped plane→model through the
// sketch plane) into colored line items, driving and driven dimensions kept apart.
func dimensionLines(plane sketch.Plane, views []app.DimensionView) []renderer.DrawItem {
	driving, driven := &segAccum{}, &segAccum{}
	for _, v := range views {
		acc := driving
		if v.Driven {
			acc = driven
		}
		for _, s := range v.Segments {
			acc.seg(plane, s[0], s[1])
		}
	}
	var items []renderer.DrawItem
	items = appendGrid(items, driving, dimensionColor)
	items = appendGrid(items, driven, dimensionDrivenColor)
	return items
}

// drawDimensionLabels overlays each dimension's value text at its projected anchor (the
// image has already been drawn at window-local cx,cy). Returns the dimension whose label
// was double-clicked this frame, or nil — the caller re-opens it for editing.
func drawDimensionLabels(cx, cy float32, cam scene.Camera, plane sketch.Plane, views []app.DimensionView) *sketch.DimensionConstraint {
	mx, my := native.MousePos()
	ox, oy := native.ItemRectMin()
	dbl := native.IsMouseDoubleClicked(native.MouseLeft)
	var hit *sketch.DimensionConstraint
	for _, v := range views {
		sx, sy, ok := renderer.Project(cam, viewportNear, viewportFar, plane.ToModel(v.LabelAt))
		if !ok {
			continue
		}
		native.SetCursorPos(cx+float32(sx), cy+float32(sy))
		native.Text(v.Label)
		if dbl && labelHit(mx-ox, my-oy, float32(sx), float32(sy)) {
			hit = v.Dim
		}
	}
	return hit
}

// labelHit reports whether the (viewport-local) cursor is within a label's rough text
// box anchored at (lblX, lblY) — the double-click target for editing a dimension.
func labelHit(curX, curY, lblX, lblY float32) bool {
	dx, dy := curX-lblX, curY-lblY
	return dx >= -4 && dx <= 64 && dy >= -2 && dy <= 18
}

// dimEdit holds the edit box's text buffer (owned across frames so ImGui edits it in
// place) and the dimension it is currently seeded for (to re-seed when a new edit opens).
var dimEdit = struct {
	buf        []byte
	dim        *sketch.DimensionConstraint
	posX, posY float32 // where to place the box (the cursor when it opened)
	place      bool    // set the position this frame (only on the opening frame)
}{buf: make([]byte, 64)}

// dimPopupCursorOffset nudges the edit box off the cursor so it does not cover the
// dimension being placed.
const dimPopupCursorOffset = 14

// drawDimensionPopup shows the value editor while a dimension is pending. It opens at the
// cursor (where the dimension was placed / double-clicked) and is then freely movable. OK
// commits the typed expression (driving the geometry through the solver); an invalid
// expression keeps the dimension pending so the box stays open. Cancel accepts the
// measured value.
func drawDimensionPopup(s *app.Session) {
	d := s.PendingDimension()
	if d == nil {
		dimEdit.dim = nil
		return
	}
	if d != dimEdit.dim { // a new edit opened — seed the buffer + remember the cursor
		seedEditBuf(s.PendingDimensionExpression())
		dimEdit.dim = d
		mx, my := native.MousePos()
		dimEdit.posX, dimEdit.posY = mx+dimPopupCursorOffset, my+dimPopupCursorOffset
		dimEdit.place = true
	}
	if dimEdit.place { // position once at the cursor, then let the user drag it
		native.SetNextWindowPos(dimEdit.posX, dimEdit.posY)
		dimEdit.place = false
	}
	native.SetNextWindowSize(260, 96)
	if native.Begin("Edit Dimension") {
		native.Text("Value")
		native.InputText("##dim-value", dimEdit.buf)
		if native.Button("OK") {
			_ = s.CommitPendingDimension(editBufText()) // keeps box open on parse error
		}
		native.SameLine()
		if native.Button("Cancel") {
			s.CancelPendingDimension()
		}
	}
	native.End()
}

// seedEditBuf copies expr into the edit buffer, NUL-terminated and zero-padded.
func seedEditBuf(expr string) {
	for i := range dimEdit.buf {
		dimEdit.buf[i] = 0
	}
	n := copy(dimEdit.buf, expr)
	if n >= len(dimEdit.buf) {
		dimEdit.buf[len(dimEdit.buf)-1] = 0
	}
}

// editBufText returns the NUL-terminated edit-buffer contents as a Go string.
func editBufText() string {
	if i := bytes.IndexByte(dimEdit.buf, 0); i >= 0 {
		return string(dimEdit.buf[:i])
	}
	return string(dimEdit.buf)
}
