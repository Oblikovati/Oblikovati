//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Hole flow in the head: while the Hole tool runs, a modeless options window lets
// the user set the diameter and depth (database units), then OK/Cancel. The placement
// face is the one clicked in the viewport.
var holeUI = struct {
	diameter, depth          float32
	through                  bool
	counterbore, countersink bool
	cDiameter, cDepth        float32
	sinkAngleDeg             float32
	pointAngleDeg            float32
	open                     bool
}{diameter: 1, depth: 2, cDiameter: 2, cDepth: 0.5, sinkAngleDeg: 90, pointAngleDeg: 118}

// drawHoleDialog shows the hole options window while the Hole tool is active.
func drawHoleDialog(s *app.Session) {
	h := s.ActiveHole()
	if h == nil {
		holeUI.open = false
		return
	}
	if !holeUI.open {
		holeUI.diameter = float32(h.Diameter())
		holeUI.depth = float32(h.Depth())
		holeUI.through = h.ThroughAll()
		holeUI.counterbore = h.Counterbore()
		holeUI.countersink = h.Countersink()
		holeUI.cDiameter = float32(h.CounterDiameter())
		holeUI.cDepth = float32(h.CounterDepth())
		holeUI.sinkAngleDeg = float32(h.SinkAngle() * 180 / stdmath.Pi)
		holeUI.pointAngleDeg = float32(h.PointAngle() * 180 / stdmath.Pi)
		holeUI.open = true
	}
	native.SetNextWindowSize(300, 390)
	if native.Begin("Hole") {
		if _, ok := h.PickedFace(); !ok {
			native.Text("Click a planar face to place the hole on")
		}
		native.Text("Diameter (" + s.LengthUnitName() + ")")
		native.InputFloat("##hole-diameter", &holeUI.diameter)
		h.SetDiameter(float64(holeUI.diameter))
		native.Checkbox("Through All", &holeUI.through)
		h.SetThroughAll(holeUI.through)
		native.BeginDisabled(holeUI.through) // depth is ignored when drilling through
		native.Text("Depth (" + s.LengthUnitName() + ")")
		native.InputFloat("##hole-depth", &holeUI.depth)
		native.EndDisabled()
		h.SetDepth(float64(holeUI.depth))
		// Drill point applies to a plain blind drilled hole (not through, no recess).
		native.BeginDisabled(holeUI.through || holeUI.counterbore || holeUI.countersink)
		native.Text("Drill point angle (deg, 0 = flat)")
		native.InputFloat("##hole-pang", &holeUI.pointAngleDeg)
		h.SetPointAngle(float64(holeUI.pointAngleDeg) * stdmath.Pi / 180)
		native.EndDisabled()
		drawRecess(s, h)
		drawHoleButtons(s, h)
	}
	native.End()
}

// drawRecess renders the counterbore/countersink toggles (mutually exclusive) and their
// parameters: a counterbore's recess Ø + depth, or a countersink's sink Ø + included angle.
func drawRecess(s *app.Session, h *app.HoleTool) {
	if native.Checkbox("Counterbore", &holeUI.counterbore) {
		h.SetCounterbore(holeUI.counterbore)
		holeUI.countersink = h.Countersink() // the tool clears the other profile
	}
	native.BeginDisabled(!holeUI.counterbore)
	native.Text("Counterbore Ø (" + s.LengthUnitName() + ")")
	native.InputFloat("##hole-cdia", &holeUI.cDiameter)
	native.Text("Counterbore depth (" + s.LengthUnitName() + ")")
	native.InputFloat("##hole-cdepth", &holeUI.cDepth)
	h.SetCounterDepth(float64(holeUI.cDepth))
	native.EndDisabled()

	if native.Checkbox("Countersink", &holeUI.countersink) {
		h.SetCountersink(holeUI.countersink)
		holeUI.counterbore = h.Counterbore()
	}
	native.BeginDisabled(!holeUI.countersink)
	native.Text("Countersink Ø (" + s.LengthUnitName() + ")")
	native.InputFloat("##hole-sdia", &holeUI.cDiameter)
	native.Text("Countersink angle (deg)")
	native.InputFloat("##hole-sang", &holeUI.sinkAngleDeg)
	h.SetSinkAngle(float64(holeUI.sinkAngleDeg) * stdmath.Pi / 180)
	native.EndDisabled()

	h.SetCounterDiameter(float64(holeUI.cDiameter)) // shared recess/sink diameter
}

func drawHoleButtons(s *app.Session, h *app.HoleTool) {
	native.BeginDisabled(!h.CanCommit())
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}
