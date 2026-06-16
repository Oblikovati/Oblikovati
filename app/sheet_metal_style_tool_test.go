// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/param"
)

// TestSheetMetalStyleSeed Start loads the rule's gauge/radius/K-factor/relief into the buffers,
// converting lengths to the document's preferred unit.
func TestSheetMetalStyleSeed(t *testing.T) {
	s, part := faceSheet(t, 4)
	rule := part.SheetMetal()
	if rule == nil {
		t.Fatal("base sheet has no rule")
	}
	tool := NewSheetMetalStyleTool()
	tool.Start(s)

	wantThk := s.DocumentUnits().ToPreferred(param.Q(rule.Thickness(), param.Length))
	if got := tool.Thickness(); !nearly(got, wantThk) {
		t.Errorf("seeded thickness = %v, want %v (display units)", got, wantThk)
	}
	if got := tool.KFactor(); !nearly(got, rule.Unfold().KFactor) {
		t.Errorf("seeded K-factor = %v, want %v", got, rule.Unfold().KFactor)
	}
	if got := tool.ReliefShapeIndex(); got != int(rule.Relief().Shape) {
		t.Errorf("seeded relief shape = %d, want %d", got, int(rule.Relief().Shape))
	}
}

// TestSheetMetalStyleCommit a committed edit re-authors the rule (thickness/radius/K-factor/
// relief) and keeps the part recomputable.
func TestSheetMetalStyleCommit(t *testing.T) {
	s, part := faceSheet(t, 4)
	rule := part.SheetMetal()
	before := rule.Thickness()

	tool := NewSheetMetalStyleTool()
	tool.Start(s)
	tool.SetThickness(tool.Thickness() * 2) // double the gauge (display units)
	tool.SetKFactor(0.33)
	tool.SetReliefShapeIndex(int(types.ReliefSquare))
	if !tool.CanCommit() {
		t.Fatal("positive gauge/radius and 0<k<1 should allow commit")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if got := rule.Thickness(); !nearly(got, before*2) {
		t.Errorf("thickness after commit = %v, want %v", got, before*2)
	}
	if got := rule.Unfold().KFactor; !nearly(got, 0.33) {
		t.Errorf("K-factor after commit = %v, want 0.33", got)
	}
	if got := rule.Relief().Shape; got != types.ReliefSquare {
		t.Errorf("relief shape after commit = %v, want square", got)
	}
}

// TestSheetMetalStyleCanCommit rejects a non-physical K-factor and a zero gauge.
func TestSheetMetalStyleCanCommit(t *testing.T) {
	s, _ := faceSheet(t, 4)
	tool := NewSheetMetalStyleTool()
	tool.Start(s)
	for _, c := range []struct {
		name string
		set  func()
		want bool
	}{
		{"seeded defaults are valid", func() {}, true},
		{"zero thickness", func() { tool.SetThickness(0) }, false},
		{"k = 0", func() { tool.SetThickness(0.3); tool.SetKFactor(0) }, false},
		{"k = 1.5", func() { tool.SetKFactor(1.5) }, false},
		{"valid again", func() { tool.SetKFactor(0.4) }, true},
	} {
		c.set()
		if got := tool.CanCommit(); got != c.want {
			t.Errorf("%s: CanCommit = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestSheetMetalStyleStartPlainPart Start on a non-sheet-metal part is a silent no-op (the
// buffers stay zero), and Commit reports the missing rule.
func TestSheetMetalStyleStartPlainPart(t *testing.T) {
	s := newSessionWithPart(t)
	tool := NewSheetMetalStyleTool()
	tool.Start(s) // no panic, no seed
	if tool.Thickness() != 0 {
		t.Errorf("plain part seeded thickness = %v, want 0", tool.Thickness())
	}
	if err := tool.Commit(s); err == nil {
		t.Error("Commit on a non-sheet-metal part should error")
	}
}

// nearly compares two lengths within a tight tolerance (unit reconversion adds float noise).
func nearly(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}
