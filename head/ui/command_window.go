//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The docked Command Window (M26 F04): a rolling, severity-coloured history pane above a
// persistent input line, docked across the bottom. Submitting a line drives the same REPL
// the API uses (app.CommandLine). Pure presentation helpers (colours) live in the no-cgo
// command_window_text.go so they are unit-tested without the native layer.

// commandInputBuf holds the editable command line across frames.
var commandInputBuf = make([]byte, 1024)

// commandWindowDocked guards the one-time dock into the bottom band; commandFocusNext asks
// for keyboard focus on the input next frame (on first show and after each submit);
// commandLastLineCount tracks scrollback growth so auto-tail only fires on new output.
var (
	commandWindowDocked  bool
	commandFocusNext     = true
	commandLastLineCount int
	commandHistoryCursor int32 // ↑/↓ recall position (len(history) ⇒ the empty line)
)

// commandInputReserve is the height (logical px) kept below the scrollback for the input.
const commandInputReserve = 30

// drawCommandWindow renders the docked Command Window when it is open. It supersedes the
// old notification surfaces as the single feedback + command-entry surface (M26).
func drawCommandWindow(s *app.Session) {
	if !s.CommandWindowOpen() {
		return
	}
	if !commandWindowDocked {
		native.SetNextWindowDock(dockSideNodes.Bottom)
		commandWindowDocked = true
	}
	if native.Begin("Command") {
		drawCommandWindowBody(s)
	}
	native.End()
}

// drawCommandWindowBody draws the scrollback pane and the input line.
func drawCommandWindowBody(s *app.Session) {
	cl := s.CommandLine()
	drawCommandScrollback(cl)
	drawCommandInputLine(s, cl)
}

// drawCommandScrollback renders the rolling history in a scrollable pane that follows the
// tail when new lines arrive, while still letting the user scroll up when nothing new is
// appended (auto-tail fires only on growth, like a terminal).
func drawCommandScrollback(cl *app.CommandLine) {
	if !native.BeginChild("##cmd-scrollback", 0, -commandInputReserve, false) {
		native.EndChild()
		return
	}
	lines := cl.Scrollback().Lines()
	for _, ln := range lines {
		native.PushStyleColor("Text", severityColor(ln.Severity))
		native.Text(ln.Text)
		native.PopStyleColor(1)
	}
	if len(lines) > commandLastLineCount {
		native.SetScrollHereY()
	}
	commandLastLineCount = len(lines)
	native.EndChild()
}

// drawCommandInputLine draws the full-width input with Up/Down history recall; Enter submits
// the line to the engine, clears it, resets the recall cursor past the newest entry, and
// refocuses for the next command (a persistent shell-style command line).
func drawCommandInputLine(s *app.Session, cl *app.CommandLine) {
	if commandFocusNext {
		native.SetKeyboardFocusHere()
		commandFocusNext = false
	}
	history := cl.Scrollback().History()
	native.SetNextItemWidth(-1)
	if native.InputTextHistory("##cmd-input", commandInputBuf, history, &commandHistoryCursor) {
		_ = cl.Submit(s, bufString(commandInputBuf))
		clearBuf(commandInputBuf)
		commandHistoryCursor = int32(len(cl.Scrollback().History())) // back to the empty line
		commandFocusNext = true
	}
}
