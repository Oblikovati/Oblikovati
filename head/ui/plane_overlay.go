//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/renderer"
)

// planesOverlay draws the part's work planes — the origin frame AND user-created datums —
// each as a translucent filled square with a solid square border (Inventor's datum-plane
// look), so a created work plane is visible and can be clicked. Hidden planes (Visibility
// toggled off) are skipped. The selected plane's border is highlighted; the plane under
// the cursor uses the hover color. Shown outside the sketch environment.
func planesOverlay(part *compdef.PartComponentDefinition, selected, hovered *feature.WorkPlane) []renderer.DrawItem {
	if part == nil {
		return nil
	}
	var items []renderer.DrawItem
	planes := part.WorkPlanes()
	for i := 0; i < planes.Count(); i++ {
		wp := planes.Item(i)
		if !wp.Visible() {
			continue
		}
		items = append(items, planeFill(wp), planeBorder(wp, planeColor(wp, selected, hovered)))
	}
	return items
}

// planeColor chooses a plane border's draw color: selected wins, then hovered, then faint.
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

// planeCorners maps a work plane's display square into model space (the four corners in
// CCW order), shared by the fill and the border.
func planeCorners(wp *feature.WorkPlane) []math.Point3 {
	pl := wp.Plane()
	s := wp.DisplaySize()
	corners2D := []math.Point2{math.P2(-s, -s), math.P2(s, -s), math.P2(s, s), math.P2(-s, s)}
	pos := make([]math.Point3, len(corners2D))
	for i, c := range corners2D {
		pos[i] = pl.ToModel(c)
	}
	return pos
}

// planeFill builds the translucent filled quad of a work plane's display square (two
// triangles). The fill color is the theme's plane-fill albedo and its alpha is the
// opacity (TokenPlaneFill), so the plane is see-through; the plane normal is supplied so
// the lit overlay pass shades it consistently.
func planeFill(wp *feature.WorkPlane) renderer.DrawItem {
	pos := planeCorners(wp)
	n := wp.Plane().Normal().AsVector()
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: pos,
		Normals:   []math.Vector3{n, n, n, n},
		Indices:   []int{0, 1, 2, 0, 2, 3},
		Color:     planeFillColor,
		Opacity:   planeFillColor[3],
	}
}

// planeBorder builds the square-border line item of a work plane's display square (the
// four edges, no diagonals), mapped from the plane's 2D frame into model space.
func planeBorder(wp *feature.WorkPlane, color [4]float32) renderer.DrawItem {
	pos := planeCorners(wp)
	idx := []int{0, 1, 1, 2, 2, 3, 3, 0} // border 0-1-2-3-0
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: idx, Color: color}
}
