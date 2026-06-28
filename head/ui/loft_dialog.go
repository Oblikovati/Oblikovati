//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// The Loft flow in the head: while the Loft tool runs, a modeless property panel drives the tool.
// It is organised as Inventor's Loft dialog is — a Curves tab (the ordered list of cross-sections,
// the guide curves, closure and output) and a Conditions tab (the start/end takeoff conditions and
// the area-graph waist) — so the flow reads the way modellers expect (#1521). The picked sections
// are outlined by the tool's preview.
var loftUI = struct {
	open            bool
	first, last     loftEndUI
	areaMid         float32 // area-graph mid-height area scale (1 = off)
	selectedSection int     // the Sections-list row highlighted for removal (-1 = none)
}{}

// loftSectionPayload tags the drag-and-drop payload that reorders the Sections list.
const loftSectionPayload = "OBK_LOFT_SECTION_ROW"

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
// 4 Tangent to plane (point/apex sections), 5 Tangent, 6 Smooth, 7 G3 (face sections).
var loftCondLabels = []string{"Free", "Angle", "Direction", "Sharp", "Tangent to plane", "Tangent", "Smooth", "G3"}

// drawLoftDialog shows the Loft property panel while the Loft tool is active.
func drawLoftDialog(s *app.Session) {
	l := s.ActiveLoft()
	if l == nil {
		loftUI.open = false
		return
	}
	refreshLoftUI(l)
	native.SetNextWindowSizeOnce(360, 520)
	if native.Begin("Loft") {
		drawFeatureBreadcrumb("Loft", "")
		if native.BeginTabBar("##loft-tabs") {
			if native.BeginTabItem("Curves") {
				drawLoftCurvesTab(l)
				native.EndTabItem()
			}
			if native.BeginTabItem("Conditions") {
				drawLoftConditionsTab(s, l)
				native.EndTabItem()
			}
			native.EndTabBar()
		}
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
	loftUI.selectedSection = -1
	loftUI.open = true
}

// drawLoftCurvesTab is Inventor's Curves tab: the ordered Sections list, the guide curves, the
// closed-loop toggle and the boolean output.
func drawLoftCurvesTab(l *app.LoftTool) {
	drawLoftSectionsList(l)
	drawLoftGuides(l)
	drawLoftCurvesOptions(l)
}

// drawLoftSectionsList is the heart of the rework: an ordered, selectable, drag-reorderable list of
// the cross-sections, one row each — replacing the old count-only chip. The order IS the blend order.
// Rows are added by clicking sections in the viewport; a row is removed from its right-click menu or
// the Remove button.
func drawLoftSectionsList(l *app.LoftTool) {
	if !propertySection("Sections") {
		return
	}
	if l.SectionCount() == 0 {
		native.Text("Click cross-sections in order in the viewport:")
		native.Text("a region, a vertex/work point for an apex, or a face for tangency.")
		return
	}
	if native.BeginChild("##loft-sections", 0, loftSectionsListHeight(l), true) {
		drawLoftSectionRows(l)
	}
	native.EndChild()
	drawLoftSectionsListButtons(l)
}

// loftSectionsListHeight sizes the list child to the row count (clamped) so a short loft does not
// leave a tall empty box and a long one scrolls instead of pushing the rest of the panel off-screen.
func loftSectionsListHeight(l *app.LoftTool) float32 { return loftListHeight(l.SectionCount()) }

// loftListHeight is the section-list child height for a row count: ~22 px per row plus padding,
// clamped at 6 visible rows (beyond which the child scrolls). Split out so it is unit-testable.
func loftListHeight(rows int) float32 {
	if rows > loftListMaxVisibleRows {
		rows = loftListMaxVisibleRows
	}
	return float32(rows)*loftListRowHeight + loftListPadding
}

const (
	loftListRowHeight      = 22
	loftListPadding        = 8
	loftListMaxVisibleRows = 6
)

// drawLoftSectionRows draws one selectable, draggable row per section. Removal is deferred until after
// the loop so the slice is not mutated mid-iteration.
func drawLoftSectionRows(l *app.LoftTool) {
	remove := -1
	for i := 0; i < l.SectionCount(); i++ {
		native.PushIDInt(i)
		if native.Selectable(loftSectionRowText(l, i), loftUI.selectedSection == i) {
			loftUI.selectedSection = i
		}
		reorderLoftSectionRow(l, i)
		if native.BeginPopupContextItem("##loft-section-menu") {
			if native.MenuItem("Remove section") {
				remove = i
			}
			native.EndPopup()
		}
		native.PopID()
	}
	if remove >= 0 {
		l.RemoveSection(remove)
		loftUI.selectedSection = -1
	}
}

// loftSectionRowText is the row caption: its position (the blend order) and its source label.
func loftSectionRowText(l *app.LoftTool, i int) string {
	return fmt.Sprintf("%d.  %s", i+1, l.SectionLabel(i))
}

// reorderLoftSectionRow wires the last-drawn row as a drag source (carrying its index) and a drop
// target that moves the dragged section to this position — the same pattern as the selection filter.
func reorderLoftSectionRow(l *app.LoftTool, i int) {
	if native.BeginDragDropSource() {
		native.SetDragDropPayloadInt(loftSectionPayload, i)
		native.Text(l.SectionLabel(i))
		native.EndDragDropSource()
	}
	if native.BeginDragDropTarget() {
		if from, ok := native.AcceptDragDropPayloadInt(loftSectionPayload); ok {
			l.MoveSection(from, i)
			loftUI.selectedSection = i
		}
		native.EndDragDropTarget()
	}
}

// drawLoftSectionsListButtons offers the explicit Remove (the selected row) and Clear all affordances
// beside the list, so removal does not depend on discovering the right-click menu.
func drawLoftSectionsListButtons(l *app.LoftTool) {
	if loftUI.selectedSection >= 0 && loftUI.selectedSection < l.SectionCount() {
		if native.Button("Remove section") {
			l.RemoveSection(loftUI.selectedSection)
			loftUI.selectedSection = -1
		}
		native.SameLine()
	}
	if native.Button("Clear all") {
		l.ClearSections()
		loftUI.selectedSection = -1
	}
}

// drawLoftGuides is the guide-curve group: the chip showing the active guide kind's picks and the
// routing combo that says what the next open-path pick becomes (a rail, the centerline, or a map
// curve). Guides are optional, so an empty chip shows the plain prompt, not the required state.
func drawLoftGuides(l *app.LoftTool) {
	if !propertySection("Guides") {
		return
	}
	drawLoftGuidesChip(l)
	propertyRow("Pick as")
	native.SetNextItemWidth(propertyFieldWidth)
	drawLoftGuideKindCombo(l)
}

// drawLoftCurvesOptions is the closure + boolean-output group on the Curves tab.
func drawLoftCurvesOptions(l *app.LoftTool) {
	if !propertySection("Options") {
		return
	}
	propertyRow("")
	closed := l.Closed()
	if native.Checkbox("Closed loop", &closed) {
		l.SetClosed(closed)
	}
	drawBooleanPropertyRow("loft-boolean", l.Operation(), l.SetOperation)
}

// drawLoftGuidesChip shows the active guide kind's pick state.
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

// drawLoftConditionsTab is Inventor's Conditions tab: the start/end takeoff conditions (how the
// surface leaves each end section) and the area-graph waist. A closed loft has no end sections.
func drawLoftConditionsTab(s *app.Session, l *app.LoftTool) {
	closed := l.Closed()
	if closed {
		native.Text("A closed loft has no end sections, so it has no end conditions.")
	} else {
		drawLoftEndConditionRows(s, "Start", "loft-start", &loftUI.first)
		drawLoftEndConditionRows(s, "End", "loft-end", &loftUI.last)
		l.SetFirstCondition(loftUI.first.toEnd())
		l.SetLastCondition(loftUI.last.toEnd())
	}
	native.Separator()
	propertyFloatRow("Area Mid", "loft-area-mid", "× (1 = off)", &loftUI.areaMid)
	l.SetAreaMidScale(float64(loftUI.areaMid))
}

// drawLoftEndConditionRows renders one end's condition combo plus, for an angle/
// direction takeoff, its angle (degrees), impact (takeoff weight) and reversed flag.
func drawLoftEndConditionRows(s *app.Session, title, id string, u *loftEndUI) {
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
	drawLoftEndConditionParams(s, id, u)
}

// drawLoftEndConditionParams renders the condition's dependent fields: nothing for
// Free/Sharp, a takeoff angle for Angle/Direction, and the impact weight + reversed
// flag for every shaped condition.
func drawLoftEndConditionParams(s *app.Session, id string, u *loftEndUI) {
	if u.cond == 0 || u.cond == 3 { // Free / Sharp: no further controls (sharp = a straight apex)
		return
	}
	if u.cond == 1 || u.cond == 2 { // Angle / Direction: takeoff angle on a profile section
		angleDegRow(s, "  Angle", id+"-angle", &u.angleDeg)
	}
	propertyFloatRow("  Impact", id+"-impact", "", &u.impact)
	propertyRow("")
	rev := u.reversed
	if native.Checkbox("Reversed##"+id, &rev) {
		u.reversed = rev
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
	case 7:
		return feature.LoftG3
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
	case feature.LoftG3:
		return 7
	default:
		return 0
	}
}
