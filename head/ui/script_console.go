//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"time"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console"
)

// scriptController is the head's live Script Console runtime (engine + dispatched caller),
// injected by main once the add-in host's router + dispatcher are wired. It stays nil in
// the headless UI tests and before injection, so the panel degrades to an inert message
// rather than crashing.
var scriptController *console.Controller

// scriptSourceBuf is the editable Lua source the console edits in place. 64 KiB is ample
// for an interactive console script; it persists across frames so the editor keeps text.
var scriptSourceBuf = make([]byte, 64<<10)

// editorHeight is the fixed height (logical px) of the source pane; the output pane below
// fills the remaining window height.
const editorHeight = 260

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
	native.InputTextMultiline("##script-source", scriptSourceBuf, 0, editorHeight)
	native.Separator()
	drawScriptStatus(snap)
	drawScriptOutput(snap)
}

// drawScriptToolbar renders Run (disabled mid-run), Stop (disabled when idle), and Clear.
func drawScriptToolbar(running bool) {
	native.BeginDisabled(running)
	if native.Button("Run") {
		_ = scriptController.Run(bufString(scriptSourceBuf))
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
