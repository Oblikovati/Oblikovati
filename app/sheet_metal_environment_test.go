// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestConvertActiveToSheetMetal converting an ordinary part enters the sheet-metal environment
// (the part reports sheet metal and the tab's tools enable); converting again is a no-op.
func TestConvertActiveToSheetMetal(t *testing.T) {
	s := newSessionWithPart(t)
	if hasActiveSheetMetalPart(s) {
		t.Fatal("a fresh part should not start as sheet metal")
	}
	if err := s.ConvertActiveToSheetMetal(); err != nil {
		t.Fatalf("ConvertActiveToSheetMetal: %v", err)
	}
	if !hasActiveSheetMetalPart(s) {
		t.Error("after convert the part should be sheet metal")
	}
	if err := s.ConvertActiveToSheetMetal(); err != nil {
		t.Errorf("converting an already sheet-metal part should be a no-op, got %v", err)
	}
}

// TestConvertRejectsNoPart convert errors when there is no active document and when the active
// document is not a part (an assembly).
func TestConvertRejectsNoPart(t *testing.T) {
	if err := NewSession().ConvertActiveToSheetMetal(); err == nil {
		t.Error("convert with no active document should error")
	}
	s := NewSession()
	if _, err := s.NewAssembly(); err != nil {
		t.Fatalf("NewAssembly: %v", err)
	}
	if err := s.ConvertActiveToSheetMetal(); err == nil {
		t.Error("convert on an assembly (non-part) should error")
	}
}

// TestNewSheetMetalPart a new sheet-metal part starts already in the environment.
func TestNewSheetMetalPart(t *testing.T) {
	s := NewSession()
	if _, err := s.NewSheetMetalPart(); err != nil {
		t.Fatalf("NewSheetMetalPart: %v", err)
	}
	if !hasActiveSheetMetalPart(s) {
		t.Error("a new sheet-metal part should report as sheet metal")
	}
}

// TestSheetMetalEnvironmentCommands the two ribbon commands run their actions: New Sheet Metal
// Part starts in the environment, and Convert enters it on the active ordinary part.
func TestSheetMetalEnvironmentCommands(t *testing.T) {
	fresh := NewSession()
	if err := RegisterStandardCommands(fresh); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if err := fresh.Execute("GetStarted.NewSheetMetalPart"); err != nil {
		t.Fatalf("New Sheet Metal Part command: %v", err)
	}
	if !hasActiveSheetMetalPart(fresh) {
		t.Error("New Sheet Metal Part should leave a sheet-metal part active")
	}

	ordinary := newSessionWithPart(t)
	if err := RegisterStandardCommands(ordinary); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if err := ordinary.Execute("SheetMetal.Convert"); err != nil {
		t.Fatalf("Convert command: %v", err)
	}
	if !hasActiveSheetMetalPart(ordinary) {
		t.Error("Convert command should make the active part sheet metal")
	}
}

// TestCanConvertPredicate the Convert enable predicate is true only for an ordinary part.
func TestCanConvertPredicate(t *testing.T) {
	if canConvertToSheetMetal(NewSession()) {
		t.Error("canConvertToSheetMetal should be false with no part")
	}
	ordinary := newSessionWithPart(t)
	if !canConvertToSheetMetal(ordinary) {
		t.Error("canConvertToSheetMetal should be true for an ordinary part")
	}
	sheetMetal, _ := sheetMetalSession(t)
	if canConvertToSheetMetal(sheetMetal) {
		t.Error("canConvertToSheetMetal should be false once the part is sheet metal")
	}
}
