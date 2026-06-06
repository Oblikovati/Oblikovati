//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"
	"strconv"

	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/model/feature"
)

// The Loft flow in the head: while the Loft tool runs, a modeless options window shows the
// picked cross-sections, the output operation, a closed-loop toggle, and the start/end-section
// conditions (Free / Angle / Direction — the takeoff that curves a two-section loft), then
// OK/Cancel. The picked sections are outlined by the tool's preview.
var loftUI = struct {
	open        bool
	first, last loftEndUI
}{}

// loftEndUI is the editable degree-state for one end condition (imgui needs stable field
// pointers across frames, so the dialog edits this and pushes it to the tool each frame).
type loftEndUI struct {
	cond     int // index into loftCondLabels: 0 Free, 1 Angle, 2 Direction
	angleDeg float32
	impact   float32
	reversed bool
}

var loftCondLabels = []string{"Free", "Angle", "Direction"}

// drawLoftDialog shows the loft options window while the Loft tool is active.
func drawLoftDialog(s *app.Session) {
	l := s.ActiveLoft()
	if l == nil {
		loftUI.open = false
		return
	}
	if !loftUI.open { // entering the tool: seed the condition editors from the tool's state
		loftUI.first = seedLoftEndUI(l.FirstCondition())
		loftUI.last = seedLoftEndUI(l.LastCondition())
	}
	loftUI.open = true
	native.SetNextWindowSize(320, 380)
	if native.Begin("Loft") {
		native.Text("Sections: " + strconv.Itoa(len(l.Sections())) + " (click regions in order)")
		loftOperationCombo(l)
		closed := l.Closed()
		if native.Checkbox("Closed loop", &closed) {
			l.SetClosed(closed)
		}
		if !closed { // a closed loft has no end sections, so conditions don't apply
			native.Separator()
			drawLoftEndCondition("Start section", &loftUI.first)
			drawLoftEndCondition("End section", &loftUI.last)
			l.SetFirstCondition(loftUI.first.toEnd())
			l.SetLastCondition(loftUI.last.toEnd())
		}
		drawLoftButtons(s, l)
	}
	native.End()
}

// drawLoftEndCondition renders one end's condition combo plus, for an angle/direction takeoff,
// its angle (degrees), impact (takeoff weight) and reversed flag.
func drawLoftEndCondition(title string, u *loftEndUI) {
	native.Text(title)
	if native.BeginCombo(title+"##cond", loftCondLabels[u.cond]) {
		for i, lbl := range loftCondLabels {
			if native.Selectable(lbl, i == u.cond) {
				u.cond = i
			}
		}
		native.EndCombo()
	}
	if u.cond == 0 { // Free: no further controls
		return
	}
	native.Text("  Angle (deg)")
	native.InputFloat(title+"##angle", &u.angleDeg)
	native.Text("  Impact")
	native.InputFloat(title+"##impact", &u.impact)
	rev := u.reversed
	if native.Checkbox(title+" reversed", &rev) {
		u.reversed = rev
	}
}

func loftOperationCombo(l *app.LoftTool) {
	preview := "New Solid"
	for _, o := range extrudeOperations {
		if o.op == l.Operation() {
			preview = o.label
		}
	}
	if native.BeginCombo("Output", preview) {
		for _, o := range extrudeOperations {
			if native.Selectable(o.label, o.op == l.Operation()) {
				l.SetOperation(o.op)
			}
		}
		native.EndCombo()
	}
}

func drawLoftButtons(s *app.Session, l *app.LoftTool) {
	native.BeginDisabled(!l.CanCommit())
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}

// seedLoftEndUI builds the degree-state editor for an end condition (impact defaults to 1).
func seedLoftEndUI(e feature.LoftEnd) loftEndUI {
	impact := float32(e.Impact)
	if impact <= 0 {
		impact = 1
	}
	return loftEndUI{cond: loftCondIndex(e.Condition), angleDeg: float32(e.Angle * 180 / math.Pi), impact: impact, reversed: e.Reversed}
}

// toEnd converts the degree-state editor back into a feature.LoftEnd (degrees → radians).
func (u loftEndUI) toEnd() feature.LoftEnd {
	return feature.LoftEnd{
		Condition: loftCondAt(u.cond),
		Angle:     float64(u.angleDeg) * math.Pi / 180,
		Impact:    float64(u.impact),
		Reversed:  u.reversed,
	}
}

func loftCondAt(i int) feature.LoftCondition {
	switch i {
	case 1:
		return feature.LoftAngle
	case 2:
		return feature.LoftDirection
	default:
		return feature.LoftFree
	}
}

func loftCondIndex(c feature.LoftCondition) int {
	switch c {
	case feature.LoftAngle:
		return 1
	case feature.LoftDirection:
		return 2
	default:
		return 0
	}
}
