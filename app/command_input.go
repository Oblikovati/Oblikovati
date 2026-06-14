// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
)

// The command-alias input box (M05-F17, #831): the user types a multi-character command
// alias and commits with Enter to run it — the reference product's command-line input.
// The buffer lives in the session so the flow is unit-testable without the head; the head
// renders a status-bar text field bound to it (and forwards Enter/Escape). Resolution
// goes through the binding engine: a typed alias first, then a single-character fallback
// to that key's shortcut so the familiar one-letter access still works.

// commandInput is the buffer state of the alias input box.
type commandInput struct {
	active bool
	buf    string
}

// BeginCommandInput opens the alias input box with an empty buffer.
func (s *Session) BeginCommandInput() {
	s.cmdInput = commandInput{active: true}
	s.notice = ""
}

// CommandInputActive reports whether the alias input box is open.
func (s *Session) CommandInputActive() bool { return s.cmdInput.active }

// CommandInputText returns the current buffer contents.
func (s *Session) CommandInputText() string { return s.cmdInput.buf }

// SetCommandInputText replaces the buffer — the head binds its text field to this so the
// session stays the single source of truth.
func (s *Session) SetCommandInputText(text string) {
	if s.cmdInput.active {
		s.cmdInput.buf = text
	}
}

// AppendCommandInputRune appends one character (the headless typing path).
func (s *Session) AppendCommandInputRune(r rune) {
	if s.cmdInput.active {
		s.cmdInput.buf += string(r)
	}
}

// BackspaceCommandInput deletes the last character of the buffer.
func (s *Session) BackspaceCommandInput() {
	if !s.cmdInput.active || s.cmdInput.buf == "" {
		return
	}
	runes := []rune(s.cmdInput.buf)
	s.cmdInput.buf = string(runes[:len(runes)-1])
}

// CancelCommandInput closes the box, discarding the buffer.
func (s *Session) CancelCommandInput() { s.cmdInput = commandInput{} }

// CommandInputMatches returns "ALIAS — Command" hints for every action whose alias has
// the current buffer as a case-insensitive prefix — the box's live match list.
func (s *Session) CommandInputMatches() []string {
	prefix := strings.ToLower(strings.TrimSpace(s.cmdInput.buf))
	if prefix == "" {
		return nil
	}
	var out []string
	for _, bd := range s.Bindings().Catalog() {
		if bd.Alias != "" && strings.HasPrefix(strings.ToLower(bd.Alias), prefix) {
			out = append(out, bd.Alias+" — "+bd.DisplayName)
		}
	}
	return out
}

// CommitCommandInput resolves the buffered alias and runs it: a typed alias first, then a
// single-character fallback to that key's shortcut. On a hit the box closes and the action
// dispatches; on a miss the box stays open and the reason is surfaced (a loud typo).
func (s *Session) CommitCommandInput() error {
	buf := strings.TrimSpace(s.cmdInput.buf)
	b := s.Bindings()
	actionID, ok := b.ResolveAlias(buf)
	if !ok && len([]rune(buf)) == 1 {
		actionID, ok = b.ResolveChord(types.KeyChord{Key: buf})
	}
	if !ok {
		s.notice = fmt.Sprintf("no command for alias %q", buf)
		return fmt.Errorf("app: no command for alias %q", buf)
	}
	s.cmdInput = commandInput{}
	return b.Dispatch(actionID, s)
}

// routeKeyToCommandInput edits the buffer from a key event while the box is open: Enter
// commits, Escape cancels, Backspace deletes, and a printable single character is appended.
func (s *Session) routeKeyToCommandInput(e KeyEvent) error {
	switch normalizeKey(e.Key) {
	case "Enter":
		return s.CommitCommandInput()
	case "Escape":
		s.CancelCommandInput()
		return nil
	case "Backspace":
		s.BackspaceCommandInput()
		return nil
	default:
		if rs := []rune(e.Key); len(rs) == 1 {
			s.AppendCommandInputRune(rs[0])
		}
		return nil
	}
}
