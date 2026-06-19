// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
)

// Input routing — the methods a viewport (or a test) calls to drive the session.
// They implement Inventor's mouse/keyboard behavior at the logic level; the actual
// device events come from the window in production and from tests headlessly.

// SetPicker installs the hit-test used to resolve clicks to selectables.
func (s *Session) SetPicker(p Picker) { s.picker = p }

// StartTool activates an interactive tool, cancelling any tool already running.
func (s *Session) StartTool(t Tool) {
	if s.tool != nil {
		s.tool.tool.Cancel(s)
		s.Graphics().ClearInteraction() // drop the previous tool's preview/overlay graphics
		s.dropCommandMiniToolbars()     // and its mini-toolbars (M05-F07)
		s.dropCommandGizmos()           // and its triad/manipulators (M05-F13)
	}
	s.notice = ""
	s.tool = &ToolInstance{tool: t}
	t.Start(s)
}

// ActiveTool returns the running tool instance, or nil.
func (s *Session) ActiveTool() *ToolInstance { return s.tool }

// PickAt hit-tests the pixel through the installed picker without changing selection —
// the viewport uses it for hover feedback (which plane/face is under the cursor).
func (s *Session) PickAt(x, y float64, filter *SelectionFilter) (Selectable, bool) {
	if s.picker == nil {
		return nil, false
	}
	return s.picker.Pick(x, y, filter)
}

// OK finishes the active tool if it has enough input (Inventor's OK), clearing it on
// success. With no active tool it is a no-op error.
func (s *Session) OK() error {
	if s.tool == nil {
		return errors.New("app: no active tool to commit")
	}
	if !s.tool.tool.CanCommit() {
		return errors.New("app: active tool is not ready to commit")
	}
	if err := s.tool.tool.Commit(s); err != nil {
		s.notice = err.Error() // surface why (the status bar shows it); keep the tool open
		return err
	}
	s.notice = ""
	s.tool = nil
	s.Graphics().ClearInteraction() // a committed command's transient preview vanishes
	s.dropCommandMiniToolbars()     // command-bound mini-toolbars die with the tool (M05-F07)
	s.dropCommandGizmos()           // and the command-bound triad/manipulators (M05-F13)
	return nil
}

// CancelTool abandons the active tool (Inventor's Escape / Cancel).
func (s *Session) CancelTool() {
	s.notice = ""
	if s.tool != nil {
		s.tool.tool.Cancel(s)
		s.tool = nil
		s.Graphics().ClearInteraction() // a cancelled command's transient preview vanishes
		s.dropCommandMiniToolbars()     // command-bound mini-toolbars die with the tool (M05-F07)
		s.dropCommandGizmos()           // and the command-bound triad/manipulators (M05-F13)
	}
}

// autoCommitter is a Tool that should finish as soon as a pick makes it ready, rather
// than waiting for a separate OK — e.g. Create 2D Sketch enters the sketch the moment a
// plane is clicked.
type autoCommitter interface {
	AutoCommitOnPick() bool
}

// feedPick hands a picked selectable to the active tool and, when the tool opts into
// auto-commit and is now ready, finishes it immediately (Inventor's click-to-proceed).
func (s *Session) feedPick(sel Selectable) {
	s.tool.tool.Pick(s, sel)
	s.autoCommitAfterPick()
}

// modifierPicker is a Tool whose pick behavior depends on held modifiers — e.g. Extrude
// adds a region to its set on Ctrl+click rather than replacing the single selection.
type modifierPicker interface {
	PickWithMods(*Session, Selectable, Modifier)
}

// feedPickMods is feedPick for graphics clicks that carry modifiers: a modifier-aware
// tool sees the held keys (Ctrl to extend a multi-selection); others get the plain pick.
func (s *Session) feedPickMods(sel Selectable, mods Modifier) {
	if mp, ok := s.tool.tool.(modifierPicker); ok {
		mp.PickWithMods(s, sel, mods)
	} else {
		s.tool.tool.Pick(s, sel)
	}
	s.autoCommitAfterPick()
}

// autoCommitAfterPick commits the active tool when it opts into auto-commit and is now
// ready (used after both 3D picks and snap-aware sketch-entity picks).
func (s *Session) autoCommitAfterPick() {
	if ac, ok := s.tool.tool.(autoCommitter); ok && ac.AutoCommitOnPick() && s.tool.tool.CanCommit() {
		_ = s.OK()
	}
}

