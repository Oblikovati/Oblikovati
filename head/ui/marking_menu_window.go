//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Tools ▸ Customize Marking Menu panel (REQ-005): environment tab selector,
// per-slot command assignment with search filter, overflow list, Restore Default,
// and radial/classic style toggle. Session owns the data and persistence (REQ-004).

// markingMenuEditorSession is the session surface the panel consumes (audit I5,
// arrowSession pattern). *app.Session satisfies it implicitly.
type markingMenuEditorSession interface {
	MarkingMenuEditorOpen() bool
	CloseMarkingMenuEditor()
	MarkingMenu(env app.Environment) wire.MarkingMenuView
	SetMarkingMenu(menu wire.MarkingMenuView) error
	ResetMarkingMenu(env app.Environment)
	SearchCommands(query string) []*app.CommandDefinition
	ClassicContextMenu() bool
	ToggleContextMenuStyle()
}

var _ markingMenuEditorSession = (*app.Session)(nil)

const mmEditorFilterLen = 64
const mmEditorNoSlot = -1

// mmEditorSlotEntry pairs a display label with a quadrant constant.
type mmEditorSlotEntry struct {
	label    string
	quadrant types.ScreenQuadrant
}

var mmEditorSlots = [8]mmEditorSlotEntry{
	{"N", types.QuadrantNorth},
	{"NE", types.QuadrantNorthEast},
	{"E", types.QuadrantEast},
	{"SE", types.QuadrantSouthEast},
	{"S", types.QuadrantSouth},
	{"SW", types.QuadrantSouthWest},
	{"W", types.QuadrantWest},
	{"NW", types.QuadrantNorthWest},
}

var mmEditorEnvLabels = [2]string{"Base", "Sketch"}
var mmEditorEnvs = [2]app.Environment{app.BaseEnvironment, app.SketchEnvironment}

// mmEditorUI holds the panel's cross-frame state: active environment tab, which
// quadrant slot (or overflow entry) is selected for assignment, command search
// filter, and the last error from SetMarkingMenu.
var mmEditorUI = struct {
	envTab  int
	slot    int  // mmEditorNoSlot or index into mmEditorSlots (0-7)
	isOver  bool // true when an overflow entry is selected instead
	overIdx int  // index into Overflow when isOver
	filter  []byte
	lastErr string
}{}

// drawMarkingMenuEditor renders the panel while it is open. The title-bar X and
// Done both close it.
func drawMarkingMenuEditor(s markingMenuEditorSession) {
	if !s.MarkingMenuEditorOpen() {
		return
	}
	if mmEditorUI.filter == nil {
		mmEditorUI.filter = make([]byte, mmEditorFilterLen)
		mmEditorUI.slot = mmEditorNoSlot
	}
	dialogSizeOnce(620, 520)
	visible, open := native.BeginClosable("Customize Marking Menu")
	if visible {
		drawMMEditorBody(s)
	}
	native.End()
	if !open {
		s.CloseMarkingMenuEditor()
	}
}

// drawMMEditorBody renders the panel: optional error line, environment tabs,
// command search section, style checkbox, and footer buttons.
func drawMMEditorBody(s markingMenuEditorSession) {
	if mmEditorUI.lastErr != "" {
		native.Text("! " + mmEditorUI.lastErr)
	}
	drawMMEditorEnvTabs(s)
	native.Separator()
	drawMMEditorSearch(s)
	native.Separator()
	classic := s.ClassicContextMenu()
	if native.Checkbox("Classic (linear) instead of radial", &classic) {
		s.ToggleContextMenuStyle()
	}
	native.Separator()
	env := mmEditorEnvs[mmEditorUI.envTab]
	if native.Button("Restore Default") {
		s.ResetMarkingMenu(env)
		mmEditorUI.slot = mmEditorNoSlot
		mmEditorUI.isOver = false
		mmEditorUI.lastErr = ""
	}
	native.SameLine()
	if native.Button("Done") {
		s.CloseMarkingMenuEditor()
	}
}

// drawMMEditorEnvTabs renders the Base/Sketch tab bar and the slot table for the
// active environment.
func drawMMEditorEnvTabs(s markingMenuEditorSession) {
	if !native.BeginTabBar("##mm-env") {
		return
	}
	for i, label := range mmEditorEnvLabels {
		if native.BeginTabItem(label) {
			if mmEditorUI.envTab != i {
				mmEditorUI.envTab = i
				mmEditorUI.slot = mmEditorNoSlot
				mmEditorUI.isOver = false
			}
			env := mmEditorEnvs[i]
			drawMMEditorSlotTable(s, env, s.MarkingMenu(env))
			native.EndTabItem()
		}
	}
	native.EndTabBar()
}

// drawMMEditorSlotTable renders the 8-slot quadrant table and overflow list for one
// environment. Clicking a row selects it for command assignment.
func drawMMEditorSlotTable(s markingMenuEditorSession, env app.Environment, menu wire.MarkingMenuView) {
	assigned := map[types.ScreenQuadrant]string{}
	for _, item := range menu.Quadrants {
		assigned[item.Quadrant] = item.CommandID
	}
	if !native.BeginTable("##mm-slots", 3, 0, 0) {
		return
	}
	native.TableSetupColumn("Slot")
	native.TableSetupColumn("Command")
	native.TableSetupColumn("")
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
	for i, entry := range mmEditorSlots {
		drawMMEditorSlotRow(s, env, i, entry, assigned[entry.quadrant])
	}
	native.EndTable()
	drawMMEditorOverflow(s, env, menu.Overflow)
}

