//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// The Loft flow in the head: while the Loft tool runs, a modeless property panel (the
// reference panel schema) drives the tool — the cross-section and guide chips, the
// closed-loop / area-graph / end-condition behavior, and the boolean output — then
// OK/Cancel. The picked sections are outlined by the tool's preview.
var loftUI = struct {
	open        bool
	first, last loftEndUI
	areaMid     float32 // area-graph mid-height area scale (1 = off)
}{}

// loftGuideLabels are the path-pick routing choices (rails / centerline / map curves).
var loftGuideLabels = []string{"Rails", "Centerline", "Map curves"}

// loftEndUI is the editable degree-state for one end condition (imgui needs stable field
// pointers across frames, so the panel edits this and pushes it to the tool each frame).
type loftEndUI struct {
	cond     int // index into loftCondLabels: 0 Free, 1 Angle, 2 Direction
	angleDeg float32
	impact   float32
	reversed bool
}

// Condition combo entries: 0 Free, 1 Angle, 2 Direction (profile takeoff), 3 Sharp,
// 4 Tangent to plane (point/apex sections), 5 Tangent, 6 Smooth (face sections).
var loftCondLabels = []string{"Free", "Angle", "Direction", "Sharp", "Tangent to plane", "Tangent", "Smooth"}

// drawLoftDialog shows the Loft property panel while the Loft tool is active.
func drawLoftDialog(s *app.Session) {
	l := s.ActiveLoft()
	if l == nil {
		loftUI.open = false
		return
	}
	refreshLoftUI(l)
	native.SetNextWindowSizeOnce(360, 500)
	if native.Begin("Loft") {
		drawFeatureBreadcrumb("Loft", "")
		drawLoftInputGeometry(l)
		drawLoftBehavior(l)
		drawLoftOutput(l)
		native.Separator()
		drawCommitCancelButtons(s, l.CanCommit())
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

// drawLoftInputGeometry is the Input Geometry section: the required Sections chip, the
// optional Guides chip, and the routing combo that says what kind of guide the next
// open-path pick becomes.
func drawLoftInputGeometry(l *app.LoftTool) {
	if !propertySection("Input Geometry") {
		return
	}
	drawPickChipRow("Sections", "loft-sections", countChipText(l.SectionCount(), "Section", "Select Sections"),
		l.SectionCount() > 0, "Click regions in order (or a vertex/point for an apex, a face for tangency)", l.ClearSections)
	drawLoftGuidesChip(l)
	propertyRow("Pick as")
	native.SetNextItemWidth(propertyFieldWidth)
	drawLoftGuideKindCombo(l)
}

// drawLoftGuidesChip shows the active guide kind's pick state. Guides are optional, so
// an empty chip renders the plain prompt rather than the red required state.
func drawLoftGuidesChip(l *app.LoftTool) {
	propertyRow("Guides")
	text, filled := loftGuideChipState(l)
	if propertySelectorChip("loft-guides", text, filled, false) {
		l.ClearGuides()
	}
	native.SetItemTooltip("Open sketch paths guiding the loft (rails, a centerline, or map curves)")
}

// loftGuideChipState is the Guides chip caption + filled flag for the active kind.
func loftGuideChipState(l *app.LoftTool) (string, bool) {
	switch l.GuideKind() {
	case 1: // centerline
		return pickChipText(l.HasCenterline(), "1 Centerline", "Select Path"), l.HasCenterline()
	case 2: // map curves
		return countChipText(l.MapCurveCount(), "Map Curve", "Select Paths"), l.MapCurveCount() > 0
	default: // rails
		return countChipText(l.RailCount(), "Rail", "Select Paths"), l.RailCount() > 0
	}
}

func drawLoftGuideKindCombo(l *app.LoftTool) {
	kind := l.GuideKind()
	if native.BeginCombo("##loft-guide-kind", loftGuideLabels[kind]) {
		for i, lbl := range loftGuideLabels {
			if native.Selectable(lbl, i == kind) {
				l.SetGuideKind(i)
			}
		}
		native.EndCombo()
	}
}

// drawLoftBehavior is the Behavior section: the closed-loop toggle, the area-graph mid
// scale, and — for an open loft — the start/end section conditions.
func drawLoftBehavior(l *app.LoftTool) {
	if !propertySection("Behavior") {
		return
	}
	propertyRow("")
	closed := l.Closed()
	if native.Checkbox("Closed loop", &closed) {
		l.SetClosed(closed)
	}
	propertyFloatRow("Area Mid", "loft-area-mid", "× (1 = off)", &loftUI.areaMid)
	l.SetAreaMidScale(float64(loftUI.areaMid))
	drawOpenLoftConditions(l, closed)
}

func drawOpenLoftConditions(l *app.LoftTool, closed bool) {
	if closed {
		return
	}
	drawLoftEndConditionRows("Start", "loft-start", &loftUI.first)
	drawLoftEndConditionRows("End", "loft-end", &loftUI.last)
	l.SetFirstCondition(loftUI.first.toEnd())
	l.SetLastCondition(loftUI.last.toEnd())
}

// drawLoftEndConditionRows renders one end's condition combo plus, for an angle/
// direction takeoff, its angle (degrees), impact (takeoff weight) and reversed flag.
func drawLoftEndConditionRows(title, id string, u *loftEndUI) {
	propertyRow(title)
	native.SetNextItemWidth(propertyComboWidth)
	if native.BeginCombo("##"+id+"-cond", loftCondLabels[u.cond]) {
		for i, lbl := range loftCondLabels {
			if native.Selectable(lbl, i == u.cond) {
				u.cond = i
			}
		}
		native.EndCombo()
	}
	drawLoftEndConditionParams(id, u)
}

// drawLoftEndConditionParams renders the condition's dependent fields: nothing for
// Free/Sharp, a takeoff angle for Angle/Direction, and the impact weight + reversed
// flag for every shaped condition.
func drawLoftEndConditionParams(id string, u *loftEndUI) {
	if u.cond == 0 || u.cond == 3 { // Free / Sharp: no further controls (sharp = a straight apex)
		return
	}
	if u.cond == 1 || u.cond == 2 { // Angle / Direction: takeoff angle on a profile section
		propertyFloatRow("  Angle", id+"-angle", "deg", &u.angleDeg)
	}
	propertyFloatRow("  Impact", id+"-impact", "", &u.impact)
	propertyRow("")
	rev := u.reversed
	if native.Checkbox("Reversed##"+id, &rev) {
		u.reversed = rev
	}
}

// drawLoftOutput is the Output section: the shared Boolean toggle row.
func drawLoftOutput(l *app.LoftTool) {
	if !propertySection("Output") {
		return
	}
	drawBooleanPropertyRow("loft-boolean", l.Operation(), l.SetOperation)
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
