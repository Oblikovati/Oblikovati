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

// Console layout: the editor fills the space above a user-resizable output pane, so growing the
// window grows the EDITOR (the output keeps its set height). A draggable splitter between them
// adjusts that height.
const (
	splitterThickness = 6  // grab height of the editor/output splitter (px)
	minEditorHeight   = 80 // the editor never collapses below this
	minOutputHeight   = 48 // nor the output pane
)

// scriptOutputHeight is the output pane's height (px), adjusted by dragging the splitter and
// kept across frames. scriptOutputLastCount tracks output growth so the pane auto-scrolls to the
// tail only when new lines arrive (leaving the user free to scroll back otherwise).
var (
	scriptOutputHeight    float32 = 150
	scriptOutputLastCount int
)

// colSplitterGrip tints the splitter handle (brighter while dragged is left to ImGui's hover).
var colSplitterGrip = [4]float32{0.45, 0.46, 0.52, 0.7}

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

// OpenScriptFind opens the find bar pre-filled with query (and selects the first hit). Exposed
// for capture drivers / tests.
func OpenScriptFind(query string) {
	e := scriptCodeEditor()
	for i := range e.find.query {
		e.find.query[i] = 0
	}
	copy(e.find.query, query)
	e.find.active = true
	e.recomputeMatches()
}

// ForceScriptDiagnostics runs the syntax check immediately (bypassing the debounce), so a capture
// driver can show error underlines without waiting real time. Exposed for capture drivers / tests.
func ForceScriptDiagnostics() {
	e := scriptCodeEditor()
	src, now := e.model.Text(), time.Now()
	e.analyzer.Observe(src, now)
	e.analyzer.Observe(src, now.Add(time.Hour)) // settle past the debounce window
}

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
	visible, open := native.BeginClosable("Script Console")
	if visible {
		drawScriptConsoleBody()
	}
	native.End()
	if !open { // the title-bar X was clicked
		s.CloseScriptConsole()
	}
}

// drawScriptConsoleBody renders the console contents: toolbar, the editor (filling the space
// above the output pane), a draggable splitter, the status line, and the output pane.
func drawScriptConsoleBody() {
	if scriptController == nil {
		native.Text("Script runtime unavailable.")
		return
	}
	snap := scriptController.Console().Snapshot()
	drawScriptToolbar(snap.Running)
	scriptCodeEditor().DrawFindBar()
	scriptCodeEditor().Draw(0, scriptEditorHeight())
	drawOutputSplitter()
	drawScriptStatus(snap)
	drawScriptOutput(snap)
}

// scriptEditorHeight returns the editor height for this frame: the remaining content height
// minus the output pane, the splitter and the status line — so the editor absorbs window
// resizing while the output keeps its set height. It also re-clamps the output height.
func scriptEditorHeight() float32 {
	_, availH := native.ContentRegionAvail()
	clampScriptOutputHeight(availH)
	h := availH - scriptOutputHeight - splitterThickness - native.FrameHeight()
	if h < minEditorHeight {
		h = minEditorHeight
	}
	return h
}

// clampScriptOutputHeight keeps the output height within [minOutputHeight, max] where max leaves
// the editor at least minEditorHeight — so the splitter cannot squeeze either pane away.
func clampScriptOutputHeight(availH float32) {
	maxOut := availH - minEditorHeight - splitterThickness - native.FrameHeight()
	if scriptOutputHeight > maxOut {
		scriptOutputHeight = maxOut
	}
	if scriptOutputHeight < minOutputHeight {
		scriptOutputHeight = minOutputHeight
	}
}

// drawOutputSplitter draws the draggable divider between the editor and the output pane.
// Dragging it down grows the editor (shrinks the output); a grip line marks it.
func drawOutputSplitter() {
	availW, _ := native.ContentRegionAvail()
	native.InvisibleButton("##script-split", availW, splitterThickness)
	if native.IsItemActive() {
		_, dy := native.MouseDelta()
		scriptOutputHeight -= dy
	}
	x, y := native.ItemRectMin()
	mid := y + splitterThickness/2
	native.DrawRectFilled(x, mid-1, x+availW, mid+1, colSplitterGrip)
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

// drawScriptOutput renders the captured print() output in a fixed-height scrollable pane that
// auto-scrolls to the tail when new lines arrive (so a chatty run stays pinned to the bottom),
// while leaving the user free to scroll back when no new output is coming.
func drawScriptOutput(snap console.Snapshot) {
	if !native.BeginChild("##script-output", 0, scriptOutputHeight, true) {
		native.EndChild()
		return
	}
	for _, line := range snap.Output {
		native.Text(line)
	}
	if len(snap.Output) > scriptOutputLastCount {
		native.SetScrollHereY() // new lines since last frame: follow the tail
	}
	scriptOutputLastCount = len(snap.Output)
	native.EndChild()
}
