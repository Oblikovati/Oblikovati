// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// TestCoilToolEndToEnd drives the Coil UI: start the tool, click the profile, set pitch
// and revolutions in the property window, OK — and asserts a valid helical solid that
// climbs pitch·revs + the profile height lands in the part.
func TestCoilToolEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~8s): `make test-corpus`")
	}
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 4, 1) // square offset from the Y axis
	s.SetPicker(stubPicker{sel: profile})

	c := NewCoilTool()
	s.StartTool(c)   // ribbon: click "Coil"
	s.Click(120, 90) // viewport: click the profile
	c.SetPitch(2)    // property window
	c.SetRevolutions(3)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after coil, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("coil body not a valid solid: %+v", r)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 0 {
		t.Errorf("coil volume = %g, want > 0", v)
	}
	// Helix climbs along Y by pitch·revs (6) plus the profile height (1).
	if span := body.RangeBox().Max.Y - body.RangeBox().Min.Y; relErrApp(span, 7) > 0.05 {
		t.Errorf("coil axial span = %g, want ≈7", span)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

func TestCoilViaRibbonCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 4, 1)
	s.SetPicker(stubPicker{sel: profile})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Coil"); err != nil { // ribbon: click the Coil button
		t.Fatalf("execute Create.Coil: %v", err)
	}
	c, ok := s.ActiveTool().Tool().(*CoilTool)
	if !ok {
		t.Fatal("Coil command did not start the coil tool")
	}
	s.Click(1, 1)
	c.SetAxis(feature.OriginYAxis)
	c.SetPitch(2)
	c.SetRevolutions(2)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Error("alias-launched coil produced no body")
	}
}

func TestCoilToolNeedsProfileAndRevolutions(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 4, 1)
	s.SetPicker(stubPicker{sel: profile})
	c := NewCoilTool()
	s.StartTool(c)
	c.SetRevolutions(0) // invalid
	if c.CanCommit() {
		t.Error("coil ready with no profile / zero revolutions")
	}
	s.Click(0, 0)
	c.SetRevolutions(3)
	if !c.CanCommit() {
		t.Error("coil not ready after profile + revolutions")
	}
}
