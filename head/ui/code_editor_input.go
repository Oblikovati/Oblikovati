//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/head/internal/native"

// handleKeys applies this frame's navigation, editing and shortcut keys. Ctrl turns the
// horizontal/Home/End moves into word- and document-wise moves; Shift extends the selection.
// Typed text is handled separately (handleChars); these are the keys that produce no character.
func (e *codeEditor) handleKeys() {
	k := native.EditorKeysPressed()
	shift, ctrl := native.KeyShift(), native.KeyCtrl()
	if ctrl && e.handleShortcuts(k) {
		return // a clipboard/undo/select-all shortcut consumed the frame
	}
	e.handleMovement(k, shift, ctrl)
	e.handleEditing(k)
}

// handleMovement maps the arrow/Home/End keys to caret moves, with Ctrl selecting the
// word/document-wise variant.
func (e *codeEditor) handleMovement(k native.EditorKeys, shift, ctrl bool) {
	switch {
	case k.Left && ctrl:
		e.model.MoveWordLeft(shift)
	case k.Left:
		e.model.MoveLeft(shift)
	case k.Right && ctrl:
		e.model.MoveWordRight(shift)
	case k.Right:
		e.model.MoveRight(shift)
	}
	e.handleVertical(k, shift, ctrl)
}

// handleVertical maps Up/Down and Home/End (Ctrl ⇒ document bounds).
func (e *codeEditor) handleVertical(k native.EditorKeys, shift, ctrl bool) {
	switch {
	case k.Up:
		e.model.MoveUp(shift)
	case k.Down:
		e.model.MoveDown(shift)
	}
	switch {
	case k.Home && ctrl:
		e.model.MoveDocStart(shift)
	case k.Home:
		e.model.MoveHome(shift)
	case k.End && ctrl:
		e.model.MoveDocEnd(shift)
	case k.End:
		e.model.MoveEnd(shift)
	}
	if k.Up || k.Down || k.Home || k.End {
		e.resetBlink()
	}
}

// handleEditing maps the text-mutating keys (Backspace/Delete/Enter/Tab).
func (e *codeEditor) handleEditing(k native.EditorKeys) {
	switch {
	case k.Backspace:
		e.model.Backspace()
	case k.Delete:
		e.model.Delete()
	case k.Enter:
		e.model.Newline()
	case k.Tab:
		e.model.Insert(indentUnit) // Phase 3 will make Tab indent a selected block
	default:
		return
	}
	e.resetBlink()
}

// handleShortcuts applies the Ctrl-modified shortcuts (copy/cut/paste/select-all/undo/redo),
// returning true when one fired so the movement handlers skip the same keystroke (e.g. Ctrl+A).
func (e *codeEditor) handleShortcuts(k native.EditorKeys) bool {
	switch {
	case k.Copy:
		e.copySelection()
	case k.Cut:
		e.cutSelection()
	case k.Paste:
		e.model.Insert(native.ClipboardText())
	case k.SelectAll:
		e.model.SelectAll()
	case k.Undo:
		e.model.Undo()
	case k.Redo:
		e.model.Redo()
	default:
		return false
	}
	e.resetBlink()
	return true
}

// copySelection puts the current selection on the clipboard (a no-op when nothing is selected).
func (e *codeEditor) copySelection() {
	if s := e.model.SelectedText(); s != "" {
		native.SetClipboardText(s)
	}
}

// cutSelection copies then deletes the selection.
func (e *codeEditor) cutSelection() {
	if s := e.model.SelectedText(); s != "" {
		native.SetClipboardText(s)
		e.model.Backspace() // deletes the active selection
	}
}

// indentUnit is one indentation step (four spaces — Lua convention; no hard tabs).
const indentUnit = "    "
