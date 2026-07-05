//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// TestInWindowShiftLetterStartsTool drives a real Shift+L keystroke with the viewport focused in a
// sketch and asserts it starts the Line tool — the #1751 S3 curated shortcut through the full head
// path. It also guards the S1/S2 boundary: a MODIFIED letter must dispatch as a shortcut, never be
// routed to command-window typing (that path is for BARE letters only). A bare "l" here would seed
// the command line and leave no active tool.
func TestInWindowShiftLetterStartsTool(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	docu, _ := compdef.AddPart(s.Workspace(), "shortcut.opd", true)
	part := docu.Content().(*compdef.PartComponentDefinition)
	sk := part.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	s.TickCameraAnimation(100) // finish the enter-sketch swing (test dt≈0)
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)

	// Keep keyboard focus on the VIEWPORT (not the command line): a Shift-only chord fires as a
	// viewport shortcut only when no text field owns typing (a focused command line would capitalise
	// the letter instead — only Ctrl/Alt chords bypass it). This is the state after the user clicks
	// into the 3D view. Suppress the command line's default focus-grab BEFORE the first frame so it
	// never claims the keyboard. commandFocusNext/commandInputBuf are package globals; reset for others.
	commandFocusNext = false
	clearBuf(commandInputBuf)
	defer func() { commandFocusNext = true; clearBuf(commandInputBuf) }()

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	frame()
	frame()

	// Shift+L with the viewport focused → the Line tool starts (not command-window typing).
	// Hold Shift on a prior frame so io.KeyShift is settled before the letter is pressed (ImGui
	// derives the modifier flag from the physical key's down-state a frame after the event).
	native.InjectKeyShift(true)
	frame()
	native.InjectLetter(int('l'-'a'), true)
	frame()
	native.InjectLetter(int('l'-'a'), false)
	native.InjectKeyShift(false)
	frame()

	ti := s.ActiveTool()
	name := "<none>"
	if ti != nil {
		name = ti.Name()
	}
	if name != "Line" {
		t.Fatalf("Shift+L in a sketch should start the Line tool, got %q (and the command window must not have eaten it)", name)
	}
	if got := bufString(commandInputBuf); got != "" {
		t.Errorf("Shift+L must dispatch, not type into the command line; buffer = %q", got)
	}
}
