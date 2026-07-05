//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// The Extrude flow in the head: while the Extrude tool runs, a modeless property panel
// (the reference panel layout: collapsible Input Geometry / Behavior / Output /
// Advanced Properties sections over a label/control grid) drives the tool, and the
// picked profile is highlighted in the viewport with a live prism preview. Distances
// are in the document's length unit (e.g. mm), the taper in its angle unit (e.g. deg).

// extrudeUI holds the panel's editable fields across frames, keyed by the tool they
// were seeded from — so switching straight from one extrude (or one feature edit) to
// another reseeds instead of carrying stale values into the new tool.
var extrudeUI = struct {
	distance, second, taper float32
	seeded                  *app.ExtrudeTool
}{distance: 10}

// drawExtrudeDialog shows the Extrusion property panel while the Extrude tool is
// active — creating a feature or re-editing a committed one (the same panel serves
// both) — syncing every control with the tool each frame; OK commits, Cancel aborts.
func drawExtrudeDialog(s *app.Session) {
	ext := s.ActiveExtrude()
	if ext == nil {
		extrudeUI.seeded = nil
		return
	}
	if extrudeUI.seeded != ext { // tool just opened — seed the fields from its state
		seedExtrudeFields(s)
		extrudeUI.seeded = ext
	}
	dialogSizeOnce(340, 430)
	if native.Begin("Extrusion") {
		drawExtrudeHeader(ext)
		drawExtrudeInputGeometry(ext)
		drawExtrudeBehavior(s, ext)
		drawExtrudeOutput(ext)
		drawExtrudeAdvanced(s)
		native.Separator()
		drawCommitCancelButtons(s, ext.CanCommit())
	}
	native.End()
}

// seedExtrudeFields copies the tool's current distance/second/taper into the panel fields.
func seedExtrudeFields(s *app.Session) {
	if d := s.ExtrudeDistanceDisplay(); d > 0 {
		extrudeUI.distance = float32(d)
	}
	extrudeUI.second = float32(s.ExtrudeSecondDistanceDisplay())
	extrudeUI.taper = float32(s.ExtrudeTaperDisplay())
}

// drawExtrudeHeader renders the panel breadcrumb — the feature name and, once a profile
// is picked, the sketch it extrudes (the reference panel's "Extrusion > Sketch" trail).
// Re-editing a committed extrude names that feature instead.
func drawExtrudeHeader(ext *app.ExtrudeTool) {
	title := "Extrusion"
	if name := ext.EditingName(); name != "" {
		title = name
	}
	drawFeatureBreadcrumb(title, ext.SourceSketchName())
}

// drawExtrudeInputGeometry is the Input Geometry section: the Profiles selection chip
// (required — red prompt until a region is picked, accent count + clear once picked)
// and the informational From row naming the source sketch plane.
func drawExtrudeInputGeometry(ext *app.ExtrudeTool) {
	if !propertySection("Input Geometry") {
		return
	}
	propertyRow("Profiles")
	if propertySelectorChip("extrude-profiles", extrudeProfileChipText(ext), len(ext.PickedProfiles()) > 0, true) {
		ext.ClearProfiles()
	}
	native.SetItemTooltip("Click a region in the viewport (Ctrl+click to add more)")
	propertyRow("From")
	drawExtrudeFromChip(ext)
}

// extrudeProfileChipText is the Profiles chip caption: the pick prompt, or the count.
func extrudeProfileChipText(ext *app.ExtrudeTool) string {
	return countChipText(len(ext.PickedProfiles()), "Profile", "Select Profile")
}

// drawExtrudeFromChip names the plane the extrusion starts from. Phase A always starts
// on the profile's sketch plane, so the chip is informational: the sketch name once
// known, a neutral placeholder before any pick.
func drawExtrudeFromChip(ext *app.ExtrudeTool) {
	name := ext.SourceSketchName()
	if name == "" {
		name = "Sketch Plane"
	}
	propertySelectorChip("extrude-from", name, false, false)
}

// extrudeDirectionToggles / extrudeExtentToggles / extrudeBooleanToggles pair each
// toggle's icon with its tooltip, in the reference panel's button order.
var extrudeDirectionToggles = propertyToggleSet{
	keys: []string{"direction-default", "direction-flipped", "direction-symmetric", "direction-asymmetric"},
	tips: []string{
		"Default — extrude along the sketch normal",
		"Flipped — extrude opposite the sketch normal",
		"Symmetric — extrude half the distance each way",
		"Asymmetric — extrude a separate distance each way",
	},
}

var extrudeExtentToggles = propertyToggleSet{
	keys: []string{"extent-distance", "extent-through-all", "extent-to-next", "extent-to-face"},
	tips: []string{
		"Distance — extrude exactly Distance A",
		"Through All — extrude through all existing material",
		"To Next — extrude up to the next face",
		"To Face — extrude up to a face or work plane you then pick",
	},
}

// extrudeExtentTypes lists the extents in the extent toggle row's order.
var extrudeExtentTypes = []feature.ExtentType{
	feature.DistanceExtent, feature.ThroughAllExtent, feature.ToNextExtent, feature.ToFaceExtent,
}

