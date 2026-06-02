//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/renderer"
)

// The Extrude flow in the head: while the Extrude tool runs, a small dialog lets the
// user type a distance and OK/Cancel, and the picked profile is highlighted in the
// viewport (with a live prism preview once a distance is set). Without this the tool
// could capture a profile but never set a distance, so OK stayed disabled and extrude
// looked broken. The distance is in the document's length unit (e.g. mm).

// extrudeUI holds the dialog's distance field across frames and whether the dialog was
// open last frame (so it seeds the field once when the tool opens).
var extrudeUI = struct {
	distance float32
	open     bool
}{distance: 10}

// drawExtrudeDialog shows the distance editor while the Extrude tool is active and keeps
// the tool's distance in sync with the field each frame; OK commits, Cancel aborts.
func drawExtrudeDialog(s *app.Session) {
	ext := s.ActiveExtrude()
	if ext == nil {
		extrudeUI.open = false
		return
	}
	if !extrudeUI.open { // tool just opened — seed the field from its current distance
		if d := s.ExtrudeDistanceDisplay(); d > 0 {
			extrudeUI.distance = float32(d)
		}
		extrudeUI.open = true
	}
	native.SetNextWindowSize(280, 132)
	if native.Begin("Extrude") {
		if n := len(ext.PickedProfiles()); n == 0 {
			native.Text("Click a region to extrude (Ctrl+click to add more)")
		} else if n > 1 {
			native.Text("Extruding " + strconv.Itoa(n) + " regions")
		}
		native.Text("Distance (" + s.LengthUnitName() + ")")
		native.InputFloat("##extrude-distance", &extrudeUI.distance)
		s.SetExtrudeDistanceDisplay(float64(extrudeUI.distance)) // keep the tool in sync
		native.BeginDisabled(!ext.CanCommit())
		if native.Button("OK") {
			_ = s.OK() // a failed commit (e.g. open profile) keeps the tool open
		}
		native.EndDisabled()
		native.SameLine()
		if native.Button("Cancel") {
			s.CancelTool()
		}
	}
	native.End()
}

// extrudeProfileHighlight outlines every region picked for extrude (and their holes) in
// the selection color, so the user sees what they have gathered. Returns nil when no
// extrude tool is active or nothing is picked yet.
func extrudeProfileHighlight(s *app.Session) []renderer.DrawItem {
	ext := s.ActiveExtrude()
	if ext == nil {
		return nil
	}
	var items []renderer.DrawItem
	for _, ph := range ext.PickedProfiles() {
		items = append(items, profileOutline(ph, sketchSelectedColor)...)
	}
	return items
}

// extrudeHoverHighlight outlines the region under the cursor in the candidate color
// while the Extrude tool is active, so the user sees which closed area a click would
// pick. Returns nil when no extrude tool is active, the cursor is off any region, or
// that region is already selected (it is then already drawn in the selected color).
func extrudeHoverHighlight(s *app.Session) []renderer.DrawItem {
	ext := s.ActiveExtrude()
	if ext == nil || !native.IsItemHovered() {
		return nil
	}
	x, y := viewportCursor()
	sel, ok := s.PickAt(x, y, app.NewSelectionFilter(app.SelectProfile))
	if !ok {
		return nil
	}
	ph, isProfile := sel.(app.ProfileHandle)
	if !isProfile || isPickedProfile(ext, ph) {
		return nil
	}
	return profileOutline(ph, sketchCandidateColor)
}

// profileOutline returns the wireframe of a region's outer loop and holes in color.
// Returns nil when the region index is stale (sketch edited under the selection).
func profileOutline(ph app.ProfileHandle, color [4]float32) []renderer.DrawItem {
	if ph.ProfileIndex >= ph.Sketch.Profiles().Count() {
		return nil
	}
	prof := ph.Sketch.Profiles().Item(ph.ProfileIndex)
	acc := &segAccum{}
	acc.polyline(ph.Sketch.Plane(), prof.OuterLoop().Polygon(), true)
	for _, hole := range prof.InnerLoops() {
		acc.polyline(ph.Sketch.Plane(), hole.Polygon(), true)
	}
	return appendGrid(nil, acc, color)
}

// isPickedProfile reports whether ph is already in the tool's selection.
func isPickedProfile(ext *app.ExtrudeTool, ph app.ProfileHandle) bool {
	for _, p := range ext.PickedProfiles() {
		if p == ph {
			return true
		}
	}
	return false
}

// activeToolPreviewItems returns the active tool's transient preview (the Extrude prism
// wireframe), drawn in the 3D view outside the sketch environment. Nil when the tool has
// no preview or nothing to show yet.
func activeToolPreviewItems(s *app.Session) []renderer.DrawItem {
	ti := s.ActiveTool()
	if ti == nil {
		return nil
	}
	pv, ok := ti.Tool().(app.Previewable)
	if !ok {
		return nil
	}
	return pv.Preview(s)
}
