//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/scene"
)

const (
	axisGizmoMargin = 46  // gizmo-center inset from the viewport's bottom-left corner, px
	axisGizmoRadius = 30  // arrow length from center to tip, px
	axisArrowHead   = 9   // arrowhead length along the shaft, px
	axisShaftWidth  = 2.4 // shaft line thickness, px
)

// drawAxisGizmo paints the orientation triad (three colored arrows for +X/+Y/+Z) in the
// viewport's bottom-left corner, over the already-drawn image. originX/originY is the
// image's top-left in screen space (native.ItemRectMin right after the image) and ph its
// height. The triad reads the live camera, so it always shows which way each axis points
// for the current view (ADR-0004). Must be called inside the viewport window so the
// ImGui window draw list and its clip rect are the viewport's.
func drawAxisGizmo(cam scene.Camera, originX, originY float32, ph int) {
	cx := originX + axisGizmoMargin
	cy := originY + float32(ph) - axisGizmoMargin
	for _, a := range axisTriad(cam, axisGizmoRadius) {
		tipX, tipY := cx+a.tipX, cy+a.tipY
		native.DrawLine(cx, cy, tipX, tipY, a.color, axisShaftWidth)
		drawArrowhead(cx, cy, tipX, tipY, a.color)
		native.DrawText(tipX+3, tipY-7, a.label, a.color)
	}
}

// drawArrowhead fills a small triangle at the tip pointing along the shaft from the
// center. A near-zero shaft (axis pointing straight at the viewer) is skipped: there is
// no in-plane direction to orient the head.
func drawArrowhead(cx, cy, tipX, tipY float32, color [4]float32) {
	dx, dy := tipX-cx, tipY-cy
	l := float32(stdmath.Hypot(float64(dx), float64(dy)))
	if l < 1e-3 {
		return
	}
	ux, uy := dx/l, dy/l                                   // unit shaft direction
	px, py := -uy, ux                                      // in-plane perpendicular
	bx, by := tipX-ux*axisArrowHead, tipY-uy*axisArrowHead // head base center
	const halfW = axisArrowHead * 0.45
	native.DrawTriangleFilled(
		tipX, tipY,
		bx+px*halfW, by+py*halfW,
		bx-px*halfW, by-py*halfW,
		color,
	)
}
