//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command editorshot is a throwaway live-capture driver for the Script Console code editor: it
// opens the real native window, wires a Script Console runtime, seeds the editor with a sample
// Lua program, runs the production DrawChrome loop for a few frames, and saves the whole window
// to a PNG — so the image is exactly what the app draws (gutter, highlighting, caret).
//
//	go run ./head/cmd/editorshot -out /tmp/editor.png
package main

import (
	"flag"
	"fmt"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/script/console"
	"oblikovati.org/script/gopherlua"
	"oblikovati.org/script/runner"
)

// sampleSource exercises every token class the highlighter handles plus the API namespace, so
// the capture proves keywords, strings, numbers, comments and identifiers all colour.
const sampleSource = `-- Script Console: parametric block
local width = 40.0      -- millimetres
local depth = 0x1E
function build(name)
    oblikovati.documents.create{ name = name }
    oblikovati.sketch.rectangle{ width = width, depth = depth }
    print("built " .. name)  --[[ trailing
       long comment ]]
    return true
end

build("bracket")`

// stubCaller satisfies client.Caller without a live host — the capture only renders the editor,
// it does not Run the script.
type stubCaller struct{}

func (stubCaller) Call(method string, req []byte) ([]byte, error) { return []byte(`{}`), nil }

// apiMethods is a representative slice of the dotted wire-method names, so the (later)
// autocomplete popup has a tree to walk in captures.
func apiMethods() []string {
	return []string{"documents.create", "documents.activate", "sketch.rectangle", "sketch.circle"}
}

// brokenSource has a deliberate syntax error (an `if` with no `then`) so the diagnostics capture
// shows the red underline and gutter marker.
const brokenSource = `local n = 10
if n > 5
    print("big")
end`

func main() {
	out := flag.String("out", "/tmp/editor.png", "window PNG output path")
	frames := flag.Int("frames", 6, "frames to render before capture")
	broken := flag.Bool("broken", false, "seed a syntax error and show diagnostics")
	flag.Parse()
	if err := run(*out, *frames, *broken); err != nil {
		fmt.Fprintln(os.Stderr, "editorshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out)
}

func run(out string, frames int, broken bool) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	if _, err := s.NewPart(); err != nil {
		return err
	}
	wireConsole()
	s.OpenScriptConsole()
	ui.SetScriptMethods(apiMethods)
	ui.FocusScriptEditor()
	if broken {
		ui.SetScriptSource(brokenSource)
		ui.ForceScriptDiagnostics()
	} else {
		ui.SetScriptSource(sampleSource)
		ui.SetScriptCaret(4, 15)     // just after the '.' of oblikovati.documents on line 5
		ui.TriggerScriptCompletion() // open the autocomplete popup at the caret
	}

	win, err := native.CreateWindow(1280, 800, "editorshot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	for i := 0; i < frames; i++ {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveWindowPNG(out)
}

// wireConsole installs a Script Console runtime backed by a stub caller, so the panel renders
// (a nil controller would degrade to an "unavailable" message).
func wireConsole() {
	run := runner.New(gopherlua.New(), stubCaller{}, apiMethods)
	ui.SetScriptController(console.NewController(run, runner.DefaultGUILimits))
}
