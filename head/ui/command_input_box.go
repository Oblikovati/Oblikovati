//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The command-alias input box (M05-F17, #831): a focused text field where the user types
// a command alias and presses Enter to run it, with a live list of matches. The session
// owns the buffer and the resolve/dispatch; this file is the ImGui surface bound to it.
// Because the field holds keyboard focus, native.WantTextInput suppresses global Ctrl
// shortcuts while typing (head/ui/chrome.go), so e.g. Ctrl+Z in the box never undoes.

const commandInputBufLen = 64

// commandInputUI is the box's cross-frame widget state.
var commandInputUI = struct {
	buf    [commandInputBufLen]byte
	primed bool // caret focused + buffer seeded for the current open session
}{}

// drawCommandInput renders the command-alias input box while it is open.
func drawCommandInput(s *app.Session) {
	if !s.CommandInputActive() {
		commandInputUI.primed = false
		return
	}
	native.SetNextWindowSizeOnce(360, 0)
	if native.Begin("Command Input") {
		drawCommandInputBody(s)
	}
	native.End()
}

// drawCommandInputBody draws the box's contents: a focused field, the live matches, and
// the Run/Cancel buttons. The buffer is seeded and focused once per open session.
func drawCommandInputBody(s *app.Session) {
	if !commandInputUI.primed {
		copyText(commandInputUI.buf[:], s.CommandInputText())
		native.SetKeyboardFocusHere()
		commandInputUI.primed = true
	}
	native.SetNextItemWidth(-1)
	submitted := native.InputTextSubmit("##command-alias", commandInputUI.buf[:])
	s.SetCommandInputText(bufString(commandInputUI.buf[:]))
	drawCommandInputMatches(s)
	if native.Button("Run") || submitted {
		runCommandInput(s)
	}
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelCommandInput()
	}
	if note := s.Notice(); note != "" {
		native.Text(note)
	}
}

// runCommandInput commits the typed alias. A miss keeps the box open with the reason in
// the session notice (shown above); a hit closes it and re-primes for next time.
func runCommandInput(s *app.Session) {
	s.SetCommandInputText(bufString(commandInputUI.buf[:]))
	if err := s.CommitCommandInput(); err != nil {
		return
	}
	commandInputUI.primed = false
}

// drawCommandInputMatches lists the alias hints for the current buffer.
func drawCommandInputMatches(s *app.Session) {
	for _, m := range s.CommandInputMatches() {
		native.BulletText(m)
	}
}
