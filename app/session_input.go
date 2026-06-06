// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati/event"
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
		s.graphics.ClearInteraction() // drop the previous tool's preview/overlay graphics
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
	s.graphics.ClearInteraction() // a committed command's transient preview vanishes
	return nil
}

// CancelTool abandons the active tool (Inventor's Escape / Cancel).
func (s *Session) CancelTool() {
	s.notice = ""
	if s.tool != nil {
		s.tool.tool.Cancel(s)
		s.tool = nil
		s.graphics.ClearInteraction() // a cancelled command's transient preview vanishes
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
	sel, ok := s.picker.Pick(e.X, e.Y, s.selection.Filter())
	if !ok {
		return
	}
	if s.tool != nil {
		s.feedPickMods(sel, e.Mods)
		return
	}
	if !e.Mods.Has(ShiftMod) && !e.Mods.Has(CtrlMod) {
		s.selection.Clear() // a plain click replaces the selection
	}
	if s.selection.Add(sel) {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
}

// undoRedoShortcut handles the Ctrl+Z / Ctrl+Y / Ctrl+Shift+Z navigators over the active
// document's transaction stream, returning handled=true when the keystroke was one of
// them. It is a no-op while an interactive tool is mid-operation — Inventor forbids undo
// while a transaction is in progress; the head additionally gates it on no text field
// having keyboard focus.
func (s *Session) undoRedoShortcut(e KeyEvent) (bool, error) {
	if !e.Mods.Has(CtrlMod) || s.tool != nil {
		return false, nil
	}
	switch e.Key {
	case "z", "Z":
		if e.Mods.Has(ShiftMod) {
			return true, s.Redo()
		}
		return true, s.Undo()
	case "y", "Y":
		return true, s.Redo()
	}
	return false, nil
}

// PressKey routes a key press: Escape cancels the active tool, Enter commits it, and
// otherwise a registered command alias runs (Inventor command aliases).
func (s *Session) PressKey(e KeyEvent) error {
	if handled, err := s.undoRedoShortcut(e); handled {
		return err
	}
	switch e.Key {
	case "Escape":
		// Esc cancels the active tool at any point in its operation; with no tool it
		// clears the selection (Inventor's behavior).
		if s.tool != nil {
			s.CancelTool()
		} else {
			s.Select(nil)
		}
		return nil
	case "Enter", "Return":
		if s.tool != nil {
			return s.OK()
		}
		return nil
	case "v", "V":
		// Toggle visibility of the selected work plane(s) (Autodesk Fusion's V binding;
		// Inventor has no default visibility hotkey). No-op when nothing applicable is
		// selected, so V stays free for other contexts.
		s.ToggleSelectedWorkPlaneVisibility()
		return nil
	default:
		if _, ok := s.commands.ByAlias(e.Key); ok {
			return s.Invoke(e.Key)
		}
		return nil
	}
}
