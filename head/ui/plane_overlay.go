//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/renderer"
)

// planesOverlay draws the part's origin work planes as finite squares (border +
// diagonals) in the viewport, so the user can see and click them to choose a sketch
// host. The selected plane is highlighted; the plane under the cursor is shown in the
// hover color (the focus the user is about to pick). Shown outside the sketch
// environment.
func planesOverlay(part *compdef.PartComponentDefinition, selected, hovered *feature.WorkPlane) []renderer.DrawItem {
	if part == nil {
		return nil
	}
	var items []renderer.DrawItem
	for _, wp := range part.OriginPlanes() {
		items = append(items, planeBorder(wp, planeColor(wp, selected, hovered)))
	}
	return items
}

// planeColor chooses a plane's draw color: selected wins, then hovered, then faint.
func planeColor(wp, selected, hovered *feature.WorkPlane) [4]float32 {
	switch wp {
	case selected:
		return selectedPlaneColor
	case hovered:
		return hoverPlaneColor
	default:
		return faintPlaneColor
	}
}

var (
	faintPlaneColor    = [4]float32{0.45, 0.5, 0.6, 1}
	hoverPlaneColor    = [4]float32{1, 0.7, 0.2, 1}
	selectedPlaneColor = [4]float32{0.3, 0.85, 1, 1}
)

// planeBorder builds the border-and-diagonals line item of a work plane's display
// square, mapped from the plane's 2D frame into model space.
func planeBorder(wp *feature.WorkPlane, color [4]float32) renderer.DrawItem {
	pl := wp.Plane()
	s := wp.DisplaySize()
	corners2D := []math.Point2{math.P2(-s, -s), math.P2(s, -s), math.P2(s, s), math.P2(-s, s)}
	pos := make([]math.Point3, len(corners2D))
	for i, c := range corners2D {
		pos[i] = pl.ToModel(c)
	}
	// Border 0-1-2-3-0 plus the two diagonals 0-2 and 1-3.
	idx := []int{0, 1, 1, 2, 2, 3, 3, 0, 0, 2, 1, 3}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: idx, Color: color}
}
