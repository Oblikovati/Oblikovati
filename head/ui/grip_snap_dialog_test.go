//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestGripSnapDialogRenders opens a real window and renders the Grip Snap Move-Options panel for an
// active tool (the Constraint combo + commit buttons), and the no-op early return when no grip-snap
// tool is running. It skips cleanly where no display/Vulkan is available.
func TestGripSnapDialogRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil // rebind the icon cache to this fresh window/context

	// Early return: no grip-snap tool active → the panel draws nothing.
	idle := app.NewSession()
	win.BeginFrame()
	drawGripSnapDialog(idle)
	win.EndFrame(0.1, 0.1, 0.1)

	// Active tool → the full Move-Options panel renders (Constraint combo + OK/Cancel).
	s := app.NewSession()
	s.StartTool(app.NewGripSnapTool())
	if s.ActiveGripSnap() == nil {
		t.Fatal("Grip Snap tool not active after StartTool")
	}
	win.BeginFrame()
	drawGripSnapDialog(s)
	win.EndFrame(0.1, 0.1, 0.1)
}
