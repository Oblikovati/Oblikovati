// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// appQatHost is the seam through which the Quick Access Toolbar and the
// Application-menu button reach the session. All six methods pre-exist on
// *app.Session — the QAT adds no new session verbs (G design, Requirement 4)
// and, because the interface hides the concrete type, no new *app.Session
// references reach head/ui (archguard ratchet stays at 440).
type appQatHost interface {
	Execute(id string) error // New Part via the ribbon command path
	Undo() error             // Undo
	Redo() error             // Redo
	InTransaction() bool     // #1750 guard — no undo/redo mid-transaction
	CanUndo() bool           // empty-stack grey-out (Edit-menu parity)
	CanRedo() bool           // empty-stack grey-out (Edit-menu parity)
}

// The seam's proof: the session satisfies the host contract. The ratchet
// excludes lines containing (*app.Session)(nil) by rule
// (archguard/head_session_ratchet_test.go).
var _ appQatHost = (*app.Session)(nil)

// qatAction is the semantic action of a QAT button.
type qatAction int

const (
	qatNone qatAction = iota
	qatNewPart
	qatOpen
	qatSave
	qatUndo
	qatRedo
)

// qatEffect is the pure-function decision for one QAT button: what to do on
// click and whether the button is disabled this frame.
type qatEffect struct {
	callID   string // Execute this command id (New Part)
	callUndo bool   // call h.Undo()
	callRedo bool   // call h.Redo()
	setOpen  bool   // defer Open via the qatOpenRequested flag
	setSave  bool   // defer Save via the qatSaveRequested flag
	disabled bool   // BeginDisabled condition
}

// qatDispatch carries the ENTIRE dispatch + enabled-state decision for a QAT
// button. It is pure — no ImGui, no session — so qat_test.go exercises the
// real logic: the id→channel mapping, the #1750 in-transaction guard
// (callUndo/callRedo are false while a transaction is open), and the
// CanUndo/CanRedo grey-out conditions (G design, r7/r9/r10/r11 final shape).
func qatDispatch(buttonID string, inTxn, canUndo, canRedo bool) qatEffect {
	switch qatActionForID(buttonID) {
	case qatNewPart:
		return qatEffect{callID: "GetStarted.NewPart"}
	case qatOpen:
		return qatEffect{setOpen: true}
	case qatSave:
		return qatEffect{setSave: true}
	case qatUndo:
		if !canUndo {
			return qatEffect{disabled: true} // greyed: never carries a dispatch effect
		}
		return qatEffect{callUndo: !inTxn}
	case qatRedo:
		if !canRedo {
			return qatEffect{disabled: true}
		}
		return qatEffect{callRedo: !inTxn}
	default:
		return qatEffect{}
	}
}

// qatActionForID maps a QAT button id to its semantic action.
func qatActionForID(id string) qatAction {
	switch id {
	case "new-part":
		return qatNewPart
	case "open":
		return qatOpen
	case "save":
		return qatSave
	case "undo":
		return qatUndo
	case "redo":
		return qatRedo
	default:
		return qatNone
	}
}

// qatButton is one QAT entry: its stable id, icon key and tooltip label.
type qatButton struct {
	id    string
	icon  string
	label string
}

// qatButtons is the fixed default set — Inventor's core defaults
// (G design D2). QAT customization is explicitly out of scope (§5).
var qatButtons = []qatButton{
	{id: "new-part", icon: "new-part", label: "New Part"},
	{id: "open", icon: "open", label: "Open"},
	{id: "save", icon: "save", label: "Save"},
	{id: "undo", icon: "undo", label: "Undo"},
	{id: "redo", icon: "redo", label: "Redo"},
}

// QAT/App-menu deferred-action flags: set by the band's buttons, consumed in
// DrawChrome's context each frame (G design D3 — the popup must open on the
// same ID stack as BeginPopup; Open/Save must run after the ribbon so the
// modal opens in the same frame). The head draws one chrome context per
// window and the GUI runs one window today — the same package-state idiom as
// fileModal / ribbonScrollbarShown / reportBugUI.
var (
	qatOpenRequested bool
	qatSaveRequested bool
	appMenuRequested bool
	appMenuX         float32
	appMenuBottomY   float32
)

// drawQuickAccess draws the pre-ribbon strip inside the band: the
// Application-menu button at the left, then the QAT buttons sharing the tab
// row (G design D1). Click actions are deferred via the package flags;
// DrawChrome consumes them after drawRibbon.
func drawQuickAccess(h appQatHost) {
	drawAppMenuButton()
	for _, b := range qatButtons {
		native.SameLine()
		drawQATButton(h, b)
	}
}

// drawAppMenuButton draws the Application-menu corner button (file-menu
// glyph). It is archguard-clean: it only draws and sets the appMenuRequested
// flag plus the button's screen position — the popup itself opens in
// DrawChrome's context (ID-stack note, G design D3).
func drawAppMenuButton() {
	px := float32(scaledIconPx(smallIconPx))
	x, _ := native.GetCursorScreenPos()
	var clicked bool
	if tex, ok := icons.texture("file-menu", "", scaledIconPx(smallIconPx)); ok {
		clicked = native.ImageButton("##app-menu", tex, px, px, identityTint)
	} else {
		clicked = native.Button("File") // missing asset never hides the button
	}
	// Capture the cursor AFTER the button: the bottom edge anchors the popup
	// below the button (r5 P3 — never use the pre-button Y alone).
	_, y2 := native.GetCursorScreenPos()
	appMenuX, appMenuBottomY = x, y2
	native.SetItemTooltip("File")
	if clicked {
		appMenuRequested = true
	}
}

// drawQATButton draws one QAT icon button with its dispatch decision from
// qatDispatch and executes the effect on click. It is the thin mechanical
// executor: all decision logic lives in the pure qatDispatch (G design,
// r5/r6/r7 — the tested surface).
func drawQATButton(h appQatHost, b qatButton) {
	eff := qatDispatch(b.id, h.InTransaction(), h.CanUndo(), h.CanRedo())
	native.BeginDisabled(eff.disabled)
	px := float32(scaledIconPx(smallIconPx))
	var clicked bool
	if tex, ok := icons.texture(b.icon, "", scaledIconPx(smallIconPx)); ok {
		clicked = native.ImageButton("##qat-"+b.id, tex, px, px, identityTint)
	} else {
		clicked = native.Button(b.label) // missing asset never hides the command
	}
	native.EndDisabled()
	native.SetItemTooltip(b.label)
	if clicked && !eff.disabled {
		applyQATEffect(h, eff)
	}
}

// applyQATEffect executes the pure function's decision. Undo/Redo drop the
// returned error exactly like the Ctrl+Z/Ctrl+Y chords; failures surface via
// s.notice (undo.go:630-637) through the command box (command_input_box.go:58).
func applyQATEffect(h appQatHost, eff qatEffect) {
	switch {
	case eff.callID != "":
		_ = h.Execute(eff.callID)
	case eff.callUndo:
		_ = h.Undo()
	case eff.callRedo:
		_ = h.Redo()
	case eff.setOpen:
		qatOpenRequested = true
	case eff.setSave:
		qatSaveRequested = true
	}
}
