//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"time"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console"
	"oblikovati.org/script/console/textbuf"
)

// scriptController is the head's live Script Console runtime (engine + dispatched caller),
// injected by main once the add-in host's router + dispatcher are wired. It stays nil in
// the headless UI tests and before injection, so the panel degrades to an inert message
// rather than crashing.
var scriptController *console.Controller

// scriptEditor is the console's Lua code editor (syntax-highlighted, gutter, autocomplete),
// created on first use so the mono font and runtime are ready. It persists across frames so the
// edited source and caret survive between draws.
var scriptEditor *codeEditor

// editorHeight is the fixed height (logical px) of the source pane; the output pane below
// fills the remaining window height.
const editorHeight = 260

// scriptMethods provides the host's dotted wire-method names for autocomplete; injected at
// startup alongside the controller (the router's Methods).
var scriptMethods func() []string

// scriptCodeEditor returns the lazily-created code editor, seeding its completion engine from
// the host method list when one has been injected.
func scriptCodeEditor() *codeEditor {
	if scriptEditor == nil {
		scriptEditor = newCodeEditor("")
		if scriptMethods != nil {
			scriptEditor.setMethods(scriptMethods())
		}
	}
	return scriptEditor
}

// SetScriptMethods injects the host's wire-method-name provider for autocomplete. Called at head
// startup with the router's Methods; safe to call before or after the editor exists.
func SetScriptMethods(methods func() []string) {
	scriptMethods = methods
	if scriptEditor != nil && methods != nil {
		scriptEditor.setMethods(methods())
	}
}

// SetScriptSource replaces the Script Console editor's text. Exposed for capture drivers and
// in-window tests that need to seed the editor before a frame.
func SetScriptSource(s string) { scriptCodeEditor().SetText(s) }

// FocusScriptEditor forces the editor's keyboard focus on (so a capture shows the caret without
// synthesising a click). Exposed for capture drivers / tests.
func FocusScriptEditor() { scriptCodeEditor().focused = true }

// SetScriptCaret moves the editor caret to (line, col). Exposed for capture drivers / tests
// (e.g. to park the caret on a bracket so the match highlight shows).
func SetScriptCaret(line, col int) {
	scriptCodeEditor().model.SetCaret(textbuf.Position{Line: line, Col: col}, false)
}

// TriggerScriptCompletion forces the autocomplete popup open at the caret. Exposed for capture
// drivers / tests.
func TriggerScriptCompletion() { scriptCodeEditor().refreshCompletion(true) }

// SetScriptController injects the console runtime. Called once at head startup, after the
// add-in host (and thus the dispatcher the script's host calls hop through) exists.
func SetScriptController(c *console.Controller) { scriptController = c }

// drawScriptConsole renders the Manage ▸ Script Console panel when open: a Lua source
// editor, Run/Stop/Clear controls, a streamed output pane, and the last run's status. The
// run executes off the UI thread (the Controller), so a looping script never freezes the
// frame loop (ADR-0028 §5).
func drawScriptConsole(s *app.Session) {
	if !s.ScriptConsoleOpen() {
		return
	}
	native.SetNextWindowSizeOnce(640, 560)
	if native.Begin("Script Console") {
		drawScriptConsoleBody()
	}
	native.End()
}

// drawScriptConsoleBody renders the console contents once the window is open.
func drawScriptConsoleBody() {
	if scriptController == nil {
		native.Text("Script runtime unavailable.")
		return
	}
	snap := scriptController.Console().Snapshot()
	drawScriptToolbar(snap.Running)
	scriptCodeEditor().Draw(0, editorHeight)
	native.Separator()
	drawScriptStatus(snap)
	drawScriptOutput(snap)
}

// drawScriptToolbar renders Run (disabled mid-run), Stop (disabled when idle), and Clear.
func drawScriptToolbar(running bool) {
	native.BeginDisabled(running)
	if native.Button("Run") {
		_ = scriptController.Run(scriptCodeEditor().Text())
	}
	native.EndDisabled()
	native.SameLine()
	native.BeginDisabled(!running)
	if native.Button("Stop") {
		scriptController.Stop()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Clear") {
		scriptController.Console().Clear()
	}
}

// drawScriptStatus shows whether a run is in flight or how the last one ended.
func drawScriptStatus(snap console.Snapshot) {
	switch {
	case snap.Running:
		native.Text("Running…")
	case !snap.HasLast:
		native.Text("Ready.")
	case snap.Last.Err != nil:
		native.Text("Error: " + snap.Last.Err.Error())
	default:
		native.Text(fmt.Sprintf("Done in %s.", snap.Last.Duration.Round(time.Millisecond)))
	}
}

// drawScriptOutput renders the captured print() output in a scrollable pane that follows
// the tail as new lines arrive.
func drawScriptOutput(snap console.Snapshot) {
	if !native.BeginChild("##script-output", 0, 0, true) {
		native.EndChild()
		return
	}
	for _, line := range snap.Output {
		native.Text(line)
	}
	if snap.Running {
		native.SetScrollHereY()
	}
	native.EndChild()
}
