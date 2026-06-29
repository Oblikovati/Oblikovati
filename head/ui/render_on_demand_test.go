//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/head/internal/native"
)

// TestWantsAnimationFrameIdleVsInteracting is the #1493 regression guard for the render-on-demand
// loop's wake predicate. It drives the FULL chrome (so the viewport's input-capturing
// InvisibleButton is present) and asserts WantsAnimationFrame is FALSE when idle — the first cut
// used IsAnyItemActive, which that button reads as active every frame, pinning the loop on and
// leaving idle CPU unchanged. With a mouse button held it must read TRUE so an in-progress drag
// never stalls. This is the behaviour that lets the loop block (idle CPU ~0) yet stay responsive.
func TestWantsAnimationFrameIdleVsInteracting(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false // rebuild the default layout for this fresh window/context
	icons = nil         // rebind the icon cache to this fresh window (it holds GPU handles)
	s := framedSession()

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	// Settle the dock layout and the first-frame mouse delta with no synthetic input.
	frame()
	frame()
	frame()
	if win.WantsAnimationFrame() {
		t.Fatal("WantsAnimationFrame is true while idle (no input) — the loop would never block, " +
			"so idle CPU stays pinned (#1493); is the predicate using a stuck signal like IsAnyItemActive?")
	}

	// A held mouse button means a drag is in progress: the loop must keep drawing.
	native.InjectMouseButton(0, true)
	frame()
	if !win.WantsAnimationFrame() {
		t.Fatal("WantsAnimationFrame is false while a mouse button is held — an in-progress drag would stall (#1493)")
	}
	native.InjectMouseButton(0, false)
}
