//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// planesOverlay draws a model's work planes — the origin frame AND user-created datums — each as a
// translucent filled square with a solid square border (the datum-plane look), so a created work
// plane is visible and can be clicked. Hidden planes (Visibility toggled off) are skipped. The
// selected plane's border is highlighted; the plane under the cursor uses the hover color. Shown
// outside the sketch environment. Takes the plane collection (not a part) so it serves a part or an
// assembly — both expose WorkPlanes() through their work geometry.
func planesOverlay(planes *feature.WorkPlanes, selected, hovered *feature.WorkPlane, hidden scopeFilter) []renderer.DrawItem {
	if planes == nil {
		return nil
	}
	var items []renderer.DrawItem
	for i := 0; i < planes.Count(); i++ {
		wp := planes.Item(i)
		if !wp.Visible() || hidden(wp.Seq()) {
			continue
		}
		items = append(items, planeFill(wp), planeBorder(wp, planeColor(wp, selected, hovered)))
	}
	return items
}

func axesOverlay(axes *feature.WorkAxes, selected *feature.WorkAxis, hidden scopeFilter) []renderer.DrawItem {
	if axes == nil {
		return nil
	}
	items := make([]renderer.DrawItem, 0, axes.Count())
	for i := 0; i < axes.Count(); i++ {
		axis := axes.Item(i)
		if axis.Visible() && !hidden(axis.Seq()) {
			items = append(items, axisLine(axis, axisColor(axis, selected)))
		}
	}
	return items
}

// scopeFilter reports whether a node with the given creation stamp is hidden by an active
// edit (created after the edited node). It is [app.Session.EditScopeHides], passed in so the
// overlay stays decoupled from the session type.
type scopeFilter func(seq uint64) bool

func axisColor(axis, selected *feature.WorkAxis) [4]float32 {
	if axis == selected {
		return chromeTheme.selectedPlaneColor
	}
	return chromeTheme.faintPlaneColor
}

// planeColor chooses a plane border's draw color: selected wins, then hovered, then faint.
func planeColor(wp, selected, hovered *feature.WorkPlane) [4]float32 {
	switch wp {
	case selected:
		return chromeTheme.selectedPlaneColor
	case hovered:
		return chromeTheme.hoverPlaneColor
	default:
		return chromeTheme.faintPlaneColor
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
		Color:     chromeTheme.planeFillColor,
		Opacity:   chromeTheme.planeFillColor[3],
		Biased:    true, // a reference overlay: depth-bias it so a coplanar body face wins (no z-fight)
	}
}

// planeBorder builds the square-border line item of a work plane's display square (the
// four edges, no diagonals), mapped from the plane's 2D frame into model space.
func planeBorder(wp *feature.WorkPlane, color [4]float32) renderer.DrawItem {
	pos := planeCorners(wp)
	idx := []int{0, 1, 1, 2, 2, 3, 3, 0} // border 0-1-2-3-0
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: idx, Color: color}
}

func axisLine(axis *feature.WorkAxis, color [4]float32) renderer.DrawItem {
	const halfLen = 4.0
	dir := axis.Direction().AsVector().Scale(halfLen)
	pos := []math.Point3{axis.Origin().TranslateBy(dir.Scale(-1)), axis.Origin().TranslateBy(dir)}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: []int{0, 1}, Color: color, OnTop: true}
}
