//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati/app"
)

// TestInWindowRibbonUploadsIconTextures drives the real chrome for a few frames and
// asserts the 3D Model tab's large-icon command (Extrude) was rasterized from its SVG
// and uploaded as a live ImGui/Vulkan texture — exercising the whole SVG→RGBA→GPU path
// on real hardware, not just the pure-Go rasterizer. Skips when no display/Vulkan is
// available (e.g. CI without a GPU).
func TestInWindowRibbonUploadsIconTextures(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false // rebuild the default layout for this fresh window/context
	icons = nil         // rebind the icon cache to this fresh window

	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if _, err := s.NewPart(); err != nil { // a part must be active for the Part ribbon (3D Model/Extrude) to show
		t.Fatalf("NewPart: %v", err)
	}
	for i := 0; i < 3; i++ { // settle the dock/tab layout, then draw the ribbon's icons
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	if icons == nil {
		t.Fatal("DrawChrome did not initialize the icon cache")
	}
	tex, ok := icons.tex[iconKey{name: "extrude", px: largeIconPx}]
	if !ok {
		t.Fatal("ribbon never uploaded the Extrude large-icon texture (was the 3D Model tab drawn?)")
	}
	if tex == 0 {
		t.Error("Extrude icon texture handle is 0 — SVG rasterization or the Vulkan upload failed")
	}
}