// drawMMEditorSlotRow renders one quadrant row: slot label, assigned command, and a
// clear button. Clicking the label column selects this slot for assignment.
func drawMMEditorSlotRow(s markingMenuEditorSession, env app.Environment, i int, entry mmEditorSlotEntry, cmdID string) {
	native.TableNextRow()
	native.TableNextColumn()
	selected := !mmEditorUI.isOver && mmEditorUI.slot == i
	if native.Selectable(entry.label+"##q"+entry.label, selected) {
		mmEditorUI.slot = i
		mmEditorUI.isOver = false
	}
	native.TableNextColumn()
	if cmdID == "" {
		native.Text("(empty)")
	} else {
		native.Text(cmdID)
	}
	native.TableNextColumn()
	if cmdID != "" && native.Button("×##q"+entry.label) {
		mmEditorRemoveQuadrant(s, env, entry.quadrant)
	}
}

// drawMMEditorOverflow renders the overflow list with remove buttons. Clicking a
// command label selects it for reassignment.
func drawMMEditorOverflow(s markingMenuEditorSession, env app.Environment, overflow []string) {
	if len(overflow) == 0 {
		return
	}
	native.Separator()
	native.Text("Overflow")
	for oi, cmdID := range overflow {
		selected := mmEditorUI.isOver && mmEditorUI.overIdx == oi
		if native.Selectable(cmdID+"##ov"+fmt.Sprintf("%d", oi), selected) {
			mmEditorUI.isOver = true
			mmEditorUI.overIdx = oi
			mmEditorUI.slot = mmEditorNoSlot
		}
		native.SameLine()
		if native.Button(fmt.Sprintf("×##ov%d", oi)) {
			mmEditorRemoveOverflow(s, env, oi)
		}
	}
}

// drawMMEditorSearch renders the command filter box and the matching command list.
// Clicking a command assigns it to the selected slot immediately.
func drawMMEditorSearch(s markingMenuEditorSession) {
	native.Text("Filter commands")
	native.SameLine()
	native.SetNextItemWidth(-1)
	native.InputText("##mm-filter", mmEditorUI.filter)
	query := bufString(mmEditorUI.filter)
	if query == "" {
		native.Text("(type to search, then click a result to assign to the selected slot)")
		return
	}
	cmds := s.SearchCommands(query)
	if len(cmds) == 0 {
		native.Text("(no matches)")
		return
	}
	for _, c := range cmds {
		label := c.DisplayName() + " [" + c.ID() + "]"
		if native.Selectable(label+"##mm-cmd", false) {
			mmEditorAssign(s, c.ID())
		}
	}
}

// mmEditorAssign assigns commandID to the currently selected slot or overflow entry.
func mmEditorAssign(s markingMenuEditorSession, commandID string) {
	if mmEditorUI.slot == mmEditorNoSlot && !mmEditorUI.isOver {
		mmEditorUI.lastErr = "select a slot first, then click a command"
		return
	}
	mmEditorUI.lastErr = ""
	env := mmEditorEnvs[mmEditorUI.envTab]
	if mmEditorUI.isOver {
		mmEditorAssignOverflow(s, env, commandID)
		return
	}
	mmEditorAssignQuadrant(s, env, mmEditorSlots[mmEditorUI.slot].quadrant, commandID)
}

// mmEditorAssignQuadrant sets commandID on one quadrant, adding the slot if absent.
func mmEditorAssignQuadrant(s markingMenuEditorSession, env app.Environment, q types.ScreenQuadrant, commandID string) {
	menu := s.MarkingMenu(env)
	for i, item := range menu.Quadrants {
		if item.Quadrant == q {
			menu.Quadrants[i].CommandID = commandID
			recordMMEditorResult(s.SetMarkingMenu(menu))
			return
		}
	}
	menu.Quadrants = append(menu.Quadrants, wire.MarkingMenuItem{Quadrant: q, CommandID: commandID})
	recordMMEditorResult(s.SetMarkingMenu(menu))
}

// mmEditorAssignOverflow replaces one overflow entry by index.
func mmEditorAssignOverflow(s markingMenuEditorSession, env app.Environment, commandID string) {
	menu := s.MarkingMenu(env)
	if mmEditorUI.overIdx < len(menu.Overflow) {
		menu.Overflow[mmEditorUI.overIdx] = commandID
		recordMMEditorResult(s.SetMarkingMenu(menu))
	}
}

// mmEditorRemoveQuadrant clears the assignment for one quadrant slot.
func mmEditorRemoveQuadrant(s markingMenuEditorSession, env app.Environment, q types.ScreenQuadrant) {
	menu := s.MarkingMenu(env)
	for i, item := range menu.Quadrants {
		if item.Quadrant == q {
			menu.Quadrants = append(menu.Quadrants[:i], menu.Quadrants[i+1:]...)
			break
		}
	}
	recordMMEditorResult(s.SetMarkingMenu(menu))
}

// mmEditorRemoveOverflow removes one overflow entry by index.
func mmEditorRemoveOverflow(s markingMenuEditorSession, env app.Environment, idx int) {
	menu := s.MarkingMenu(env)
	if idx >= len(menu.Overflow) {
		return
	}
	menu.Overflow = append(menu.Overflow[:idx], menu.Overflow[idx+1:]...)
	recordMMEditorResult(s.SetMarkingMenu(menu))
	if mmEditorUI.isOver && mmEditorUI.overIdx >= len(menu.Overflow) {
		mmEditorUI.isOver = false
		mmEditorUI.overIdx = 0
	}
}

// recordMMEditorResult clears the panel error on success or shows the reason.
func recordMMEditorResult(err error) {
	if err != nil {
		mmEditorUI.lastErr = err.Error()
		return
	}
	mmEditorUI.lastErr = ""
}
