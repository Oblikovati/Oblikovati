//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Hole flow in the head: while the Hole tool runs, a modeless property panel (the
// reference panel schema: Input Geometry / Type / Behavior sections) drives the tool —
// the placement-face chip, the seat (none / counterbore / countersink) with its
// dimensions, the termination, the hole size, and the drill point — then OK/Cancel.
// Lengths are in the document's length unit, angles in degrees.
var holeUI = struct {
	diameter, depth          float32
	through                  bool
	counterbore, countersink bool
	cDiameter, cDepth        float32
	sinkAngleDeg             float32
	pointAngleDeg            float32
	seeded                   *app.HoleTool // the tool the fields were seeded from (nil = none)
}{diameter: 1, depth: 2, cDiameter: 2, cDepth: 0.5, sinkAngleDeg: 90, pointAngleDeg: 118}

// drawHoleDialog shows the Hole property panel while the Hole tool is active.
func drawHoleDialog(s *app.Session) {
	h := s.ActiveHole()
	if h == nil {
		holeUI.seeded = nil
		return
	}
	refreshHoleUI(h)
	native.SetNextWindowSizeOnce(360, 460)
	if native.Begin("Hole") {
		title := "Hole"
		if name := h.EditingName(); name != "" {
			title = name // re-editing a committed hole: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, "")
		drawHoleInputGeometry(h)
		drawHoleType(s, h)
		drawHoleBehavior(s, h)
		native.Separator()
		drawCommitCancelButtons(s, h.CanCommit())
	}
	native.End()
}

func refreshHoleUI(h *app.HoleTool) {
	if holeUI.seeded == h {
		return
	}
	holeUI.diameter = float32(h.Diameter())
	holeUI.depth = float32(h.Depth())
	holeUI.through = h.ThroughAll()
	holeUI.counterbore = h.Counterbore()
	holeUI.countersink = h.Countersink()
	holeUI.cDiameter = float32(h.CounterDiameter())
	holeUI.cDepth = float32(h.CounterDepth())
	holeUI.sinkAngleDeg = float32(h.SinkAngle() * 180 / stdmath.Pi)
	holeUI.pointAngleDeg = float32(h.PointAngle() * 180 / stdmath.Pi)
	holeUI.seeded = h
}

// drawHoleInputGeometry is the Input Geometry section: the required placement-face chip.
func drawHoleInputGeometry(h *app.HoleTool) {
	if !propertySection("Input Geometry") {
		return
	}
	propertyRow("Position")
	picked := h.HasPlacement() // a fresh pick, or the edited feature's retained face
	if propertySelectorChip("hole-position", pickChipText(picked, "1 Face", "Select Face"), picked, true) {
		h.ClearFace()
	}
	native.SetItemTooltip("Click a planar face in the viewport to place the hole on")
}

// holeSeatToggles is the Type section's Seat row (the reference panel's seat shapes we
// support: plain, counterbore, countersink).
var holeSeatToggles = propertyToggleSet{
	keys: []string{"seat-none", "seat-counterbore", "seat-countersink"},
	tips: []string{
		"None — a plain drilled hole",
		"Counterbore — a flat-bottomed recess above the hole",
		"Countersink — a conical recess above the hole",
	},
}

// drawHoleType is the Type section: the Seat toggle row and the selected seat's
// dimension rows.
func drawHoleType(s *app.Session, h *app.HoleTool) {
	if !propertySection("Type") {
		return
	}
	propertyRow("Seat")
	if i := propertyIconToggles("hole-seat", holeSeatToggles.keys, holeSeatToggles.tips, holeSeatIndex(h)); i >= 0 {
		applyHoleSeat(h, i)
	}
	drawHoleSeatParams(s, h)
}

// holeSeatIndex maps the tool's mutually-exclusive seat flags onto the Seat row.
func holeSeatIndex(h *app.HoleTool) int {
	switch {
	case h.Counterbore():
		return 1
	case h.Countersink():
		return 2
	default:
		return 0
	}
}

// applyHoleSeat writes one Seat toggle back to the tool (the tool keeps the two seat
// profiles mutually exclusive) and re-syncs the panel's flags from it.
func applyHoleSeat(h *app.HoleTool, index int) {
	h.SetCounterbore(index == 1)
	h.SetCountersink(index == 2)
	holeUI.counterbore = h.Counterbore()
	holeUI.countersink = h.Countersink()
}

// drawHoleSeatParams renders the selected seat's dimensions: a counterbore's recess
// Ø + depth, or a countersink's sink Ø + included angle.
func drawHoleSeatParams(s *app.Session, h *app.HoleTool) {
	if h.Counterbore() {
		lengthCmRow(s, "Seat Ø", "hole-cdia", &holeUI.cDiameter)
		lengthCmRow(s, "Seat Depth", "hole-cdepth", &holeUI.cDepth)
		h.SetCounterDiameter(float64(holeUI.cDiameter))
		h.SetCounterDepth(float64(holeUI.cDepth))
	}
	if h.Countersink() {
		lengthCmRow(s, "Seat Ø", "hole-sdia", &holeUI.cDiameter)
		angleDegRow(s, "Seat Angle", "hole-sang", &holeUI.sinkAngleDeg)
		h.SetCounterDiameter(float64(holeUI.cDiameter))
		h.SetSinkAngle(float64(holeUI.sinkAngleDeg) * stdmath.Pi / 180)
	}
}

// holeTerminationToggles / holeDrillPointToggles are the Behavior section's toggle rows.
var holeTerminationToggles = propertyToggleSet{
	keys: []string{"extent-distance", "extent-through-all"},
	tips: []string{
		"Distance — drill exactly Depth deep",
		"Through All — drill through all existing material",
	},
}

var holeDrillPointToggles = propertyToggleSet{
	keys: []string{"drill-flat", "drill-angle"},
	tips: []string{
		"Flat — a flat-bottomed hole",
		"Angle — a conical drill point of the given included angle",
	},
}

// drawHoleBehavior is the Behavior section: termination, hole size, and drill point.
func drawHoleBehavior(s *app.Session, h *app.HoleTool) {
	if !propertySection("Behavior") {
		return
	}
	drawHoleTerminationRow(h)
	lengthCmRow(s, "Diameter", "hole-diameter", &holeUI.diameter)
	h.SetDiameter(float64(holeUI.diameter))
	drawHoleDepthRow(s, h)
	drawHoleDrillPointRow(s, h)
}

// drawHoleTerminationRow renders the Termination toggles (distance / through-all).
func drawHoleTerminationRow(h *app.HoleTool) {
	propertyRow("Termination")
	active := 0
	if h.ThroughAll() {
		active = 1
	}
	tt := holeTerminationToggles
	if i := propertyIconToggles("hole-termination", tt.keys, tt.tips, active); i >= 0 {
		holeUI.through = i == 1
		h.SetThroughAll(holeUI.through)
	}
}

// drawHoleDepthRow renders the Depth field, greyed while drilling through everything.
func drawHoleDepthRow(s *app.Session, h *app.HoleTool) {
	native.BeginDisabled(holeUI.through)
	lengthCmRow(s, "Depth", "hole-depth", &holeUI.depth)
	native.EndDisabled()
	h.SetDepth(float64(holeUI.depth))
}

// drawHoleDrillPointRow renders the Drill Point toggles with the included-angle field
// beside them in angle mode. The row greys out when the bottom shape is moot: a
// through hole has no bottom, and a seated hole's recess owns the profile.
func drawHoleDrillPointRow(s *app.Session, h *app.HoleTool) {
	native.BeginDisabled(holeUI.through || holeUI.counterbore || holeUI.countersink)
	propertyRow("Drill Point")
	active := 0
	if h.PointAngle() > 0 {
		active = 1
	}
	dt := holeDrillPointToggles
	if i := propertyIconToggles("hole-drill", dt.keys, dt.tips, active); i >= 0 {
		applyHoleDrillPoint(h, i)
	}
	drawHolePointAngleField(s, h)
	native.EndDisabled()
}

// drawHolePointAngleField is the included-angle input shown beside the toggles while a
// conical drill point is selected.
func drawHolePointAngleField(s *app.Session, h *app.HoleTool) {
	if h.PointAngle() <= 0 {
		return
	}
	native.SameLine()
	native.SetNextItemWidth(60)
	disp := float32(s.AngleDegToDisplay(float64(holeUI.pointAngleDeg)))
	if native.InputFloat("##hole-pang", &disp) {
		holeUI.pointAngleDeg = float32(s.AngleDisplayToDeg(float64(disp)))
	}
	native.SameLine()
	native.Text(s.AngleUnitName())
	h.SetPointAngle(float64(holeUI.pointAngleDeg) * stdmath.Pi / 180)
}

// applyHoleDrillPoint writes one Drill Point toggle back to the tool: flat zeroes the
// angle; angle mode restores the panel's angle (falling back to the 118° drill default).
func applyHoleDrillPoint(h *app.HoleTool, index int) {
	if index == 0 {
		h.SetPointAngle(0)
		return
	}
	if holeUI.pointAngleDeg <= 0 {
		holeUI.pointAngleDeg = 118
	}
	h.SetPointAngle(float64(holeUI.pointAngleDeg) * stdmath.Pi / 180)
}
