//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// The in-place dimension boxes (#2014): one small value box per dimensionable quantity of the
// shape being placed, sitting at the midpoint of its dotted witness line. The active box is
// filled and takes keystrokes; a box whose value the user locked with Tab carries a padlock, and
// that value becomes a driving dimension when the shape commits.

// box geometry and colours.
const (
	placementBoxPadX    = 5 // horizontal text padding inside a box
	placementBoxPadY    = 2 // vertical text padding inside a box
	placementBoxLockGap = 3 // gap between the value text and the padlock
	placementBoxOffsetY = 6 // lift above the witness-line midpoint (px)
)

var (
	placementBoxFill       = [4]float32{0.96, 0.96, 0.98, 0.95}
	placementBoxText       = [4]float32{0.08, 0.09, 0.12, 1}
	placementBoxActiveFill = [4]float32{0.16, 0.34, 0.72, 0.95}
	placementBoxActiveText = [4]float32{1, 1, 1, 1}
	placementBoxBorder     = [4]float32{0.55, 0.58, 0.66, 0.9}
	padlockColor           = [4]float32{0.78, 0.60, 0.10, 1}
)

// drawPlacementFieldBoxes paints the in-place dimension boxes over the rendered viewport, each
// at its witness line's projected midpoint. bx,by is the viewport image's top-left in screen
// pixels. A no-op when no shape is being placed.
func drawPlacementFieldBoxes(s *app.Session, cam scene.Camera, bx, by float32) {
	sk := s.ActiveSketch()
	if sk == nil {
		return
	}
	plane := sk.Plane()
	for _, f := range s.PlacementFields() {
		mid := witnessMidpoint(f)
		x, y, ok := renderer.Project(cam, viewportNear, viewportFar, plane.ToModel(mid))
		if !ok {
			continue
		}
		drawPlacementFieldBox(f, bx+float32(x), by+float32(y)-placementBoxOffsetY)
	}
}

// witnessMidpoint is where a field's box sits: the middle of its extension line.
func witnessMidpoint(f app.PlacementFieldView) math.Point2 {
	a, b := f.Witness[0], f.Witness[1]
	return math.P2((a.X+b.X)/2, (a.Y+b.Y)/2)
}

// drawPlacementFieldBox paints one box centred above (x,y): a filled rounded cell with the
// value, the unit, and a padlock when the value is locked.
func drawPlacementFieldBox(f app.PlacementFieldView, x, y float32) {
	label := f.Value + " " + f.Unit
	w := native.CalcTextWidth(label) + 2*placementBoxPadX
	if f.Locked {
		w += native.TextLineHeight()*0.55 + placementBoxLockGap
	}
	h := native.TextLineHeight() + 2*placementBoxPadY
	x0, y0 := x-w/2, y-h
	fill, text := placementBoxFill, placementBoxText
	if f.Active {
		fill, text = placementBoxActiveFill, placementBoxActiveText
	}
	native.DrawRectFilled(x0, y0, x0+w, y0+h, fill)
	drawPlacementBoxBorder(x0, y0, x0+w, y0+h)
	native.DrawText(x0+placementBoxPadX, y0+placementBoxPadY, label, text)
	if f.Locked {
		drawPadlock(x0+w-placementBoxPadX-native.TextLineHeight()*0.55, y0+placementBoxPadY, native.TextLineHeight())
	}
}

// drawPlacementBoxBorder outlines a box so a light value box still reads against light geometry.
func drawPlacementBoxBorder(x0, y0, x1, y1 float32) {
	native.DrawLine(x0, y0, x1, y0, placementBoxBorder, 1)
	native.DrawLine(x1, y0, x1, y1, placementBoxBorder, 1)
	native.DrawLine(x1, y1, x0, y1, placementBoxBorder, 1)
	native.DrawLine(x0, y1, x0, y0, placementBoxBorder, 1)
}

// drawPadlock paints a small padlock at (x,y) sized to a text line: a filled body with a
// three-segment shackle above it. It marks an input whose value the user locked, so the drag no
// longer changes it. Drawn from primitives rather than an icon so it scales with the font and
// needs no asset pipeline.
func drawPadlock(x, y, h float32) {
	body := h * 0.55
	top := y + h - body
	native.DrawRectFilled(x, top, x+body, y+h, padlockColor)
	shackle := body * 0.35
	native.DrawLine(x+body*0.2, top, x+body*0.2, top-shackle, padlockColor, 1.2)
	native.DrawLine(x+body*0.2, top-shackle, x+body*0.8, top-shackle, padlockColor, 1.2)
	native.DrawLine(x+body*0.8, top-shackle, x+body*0.8, top, padlockColor, 1.2)
}
