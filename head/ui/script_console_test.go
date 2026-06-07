//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"
	"time"

	"oblikovati/addin/opregistry"
	"oblikovati/addin/router"
	"oblikovati/script"
	"oblikovati/script/bridge"
	"oblikovati/script/console"
	"oblikovati/script/gopherlua"
	"oblikovati/script/runner"
)

// directConsoleController builds a Script Console controller over a synchronous in-proc
// caller (no dispatcher needed in a test) against a fresh session+router, so a run drives
// the real Lua engine + wire surface.
func directConsoleController() *console.Controller {
	s := framedSession()
	rtr := router.New(opregistry.Default())
	caller := bridge.NewDirectCaller(rtr.Handle, s)
	run := runner.New(gopherlua.New(), caller, rtr.Methods)
	return console.NewController(run, script.Limits{Wall: 5 * time.Second})
}

// TestInWindowScriptConsoleRendersAndRuns opens the real window with the Script Console
// visible and renders frames — so a mismatched ImGui Begin/End in the editor, output child,
// or disabled toolbar (the new InputTextMultiline/BeginChild verbs) would trip Dear ImGui's
// assertions. It then runs a script through the injected controller and asserts the streamed
// print() output reaches the panel's snapshot.
func TestInWindowScriptConsoleRendersAndRuns(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	ctrl := directConsoleController()
	SetScriptController(ctrl)
	defer SetScriptController(nil)
	s.OpenScriptConsole()
	defer s.CloseScriptConsole()

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	// A couple of frames so the console window appears and its panes render.
	frame()
	frame()

	if err := ctrl.Run(`print("hello from lua"); print(#oblikovati.methods() > 0)`); err != nil {
		t.Fatalf("Run: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for ctrl.Console().Running() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	// Render a few more frames so the output pane draws the streamed lines.
	frame()
	frame()

	snap := ctrl.Console().Snapshot()
	if !snap.HasLast || snap.Last.Err != nil {
		t.Fatalf("script run failed: %+v", snap.Last)
	}
	if len(snap.Output) == 0 || snap.Output[0] != "hello from lua" {
		t.Fatalf("console output = %v, want it to start with the printed line", snap.Output)
	}
}

// TestInWindowScriptConsoleWithoutControllerIsInert: with no runtime injected (the headless
// default), the open console renders an inert message instead of crashing.
func TestInWindowScriptConsoleWithoutControllerIsInert(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	SetScriptController(nil)
	s := framedSession()
	s.OpenScriptConsole()
	defer s.CloseScriptConsole()

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}
