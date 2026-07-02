//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/renderer"
)

// revolveCenterlineHighlight draws the Revolve axis centerline — a sketch line, which the generic
// tool highlight (drawSelectable, B-rep only) does not cover. The chosen centerline shows in the
// selection colour; a centerline under the cursor while choosing the axis shows as a candidate.
// (The profile itself is highlighted by the generic toolHoverHighlight/toolSelectedHighlight.)
func revolveCenterlineHighlight(s *app.Session) []renderer.DrawItem {
	rv := s.ActiveRevolve()
	if rv == nil {
		return nil
	}
	var items []renderer.DrawItem
	if pts, plane, ok := rv.CenterlineOutline(); ok {
		acc := &segAccum{}
		acc.polyline(plane, pts, false)
		items = appendGrid(items, acc, chromeTheme.sketchSelectedColor)
	}
	if native.IsItemHovered() {
		x, y := viewportCursor()
		if pts, plane, ok := s.HoveredCenterlineOutline(x, y); ok {
			acc := &segAccum{}
			acc.polyline(plane, pts, false)
			items = appendGrid(items, acc, chromeTheme.sketchCandidateColor)
		}
	}
	return items
}
