//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// TestInWindowParametersDialogDraws drives the real chrome with the Parameters dialog
// open over a part that has every parameter flavor, for a few frames. It exercises the
// table bindings, the multi-value combo, and the value cells on real hardware — a guard
// that the dialog renders without crashing. Skips when no display/Vulkan is available.
func TestInWindowParametersDialogDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "Part1", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := s.AddNumericUserParameter("width", "20 mm"); err != nil {
		t.Fatalf("AddNumericUserParameter: %v", err)
	}
	_ = s.AddTextUserParameter("finish", "anodized")
	_ = s.AddBooleanUserParameter("vented", true)
	id, _ := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Parameters().ByName("width")
	_ = s.SetParameterValueList(id.ID(), []string{"20 mm", "25 mm"}, true)
	s.OpenParameters()

	for range 3 {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	if !s.ParametersOpen() {
		t.Error("Parameters dialog should still be open after drawing")
	}
}
