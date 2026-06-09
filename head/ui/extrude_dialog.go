//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// The Extrude flow in the head: while the Extrude tool runs, a modeless options window
// (Inventor's Extrude property panel) lets the user pick the operation, the extent and
// direction, the distance(s) and a draft taper, then OK/Cancel — and the picked profile is
// highlighted in the viewport with a live prism preview. Distances are in the document's
// length unit (e.g. mm), the taper in its angle unit (e.g. deg).

// extrudeUI holds the dialog's editable fields across frames and whether the dialog was
// open last frame (so it seeds the fields once when the tool opens).
var extrudeUI = struct {
	distance, second, taper float32
	open                    bool
}{distance: 10}

// drawExtrudeDialog shows the extrude options window while the Extrude tool is active,
// syncing each control with the tool every frame; OK commits, Cancel aborts.
func drawExtrudeDialog(s *app.Session) {
	ext := s.ActiveExtrude()
	if ext == nil {
		extrudeUI.open = false
		return
	}
	if !extrudeUI.open { // tool just opened — seed the fields from its current state
		seedExtrudeFields(s)
		extrudeUI.open = true
	}
	native.SetNextWindowSize(300, 300)
	if native.Begin("Extrude") {
		drawExtrudeProfileText(ext)
		extrudeOperationCombo(ext)
		extrudeExtentCombo(ext)
		extrudeDirectionCombo(ext)
		drawExtrudeDistances(s, ext)
		drawExtrudeTaper(s)
		drawExtrudeButtons(s, ext)
	}
	native.End()
}

// seedExtrudeFields copies the tool's current distance/second/taper into the dialog fields.
func seedExtrudeFields(s *app.Session) {
	if d := s.ExtrudeDistanceDisplay(); d > 0 {
		extrudeUI.distance = float32(d)
	}
	extrudeUI.second = float32(s.ExtrudeSecondDistanceDisplay())
	extrudeUI.taper = float32(s.ExtrudeTaperDisplay())
}

func drawExtrudeProfileText(ext *app.ExtrudeTool) {
	switch n := len(ext.PickedProfiles()); {
	case n == 0:
		native.Text("Click a region to extrude (Ctrl+click to add more)")
	case n > 1:
		native.Text("Extruding " + strconv.Itoa(n) + " regions")
	}
}

// extrudeOperations / extrudeExtents / extrudeDirections name each enum value for its
// combo, paired with the value the dialog writes back to the tool.
var extrudeOperations = []struct {
	label string
	op    ops.PartFeatureOperation
}{{"New Solid", ops.NewBody}, {"Join", ops.Join}, {"Cut", ops.Cut}, {"Intersect", ops.Intersect}}

var extrudeExtents = []struct {
	label string
	t     feature.ExtentType
}{{"Distance", feature.DistanceExtent}, {"Through All", feature.ThroughAllExtent}, {"To Next", feature.ToNextExtent}}

var extrudeDirections = []struct {
	label string
	d     feature.ExtentDirection
}{{"Default", feature.PositiveDir}, {"Flipped", feature.NegativeDir}, {"Symmetric", feature.SymmetricDir}}

func extrudeOperationCombo(ext *app.ExtrudeTool) {
	preview := "New Solid"
	for _, o := range extrudeOperations {
		if o.op == ext.Operation() {
			preview = o.label
		}
	}
	if native.BeginCombo("Output", preview) {
		for _, o := range extrudeOperations {
			if native.Selectable(o.label, o.op == ext.Operation()) {
				ext.SetOperation(o.op)
			}
		}
		native.EndCombo()
	}
}

func extrudeExtentCombo(ext *app.ExtrudeTool) {
	preview := "Distance"
	for _, e := range extrudeExtents {
		if e.t == ext.ExtentType() {
			preview = e.label
		}
	}
	if native.BeginCombo("Extents", preview) {
		for _, e := range extrudeExtents {
			if native.Selectable(e.label, e.t == ext.ExtentType()) {
				ext.SetExtentType(e.t)
			}
		}
		native.EndCombo()
	}
}

func extrudeDirectionCombo(ext *app.ExtrudeTool) {
	preview := "Default"
	for _, d := range extrudeDirections {
		if d.d == ext.Direction() {
			preview = d.label
		}
	}
	if native.BeginCombo("Direction", preview) {
		for _, d := range extrudeDirections {
			if native.Selectable(d.label, d.d == ext.Direction()) {
				ext.SetDirection(d.d)
			}
		}
		native.EndCombo()
	}
}

// drawExtrudeDistances shows the distance field for distance-gauged extents, plus an
// asymmetric two-direction toggle and its second-distance field.
func drawExtrudeDistances(s *app.Session, ext *app.ExtrudeTool) {
	if ext.ExtentType() != feature.DistanceExtent {
		return
	}
	native.Text("Distance (" + s.LengthUnitName() + ")")
	native.InputFloat("##extrude-distance", &extrudeUI.distance)
	s.SetExtrudeDistanceDisplay(float64(extrudeUI.distance))
	asym := ext.Asymmetric()
	if native.Checkbox("Two directions", &asym) {
		ext.SetAsymmetric(asym)
	}
	if asym {
		native.Text("Second distance (" + s.LengthUnitName() + ")")
		native.InputFloat("##extrude-second", &extrudeUI.second)
		s.SetExtrudeSecondDistanceDisplay(float64(extrudeUI.second))
	}
}

func drawExtrudeTaper(s *app.Session) {
	native.Text("Taper (" + s.AngleUnitName() + ")")
	native.InputFloat("##extrude-taper", &extrudeUI.taper)
	s.SetExtrudeTaperDisplay(float64(extrudeUI.taper))
}

func drawExtrudeButtons(s *app.Session, ext *app.ExtrudeTool) {
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