// drawExtrudeBehavior is the Behavior section: the direction toggle row, the Distance A
// field with the extent toggles beside it, and the Distance B field while asymmetric.
func drawExtrudeBehavior(s *app.Session, ext *app.ExtrudeTool) {
	if !propertySection("Behavior") {
		return
	}
	propertyRow("Direction")
	d := extrudeDirectionToggles
	if i := propertyIconToggles("extrude-direction", d.keys, d.tips, extrudeDirectionIndex(ext)); i >= 0 {
		applyExtrudeDirection(ext, i)
	}
	drawExtrudeDistanceRow(s, ext)
	if ext.ExtentType() == feature.DistanceExtent && ext.Asymmetric() {
		drawExtrudeSecondDistanceRow(s)
	}
}

// extrudeDirectionIndex maps the tool's direction state onto the toggle row: the
// asymmetric two-distance mode is the fourth toggle, the single-direction modes the
// first three.
func extrudeDirectionIndex(ext *app.ExtrudeTool) int {
	if ext.Asymmetric() {
		return 3
	}
	switch ext.Direction() {
	case feature.NegativeDir:
		return 1
	case feature.SymmetricDir:
		return 2
	default:
		return 0
	}
}

// applyExtrudeDirection writes one direction toggle back to the tool. The asymmetric
// toggle turns on the two-distance mode (direction is implied by the two depths); any
// single-direction toggle turns it off again.
func applyExtrudeDirection(ext *app.ExtrudeTool, index int) {
	if index == 3 {
		ext.SetAsymmetric(true)
		ext.SetDirection(feature.PositiveDir)
		return
	}
	ext.SetAsymmetric(false)
	ext.SetDirection([]feature.ExtentDirection{feature.PositiveDir, feature.NegativeDir, feature.SymmetricDir}[index])
}

// drawExtrudeDistanceRow renders Distance A (greyed when the extent is measured from
// the model instead) with the extent toggle row beside it, like the reference panel's
// distance row of termination buttons.
func drawExtrudeDistanceRow(s *app.Session, ext *app.ExtrudeTool) {
	propertyRow("Distance A")
	native.BeginDisabled(ext.ExtentType() != feature.DistanceExtent)
	native.SetNextItemWidth(propertyFieldWidth)
	parameterField(s, "extrude-distance", s.LengthUnitName(), s.LengthPrecision(), paramLength, &extrudeUI.distance)
	s.SetExtrudeDistanceDisplay(float64(extrudeUI.distance))
	native.EndDisabled()
	native.SameLine()
	e := extrudeExtentToggles
	if i := propertyIconToggles("extrude-extent", e.keys, e.tips, extrudeExtentIndex(ext)); i >= 0 {
		ext.SetExtentType(extrudeExtentTypes[i])
	}
}

// extrudeExtentIndex maps the tool's extent type onto the extent toggle row.
func extrudeExtentIndex(ext *app.ExtrudeTool) int {
	for i, t := range extrudeExtentTypes {
		if t == ext.ExtentType() {
			return i
		}
	}
	return 0
}

// drawExtrudeSecondDistanceRow renders Distance B, the asymmetric mode's second-
// direction depth.
func drawExtrudeSecondDistanceRow(s *app.Session) {
	propertyRow("Distance B")
	native.SetNextItemWidth(propertyFieldWidth)
	parameterField(s, "extrude-second", s.LengthUnitName(), s.LengthPrecision(), paramLength, &extrudeUI.second)
	s.SetExtrudeSecondDistanceDisplay(float64(extrudeUI.second))
}

// drawExtrudeOutput is the Output section: the shared Boolean toggle row.
func drawExtrudeOutput(ext *app.ExtrudeTool) {
	if !propertySection("Output") {
		return
	}
	drawBooleanPropertyRow("extrude-boolean", ext.Operation(), ext.SetOperation)
}

// drawExtrudeAdvanced is the Advanced Properties section: the Taper A draft angle.
func drawExtrudeAdvanced(s *app.Session) {
	if !propertySection("Advanced Properties") {
		return
	}
	parameterFloatRow(s, "Taper A", "extrude-taper", paramAngle, "", &extrudeUI.taper)
	s.SetExtrudeTaperDisplay(float64(extrudeUI.taper))
}

// extrudeOperations names each boolean operation for the combo-based feature dialogs
// (Revolve, Sweep, Loft, Coil) that share the list.
var extrudeOperations = []struct {
	label string
	op    ops.PartFeatureOperation
}{{"New Solid", ops.NewBody}, {"Join", ops.Join}, {"Cut", ops.Cut}, {"Intersect", ops.Intersect}}

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

// activeToolPreviewItems returns the active tool's transient preview, drawn in the 3D view
// outside the sketch environment — the translucent solid result preview when the tool can
// draft its feature, else the legacy wireframe. The session resolves which (see
// Session.ToolPreview); nil when nothing is shown yet.
func activeToolPreviewItems(s *app.Session) []renderer.DrawItem {
	return s.ToolPreview()
}
