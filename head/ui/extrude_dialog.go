//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
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
		_, picked := ext.PickedProfile()
		if !picked {
			native.Text("Click a sketch profile to extrude")
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

// extrudeProfileHighlight outlines the profile picked for extrude (and its holes) in the
// selection color, so the user sees what they clicked. Returns nil when no extrude tool
// is active or no profile is picked yet.
func extrudeProfileHighlight(s *app.Session) []renderer.DrawItem {
	ext := s.ActiveExtrude()
	if ext == nil {
		return nil
	}
	ph, ok := ext.PickedProfile()
	if !ok || ph.ProfileIndex >= ph.Sketch.Profiles().Count() {
		return nil
	}
	prof := ph.Sketch.Profiles().Item(ph.ProfileIndex)
	acc := &segAccum{}
	acc.polyline(ph.Sketch.Plane(), prof.OuterLoop().Polygon(), true)
	for _, hole := range prof.InnerLoops() {
		acc.polyline(ph.Sketch.Plane(), hole.Polygon(), true)
	}
	return appendGrid(nil, acc, sketchSelectedColor)
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
