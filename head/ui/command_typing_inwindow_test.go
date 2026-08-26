//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
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
//
// This exact single-frame timing has shown up as intermittently flaky on a slower/more heavily
// loaded CI runner (#2161) — not reproducible locally, and not something a fixed number of extra
// settle frames safely papers over (tried; it just moves which frame the race lands on, and once
// even flipped a DIFFERENT assertion by letting the command window's own default initial-focus
// grab — commandFocusNext starts true — land before typing began). Retrying the whole attempt
// with a fresh window/session is the safe way to relax a genuine one-shot timing race without
// weakening what's actually being pinned: bareLetterAttempt either lands "line" or it doesn't,
// full stop, on every attempt — only how MANY attempts get to try is relaxed.
func TestInWindowBareLetterTypesIntoCommandWindow(t *testing.T) {
	const attempts = 3
	var lastErr string
	for range attempts {
		lastErr = bareLetterAttempt(t)
		if lastErr == "" {
			return
		}
	}
	t.Fatalf("%s (failed on all %d attempts)", lastErr, attempts)
}

// bareLetterAttempt is one full attempt at TestInWindowBareLetterTypesIntoCommandWindow's
// scenario, returning a non-empty description of what went wrong instead of failing the test
// directly, so the caller can retry on a fresh window/session.
func bareLetterAttempt(t *testing.T) string {
	t.Helper()
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
		return "no text field should own the keyboard before typing begins"
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
		return fmt.Sprintf("command input = %q, want \"line\" (a bare letter must focus + seed the command window exactly once)", got)
	}
	if !native.WantTextInput() {
		return "after typing a letter the command window input should own the keyboard"
	}
	return ""
}
