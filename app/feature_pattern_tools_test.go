// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
)

// extrudedPart builds a part with one extrude feature (the source to pattern) and returns
// the session, the part definition, and the extrude's feature handle.
func extrudedPart(t *testing.T) (*Session, *compdef.PartComponentDefinition, FeatureHandle) {
	t.Helper()
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(120, 90)
	ext.SetDistance(5)
	if err := s.OK(); err != nil {
		t.Fatalf("extrude OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() != 1 {
		t.Fatalf("expected 1 source feature, got %d", def.Features().Count())
	}
	return s, def, FeatureHandle{Feature: def.Features().Item(0)}
}

func TestFeatureRectPatternTool(t *testing.T) {
	s, def, src := extrudedPart(t)
	tool := NewFeatureRectPatternTool()
	tool.countX = 3 // a 3×1 grid → 2 new copies
	s.StartTool(tool)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("rectangular pattern OK: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Fatal("pattern tool should deactivate after OK")
	}
	if def.SurfaceBodies().Count() != 3 {
		t.Fatalf("bodies after 3×1 pattern = %d, want 3", def.SurfaceBodies().Count())
	}
}

func TestFeatureMirrorTool(t *testing.T) {
	s, def, src := extrudedPart(t)
	tool := NewFeatureMirrorTool() // default normal +X → mirror across YZ
	s.StartTool(tool)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("mirror OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("bodies after mirror = %d, want 2 (original + reflection)", def.SurfaceBodies().Count())
	}
}

func TestFeatureCircPatternTool(t *testing.T) {
	s, def, src := extrudedPart(t)
	tool := NewFeatureCircPatternTool() // count 4 over 360°
	s.StartTool(tool)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("circular pattern OK: %v", err)
	}
	if def.SurfaceBodies().Count() <= 1 {
		t.Fatalf("bodies after circular pattern = %d, want > 1", def.SurfaceBodies().Count())
	}
}

// Selecting nothing leaves the tool unable to commit.
func TestFeaturePatternNeedsSource(t *testing.T) {
	s, _, _ := extrudedPart(t)
	tool := NewFeatureRectPatternTool()
	s.StartTool(tool)
	if tool.CanCommit() {
		t.Error("pattern with no source selected should not be committable")
	}
}

func TestPatternFeatureCommandsRegistered(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	for _, id := range []string{"Modify.RectangularPattern", "Modify.CircularPattern", "Modify.Mirror"} {
		if _, ok := s.Commands().ByID(id); !ok {
			t.Errorf("command %q not registered", id)
		}
	}
}
