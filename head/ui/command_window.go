//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/app/cmdline"
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

	// Autocomplete state: the candidate list for the current input, the highlighted index,
	// and the buffer it was computed for (so it is recomputed only when the input changes).
	commandCompletions []string
	commandCompSel     int32
	commandCompForBuf  string

	// Last mirrored prompt / status text, so each is appended to the scrollback only when it
	// changes (the command window is the single feedback surface — these used to live in the
	// status bar).
	commandLastPrompt string
	commandLastStatus string
)

const (
	// commandInputReserve is the height (logical px) kept below the scrollback for the input.
	commandInputReserve = 30
	// commandCompLineHeight is the per-row height of the autocomplete hint list.
	commandCompLineHeight = 19
	// commandCompMax caps how many autocomplete suggestions are shown at once.
	commandCompMax = 8
)

// drawCommandWindow renders the docked Command Window when it is open. It supersedes the
// old notification surfaces as the single feedback + command-entry surface (M26).
func drawCommandWindow(s *app.Session) {
	mirrorCommandFeedback(s) // always mirror prompt/status text, even while collapsed/hidden
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

// mirrorCommandFeedback appends the active command's current step prompt and any add-in status
// text to the scrollback when either changes (M26): the command window is the single feedback
// surface, so what used to show in the status bar now lands here, for tools started from the
// ribbon as well as from the command line. Change-detection avoids per-frame repeats.
func mirrorCommandFeedback(s *app.Session) {
	cl := s.CommandLine()
	if p := cl.Prompt(s); p != commandLastPrompt {
		commandLastPrompt = p
		if p != "" {
			cl.Scrollback().Append(p, cmdline.Prompt)
		}
	}
	if st := s.StatusText(); st != commandLastStatus {
		commandLastStatus = st
		if st != "" {
			cl.Scrollback().Append(st, cmdline.Info)
		}
	}
}

// drawCommandWindowBody draws the scrollback pane, the autocomplete hint list, and the input.
func drawCommandWindowBody(s *app.Session) {
	cl := s.CommandLine()
	comps := refreshCompletions(s, cl)
	reserve := float32(commandInputReserve) + float32(len(comps))*commandCompLineHeight
	drawCommandScrollback(cl, reserve)
	drawCommandCompletions(comps)
	drawCommandInputLine(s, cl, comps)
}

// refreshCompletions recomputes the autocomplete candidates when the input changes (resetting
// the highlighted index), and keeps them stable while the user only navigates the list — so
// Up/Down move the selection without the list shifting under them. Capped at commandCompMax.
func refreshCompletions(s *app.Session, cl *app.CommandLine) []string {
	buf := bufString(commandInputBuf)
	if buf != commandCompForBuf {
		commandCompForBuf = buf
		commandCompletions = cl.Completions(s, buf)
		if len(commandCompletions) > commandCompMax {
			commandCompletions = commandCompletions[:commandCompMax]
		}
		commandCompSel = 0
	}
	if int(commandCompSel) >= len(commandCompletions) {
		commandCompSel = 0
	}
	return commandCompletions
}

// drawCommandScrollback renders the rolling history in a scrollable pane that follows the
// tail when new lines arrive, while still letting the user scroll up when nothing new is
// appended (auto-tail fires only on growth, like a terminal). reserve is the height kept
// below it for the hint list + input.
func drawCommandScrollback(cl *app.CommandLine, reserve float32) {
	if !native.BeginChild("##cmd-scrollback", 0, -reserve, false) {
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

// drawCommandCompletions renders the autocomplete hint list above the input, the highlighted
// candidate (Up/Down) in the accent colour and the rest dimmed; Tab completes the highlight.
// Clicking a row also completes it.
func drawCommandCompletions(comps []string) {
	for i, c := range comps {
		selected := int32(i) == commandCompSel
		native.PushStyleColor("Text", completionColor(selected))
		native.Text(completionLabel(c, selected))
		native.PopStyleColor(1)
		if native.IsItemClicked(native.MouseLeft) {
			setBuf(commandInputBuf, c)
			commandCompForBuf = c
			commandFocusNext = true
		}
	}
}

// drawCommandInputLine draws the full-width input with Tab autocompletion and Up/Down
// navigation (the hint list when shown, else command history). Enter submits the typed line,
// clears it, resets the cursors, and refocuses for the next command.
func drawCommandInputLine(s *app.Session, cl *app.CommandLine, comps []string) {
	if commandFocusNext {
		native.SetKeyboardFocusHere()
		commandFocusNext = false
	}
	history := cl.Scrollback().History()
	native.SetNextItemWidth(-1)
	if native.InputTextCommand("##cmd-input", commandInputBuf, history, &commandHistoryCursor, comps, &commandCompSel) {
		_ = cl.Submit(s, bufString(commandInputBuf))
		clearBuf(commandInputBuf)
		commandHistoryCursor = int32(len(cl.Scrollback().History())) // back to the empty line
		commandCompletions, commandCompForBuf, commandCompSel = nil, "", 0
		commandFocusNext = true
	}
}
