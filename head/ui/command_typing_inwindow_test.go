//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/head/internal/native"
)

// TestSeedCommandInput covers the pure seeder without a window: it appends the handed-off
// character to the input buffer (preserving existing text) and requests focus (#1751 S2).
func TestSeedCommandInput(t *testing.T) {
	defer func() { clearBuf(commandInputBuf); commandFocusNext = false }()
	clearBuf(commandInputBuf)
	commandFocusNext = false

	seedCommandInput("l")
	if got := bufString(commandInputBuf); got != "l" {
		t.Errorf("seed into empty buffer = %q, want \"l\"", got)
	}
	if !commandFocusNext {
		t.Error("seeding must request keyboard focus for the input")
	}

	seedCommandInput("i") // appends, so a partially-typed command is never clobbered
	if got := bufString(commandInputBuf); got != "li" {
		t.Errorf("seed append = %q, want \"li\"", got)
	}
}

// TestInWindowBareLetterTypesIntoCommandWindow drives real letter keystrokes with the viewport
// focused and asserts they land in the docked Command Window input — the #1751 S2 behaviour:
// pressing a bare letter means "I want to type a command", so keyboard focus moves to the command
// line seeded with that letter and subsequent letters extend it. It exists to pin the ImGui
// char-queue timing on the focus-grab frame: the triggering character must land exactly once
// (no drop, no double), yielding "line" — not "ine" or "lline".
func TestInWindowBareLetterTypesIntoCommandWindow(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false // rebuild the default layout (docks the command window at the bottom)
	icons = nil         // rebind the icon cache to this fresh window/context
	clearBuf(commandInputBuf)
	commandFocusNext = true
	defer clearBuf(commandInputBuf)
	s := framedSession()

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	// Settle the dock layout; no text field should own the keyboard before typing begins.
	frame()
	frame()
	if native.WantTextInput() {
		t.Fatal("no text field should own the keyboard before typing begins")
	}

	// Type "line" one faithful keystroke at a time: the down-frame emits the key + input char,
	// the up-frame releases so the next letter registers as a fresh press.
	for _, r := range "line" {
		native.InjectLetter(int(r-'a'), true)
		frame()
		native.InjectLetter(int(r-'a'), false)
		frame()
	}

	_ = win.SaveWindowPNG(filepath.Join(outDir(), "1751-command-typing.png"))
	if got := bufString(commandInputBuf); got != "line" {
		t.Fatalf("command input = %q, want \"line\" (a bare letter must focus + seed the command window exactly once)", got)
	}
	if !native.WantTextInput() {
		t.Error("after typing a letter the command window input should own the keyboard")
	}
}
