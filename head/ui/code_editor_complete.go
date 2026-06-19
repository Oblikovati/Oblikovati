//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console/complete"
	"oblikovati.org/script/console/textbuf"
)

// completer holds the editor's live autocomplete state: the engine (built from the host method
// list), whether the popup is showing, the current candidates, the highlighted index, and the
// span a chosen candidate replaces. It is part of the codeEditor; the heavy lifting (context +
// ranking) is the tested complete.Engine.
type completer struct {
	engine *complete.Engine
	active bool
	items  []complete.Candidate
	sel    int
	ctx    complete.Context
}

// setMethods (re)builds the completion engine from the host's dotted wire-method names.
func (e *codeEditor) setMethods(methods []string) { e.completion.engine = complete.New(methods) }

// completionVisible reports whether the popup is showing candidates.
func (e *codeEditor) completionVisible() bool {
	return e.completion.active && len(e.completion.items) > 0
}

// refreshCompletion recomputes candidates for the caret's context. trigger forces the popup
// even with an empty prefix (Ctrl+Space / typing a '.'), so `oblikovati.` lists its groups.
func (e *codeEditor) refreshCompletion(trigger bool) {
	c := &e.completion
	if c.engine == nil {
		return
	}
	caret := e.model.Caret()
	items, ctx := c.engine.Suggest(e.model.Line(caret.Line), caret.Col)
	c.items, c.ctx = items, ctx
	if c.sel >= len(items) {
		c.sel = 0
	}
	prefixLen := caret.Col - ctx.ReplaceStart
	c.active = (trigger || prefixLen > 0) && len(items) > 0
}

// dismissCompletion hides the popup.
func (e *codeEditor) dismissCompletion() { e.completion.active = false }

// handleCompletionKeys consumes the keys that drive the popup while it is visible: Up/Down move
// the selection, Enter/Tab accept, Esc dismisses. It returns true when it handled the frame so
// the normal editor key handling is skipped.
func (e *codeEditor) handleCompletionKeys(k native.EditorKeys) bool {
	switch {
	case k.Escape:
		e.dismissCompletion()
		return true
	case k.Up:
		e.moveCompletion(-1)
		return true
	case k.Down:
		e.moveCompletion(1)
		return true
	case k.Enter, k.Tab:
		e.acceptCompletion()
		return true
	}
	return false
}

// moveCompletion steps the highlighted candidate, clamping at the ends.
func (e *codeEditor) moveCompletion(delta int) {
	c := &e.completion
	c.sel += delta
	if c.sel < 0 {
		c.sel = 0
	}
	if c.sel >= len(c.items) {
		c.sel = len(c.items) - 1
	}
}

// acceptCompletion replaces the typed prefix with the highlighted candidate and closes the popup.
func (e *codeEditor) acceptCompletion() {
	c := &e.completion
	caret := e.model.Caret()
	e.model.SetCaret(textbuf.Position{Line: caret.Line, Col: c.ctx.ReplaceStart}, false)
	e.model.SetCaret(caret, true) // select the prefix span
	e.model.Insert(c.items[c.sel].Text)
	c.active = false
	e.resetBlink()
}
