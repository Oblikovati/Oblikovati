//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestToolDialogRegistryPopulated pins that the bespoke dialogs migrated off chrome.go's
// roll-call are present in the default set (audit I4) — a dialog dropped from
// defaultToolDialogSet would silently stop drawing, so this is the parity guard that made
// deleting the roll-call safe.
func TestToolDialogRegistryPopulated(t *testing.T) {
	want := []string{
		"ChamferTool", "CoilTool", "DeleteFaceTool", "DraftTool", "ExtrudeTool",
		"FaceFilletTool", "FaceOffsetTool", "FilletTool", "FullRoundFilletTool", "HoleTool",
		"LoftTool", "ReplaceFaceTool", "RevolveTool", "ShellTool", "SheetMetalFaceTool",
		"SplitTool", "SweepTool", "ThickenTool", "ThreadTool",
	}
	got := map[string]struct{}{}
	for _, name := range registeredDialogTools() {
		got[name] = struct{}{}
	}
	for _, name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("tool %q lost its dialog registration — its init() no longer calls registerToolDialog", name)
		}
	}
	if len(toolDialogs.draws) == 0 {
		t.Fatal("no dialog draws in the default set — chrome would draw no tool panels")
	}
}

// TestRegisterToolDialogPanicsOnDuplicate asserts a second dialog claiming an
// already-registered tool key fails at construction, not in a live session (the house
// registry contract, serialize_registry #1416). It uses a fresh set so it does not mutate
// the shared default.
func TestRegisterToolDialogPanicsOnDuplicate(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("registerToolDialog did not panic on a duplicate tool key")
		}
	}()
	s := newToolDialogSet()
	s.registerToolDialog(func(*app.Session) {}, "ChamferTool")
	s.registerToolDialog(func(*app.Session) {}, "ChamferTool")
}
