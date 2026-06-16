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

// TestConvertRejectsNoPart convert errors when there is no active part.
func TestConvertRejectsNoPart(t *testing.T) {
	if err := NewSession().ConvertActiveToSheetMetal(); err == nil {
		t.Error("convert with no active document should error")
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
