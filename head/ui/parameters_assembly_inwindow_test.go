//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// TestInWindowAssemblyParametersDialogDraws drives the real chrome with the Parameters dialog
// open over an ACTIVE ASSEMBLY that holds a user parameter and links a parameter from a part,
// with the Link… picker open. It guards that the assembly Parameters path and the new
// derived-tables section + link picker render without crashing (M39-F04, #1560). Skips when
// no display/Vulkan is available.
func TestInWindowAssemblyParametersDialogDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := app.NewSession()
	// A part source with a numeric user parameter, so the link picker has a candidate.
	gears, err := compdef.AddPart(s.Workspace(), "gears.obk", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := s.AddNumericUserParameter("module", "2 mm"); err != nil {
		t.Fatalf("seed source param: %v", err)
	}
	// The assembly, made active, with its own user parameter.
	asm, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asm); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	if err := s.AddNumericUserParameter("plateWidth", "40 mm"); err != nil {
		t.Fatalf("add assembly param: %v", err)
	}
	s.OpenParameters()
	openLinkPicker() // exercise the derived-table source picker too

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	if !s.ParametersOpen() {
		t.Error("Parameters dialog should still be open after drawing on an assembly")
	}
	// The picker should have listed the part as a linkable source.
	if got := s.LinkableSourceDocuments(); len(got) != 1 || got[0].FullName != gears.FullDocumentName() {
		t.Errorf("linkable sources = %+v, want [%s]", got, gears.FullDocumentName())
	}
}
