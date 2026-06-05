// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// partWithMidPlane builds a side×side×2 block and a work plane at z=1 cutting through it.
func partWithMidPlane(t *testing.T, side float64) (*Session, *compdef.PartComponentDefinition, *feature.WorkPlane) {
	t.Helper()
	s, _ := newPartWithBlock(t, side) // block on XY, z 0..2
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 1 })
	def.Recompute()
	return s, def, wp
}

// TestSplitToolEndToEnd drives the Split UI: pick a cutting work plane, OK — and asserts the
// part divided into two valid solids.
func TestSplitToolEndToEnd(t *testing.T) {
	s, def, wp := partWithMidPlane(t, 6)

	split := NewSplitTool()
	s.StartTool(split)
	split.Pick(s, WorkPlaneHandle{Plane: wp})
	if !split.CanCommit() {
		t.Fatal("split tool not ready after picking a plane")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("after split: %d bodies, want 2", def.SurfaceBodies().Count())
	}
	for i := 0; i < def.SurfaceBodies().Count(); i++ {
		b := def.SurfaceBodies().Item(i)
		if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
			t.Fatalf("split piece %d not a valid solid: %+v", i, r)
		}
	}
}

// Trim mode (keep one side) leaves a single body.
func TestSplitToolTrimsOneSide(t *testing.T) {
	s, def, wp := partWithMidPlane(t, 6)

	split := NewSplitTool()
	s.StartTool(split)
	split.Pick(s, WorkPlaneHandle{Plane: wp})
	split.SetKeepNegative()
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Errorf("after trim: %d bodies, want 1", def.SurfaceBodies().Count())
	}
}

func TestSplitNeedsPlane(t *testing.T) {
	s, _, _ := partWithMidPlane(t, 4)
	split := NewSplitTool()
	s.StartTool(split)
	if split.CanCommit() {
		t.Error("split committable with no plane picked")
	}
}

func TestSplitViaRibbonCommand(t *testing.T) {
	s, _, _ := partWithMidPlane(t, 4)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.Split"); err != nil {
		t.Fatalf("execute Modify.Split: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*SplitTool); !ok {
		t.Fatal("Split command did not start the split tool")
	}
}
