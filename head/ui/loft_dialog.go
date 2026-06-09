//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// The Loft flow in the head: while the Loft tool runs, a modeless options window shows the
// picked cross-sections, the output operation, a closed-loop toggle, and the start/end-section
// conditions (Free / Angle / Direction — the takeoff that curves a two-section loft), then
// OK/Cancel. The picked sections are outlined by the tool's preview.
var loftUI = struct {
	open        bool
	first, last loftEndUI
	areaMid     float32 // area-graph mid-height area scale (1 = off)
}{}

// loftGuideLabels are the path-pick routing choices (rails / centerline / map curves).
var loftGuideLabels = []string{"Rails", "Centerline", "Map curves"}

// loftEndUI is the editable degree-state for one end condition (imgui needs stable field
// pointers across frames, so the dialog edits this and pushes it to the tool each frame).
type loftEndUI struct {
	cond     int // index into loftCondLabels: 0 Free, 1 Angle, 2 Direction
	angleDeg float32
	impact   float32
	reversed bool
}

// Condition combo entries: 0 Free, 1 Angle, 2 Direction (profile takeoff), 3 Sharp,
// 4 Tangent to plane (point/apex sections), 5 Tangent, 6 Smooth (face sections).
var loftCondLabels = []string{"Free", "Angle", "Direction", "Sharp", "Tangent to plane", "Tangent", "Smooth"}

// drawLoftDialog shows the loft options window while the Loft tool is active.
func drawLoftDialog(s *app.Session) {
	l := s.ActiveLoft()
	if l == nil {
		loftUI.open = false
		return
	}
	refreshLoftUI(l)
	native.SetNextWindowSize(340, 460)
	if native.Begin("Loft") {
		native.Text("Sections: " + strconv.Itoa(l.SectionCount()) + " (regions, or a vertex/point for an apex, or a face for tangency)")
		drawLoftGuides(l)
		loftOperationCombo(l)
		closed := l.Closed()
		if native.Checkbox("Closed loop", &closed) {
			l.SetClosed(closed)
		}
		drawOpenLoftConditions(l, closed)
		drawLoftButtons(s, l)
	}
	native.End()
}

func refreshLoftUI(l *app.LoftTool) {
	if loftUI.open {
		return
	}
	loftUI.first = seedLoftEndUI(l.FirstCondition())
	loftUI.last = seedLoftEndUI(l.LastCondition())
	loftUI.areaMid = float32(l.AreaMidScale())
	if loftUI.areaMid == 0 {
		loftUI.areaMid = 1
	}
	loftUI.open = true
}

func drawOpenLoftConditions(l *app.LoftTool, closed bool) {
	if closed {
		return
	}
	native.Separator()
	drawLoftEndCondition("Start section", &loftUI.first)
	drawLoftEndCondition("End section", &loftUI.last)
	l.SetFirstCondition(loftUI.first.toEnd())
	l.SetLastCondition(loftUI.last.toEnd())
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
	if u.cond == 0 || u.cond == 3 { // Free / Sharp: no further controls (sharp = a straight apex)
		return
	}
	if u.cond == 1 || u.cond == 2 { // Angle / Direction: takeoff angle on a profile section
		native.Text("  Angle (deg)")
		native.InputFloat(title+"##angle", &u.angleDeg)
	}
	native.Text("  Impact") // weight for angle/direction and the tangent-to-plane dome
	native.InputFloat(title+"##impact", &u.impact)
	rev := u.reversed
	if native.Checkbox(title+" reversed", &rev) {
		u.reversed = rev
	}
}

// drawLoftGuides shows the guide controls: which kind a picked open path becomes (rails /
// centerline / map curves), the active kind's count/status, and the area-graph mid scale.
func drawLoftGuides(l *app.LoftTool) {
	kind := l.GuideKind()
	if native.BeginCombo("Guide path", loftGuideLabels[kind]) {
		for i, lbl := range loftGuideLabels {
			if native.Selectable(lbl, i == kind) {
				l.SetGuideKind(i)
			}
		}
		native.EndCombo()
	}
	switch kind {
	case 1: // centerline
		status := "none"
		if l.HasCenterline() {
			status = "set"
		}
		native.Text("  Centerline: " + status + " (click an open path)")
	case 2: // map curves
		native.Text("  Map curves: " + strconv.Itoa(l.MapCurveCount()) + " (a path of anchors, one per section)")
	default: // rails
		native.Text("  Rails: " + strconv.Itoa(l.RailCount()) + " (click open paths to guide)")
	}
	native.Text("Area-graph mid scale (1 = off)")
	native.InputFloat("##loft-area-mid", &loftUI.areaMid)
	l.SetAreaMidScale(float64(loftUI.areaMid))
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
	case 3:
		return feature.LoftSharpPoint
	case 4:
		return feature.LoftTangentToPlane
	case 5:
		return feature.LoftTangent
	case 6:
		return feature.LoftSmooth
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
	case feature.LoftSharpPoint:
		return 3
	case feature.LoftTangentToPlane:
		return 4
	case feature.LoftTangent:
		return 5
	case feature.LoftSmooth:
		return 6
	default:
		return 0
	}
}