// Select replaces the selection with a single selectable and emits SelectionChanged
// (the entry point for browser-node and graphics selection outside a tool).
func (s *Session) Select(sel Selectable) {
	s.selection.Clear()
	if sel != nil && s.selection.Add(sel) {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
}

// SelectBrowserNode acts on a clicked browser node: while a tool is active the node's
// entity is fed to it as a pick (so clicking "XY Plane" in the tree picks it for the
// Create Sketch tool); otherwise it becomes the selection (e.g. to pre-select a plane).
func (s *Session) SelectBrowserNode(n BrowserNode) {
	if n.Select == nil {
		return
	}
	if s.tool != nil {
		s.feedPick(n.Select)
		return
	}
	s.Select(n.Select)
}

// Click is a left-button click at a viewport coordinate (the common case). It picks
// the front-most selectable honoring the active filter: while a tool is active the
// pick is fed to the tool; otherwise it joins the selection set.
func (s *Session) Click(x, y float64) {
	s.Pointer(PointerEvent{X: x, Y: y, Button: LeftButton})
}

// Pointer routes a pointer event per Inventor mouse behavior. Left selects/feeds the
// tool; right and middle are reserved for the marking menu and orbit/pan (handled by
// the viewport, no model effect here yet).
func (s *Session) Pointer(e PointerEvent) {
	if e.Button != LeftButton {
		return
	}
	// A sketch tool consumes plane-point clicks directly (not entity picks).
	if s.sketchClick(e.X, e.Y) {
		return
	}
	// In the sketch environment clicks pick sketch entities: fed to an active
	// constraint/dimension tool, or (with no tool) added to the selection.
	if s.InSketch() {
		s.sketchEntityPointer(e)
		return
	}
	if s.picker == nil {
		return
	}
	sel, ok := s.picker.Pick(e.X, e.Y, s.pickFilter())
	if !ok {
		s.clearSelectionOnEmptyClick(e.Mods)
		return
	}
	if s.tool != nil {
		s.feedPickMods(sel, e.Mods)
		return
	}
	s.applyPickToSelection(sel, e.Mods)
}

// clearSelectionOnEmptyClick clears the selection when the user clicks empty space with
// no active tool — Inventor (GUID-B8F6E805): "click in an empty area to deselect". A
// modifier-held empty click is a no-op (it neither clears nor extends), and an active tool
// owns its own miss handling, so both are left untouched.
func (s *Session) clearSelectionOnEmptyClick(mods Modifier) {
	if s.tool != nil || mods.Has(ShiftMod) || mods.Has(CtrlMod) || s.selection.Count() == 0 {
		return
	}
	s.selection.Clear()
	event.Emit(s.bus, event.After, SelectionChanged{Count: 0})
}

// applyPickToSelection updates the selection set for a viewport pick with no active tool,
// mirroring Inventor (GUID-B8F6E805): a plain click replaces the selection; Shift/Ctrl+click
// toggles the clicked object's membership (add if new, remove if already selected).
func (s *Session) applyPickToSelection(sel Selectable, mods Modifier) {
	var changed bool
	if mods.Has(ShiftMod) || mods.Has(CtrlMod) {
		changed = s.selection.Toggle(sel)
	} else {
		s.selection.Clear()
		changed = s.selection.Add(sel)
	}
	if changed {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
}

// keyEventToChord converts a device key event into a canonical [types.KeyChord],
// normalizing key synonyms (Return → Enter) so a binding matches however the platform
// spells the key.
func keyEventToChord(e KeyEvent) types.KeyChord {
	return types.KeyChord{
		Key:   normalizeKey(e.Key),
		Ctrl:  e.Mods.Has(CtrlMod),
		Alt:   e.Mods.Has(AltMod),
		Shift: e.Mods.Has(ShiftMod),
	}
}

// normalizeKey maps platform key synonyms onto the canonical token the binding table
// uses, then canonicalizes case so "z" and "Z" are one chord.
func normalizeKey(key string) string {
	if key == "Return" {
		return "Enter"
	}
	return types.CanonicalKey(key)
}

// PressKey routes a key press through the binding engine (M05-F17, #831). While the legacy
// command-alias input box is open the keystroke edits its buffer; otherwise the chord the
// event forms is resolved to an action and dispatched. With no matching binding it is a
// no-op. The built-in guards (e.g. undo is suppressed mid-tool) live in the dispatch.
//
// M26 F05: a modifier chord (Ctrl/Alt, e.g. Ctrl+S, Ctrl+Z) runs through the Command Window
// — its canonical word is echoed and then it dispatches — so a chord reads like a typed
// command ("Ctrl+S" shows "SAVE" and saves). Plain keys still dispatch directly (this is the
// no-text-field-focused path; in the running app a plain letter is typed into the focused
// command line instead, filling it to await Enter).
func (s *Session) PressKey(e KeyEvent) error {
	if s.CommandInputActive() {
		return s.routeKeyToCommandInput(e)
	}
	chord := keyEventToChord(e)
	actionID, ok := s.Bindings().ResolveChord(chord)
	if !ok {
		return nil
	}
	if chord.Ctrl || chord.Alt {
		return s.CommandLine().RunChord(s, actionID)
	}
	return s.Bindings().Dispatch(actionID, s)
}
