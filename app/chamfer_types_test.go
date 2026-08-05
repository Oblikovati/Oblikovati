// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
)

// #2045: ChamferDefinition carries three setback modes and the wire API routes all three, but the
// interactive tool only ever built the equal-distance one — it had no second distance, no angle
// and no type selector.

// chamferBlockEdge drives the tool over one vertical edge of a 2×2×2 block and returns the
// committed chamfer's definition and the resulting solid volume.
func chamferBlockEdge(t *testing.T, setup func(*ChamferTool)) (*feature.ChamferDefinition, float64) {
	t.Helper()
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})

	ch := NewChamferTool()
	s.StartTool(ch)
	s.Click(50, 50)
	setup(ch)
	if !ch.CanCommit() {
		t.Fatal("chamfer not ready after edge + inputs")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("chamfered body not a valid solid: %+v", r)
	}
	def := ch.AddedFeature().Definition().(*feature.ChamferFeature).Definition()
	return def, ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
}

func TestChamferToolBuildsTwoDistanceChamfer(t *testing.T) {
	def, vol := chamferBlockEdge(t, func(ch *ChamferTool) {
		ch.SetChamferTypeIndex(1) // Two distances
		ch.SetDistance(0.3)
		ch.SetDistance2(0.6)
	})
	if def.Type != types.ChamferTwoDistances {
		t.Errorf("chamfer type = %v, want ChamferTwoDistances", def.Type)
	}
	if d2 := def.Distance2; d2 == nil || d2() != 0.6 {
		t.Error("Distance2 is not the 0.6 the panel set")
	}
	// The wedge is a right triangle 0.3 × 0.6 over the 2-long edge.
	if want := 8 - 0.5*0.3*0.6*2; relErrApp(vol, want) > 1e-6 {
		t.Errorf("two-distance chamfer volume = %g, want %g", vol, want)
	}
}

func TestChamferToolBuildsDistanceAndAngleChamfer(t *testing.T) {
	def, vol := chamferBlockEdge(t, func(ch *ChamferTool) {
		ch.SetChamferTypeIndex(2) // Distance and angle
		ch.SetDistance(0.4)
		ch.SetAngleDegrees(30)
	})
	if def.Type != types.ChamferDistanceAndAngle {
		t.Errorf("chamfer type = %v, want ChamferDistanceAndAngle", def.Type)
	}
	if a := def.Angle; a == nil || stdmath.Abs(a()-stdmath.Pi/6) > 1e-9 {
		t.Error("Angle is not the 30° the panel set")
	}
	// 30° off the first face puts the second setback at d·tan30°.
	if want := 8 - 0.5*0.4*(0.4*stdmath.Tan(stdmath.Pi/6))*2; relErrApp(vol, want) > 1e-6 {
		t.Errorf("distance-angle chamfer volume = %g, want %g", vol, want)
	}
}

func TestChamferToolDefaultsToEqualDistance(t *testing.T) {
	def, vol := chamferBlockEdge(t, func(ch *ChamferTool) { ch.SetDistance(0.5) })
	if def.Type != types.ChamferDistance {
		t.Errorf("chamfer type = %v, want ChamferDistance", def.Type)
	}
	if want := 8 - 0.5*0.5*0.5*2; relErrApp(vol, want) > 1e-6 {
		t.Errorf("equal-distance chamfer volume = %g, want %g", vol, want)
	}
}

// A mode whose second input is unusable must not commit — a zero second distance or a
// degenerate angle builds a chamfer with no face.
func TestChamferToolBlocksCommitOnUnusableModeInput(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	ch := NewChamferTool()
	s.StartTool(ch)
	s.Click(50, 50)
	ch.SetDistance(0.5)

	ch.SetChamferTypeIndex(1)
	ch.SetDistance2(0)
	if ch.CanCommit() {
		t.Error("two-distance chamfer with a zero second distance should not commit")
	}
	ch.SetChamferTypeIndex(2)
	ch.SetAngleDegrees(90)
	if ch.CanCommit() {
		t.Error("distance-and-angle chamfer at 90° should not commit")
	}
	ch.SetAngleDegrees(30)
	if !ch.CanCommit() {
		t.Error("distance-and-angle chamfer at 30° should commit")
	}
}

// Re-editing a committed chamfer seeds the panel with its mode and that mode's input, and
// writing back keeps them.
func TestChamferEditRoundTripsTheMode(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	ch := NewChamferTool()
	s.StartTool(ch)
	s.Click(50, 50)
	ch.SetChamferTypeIndex(2)
	ch.SetDistance(0.4)
	ch.SetAngleDegrees(30)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	pf := ch.AddedFeature()
	edit := editChamferTool(pf, pf.Definition().(*feature.ChamferFeature))
	if edit.ChamferTypeIndex() != 2 {
		t.Errorf("edit seeded type index %d, want 2 (distance and angle)", edit.ChamferTypeIndex())
	}
	if stdmath.Abs(edit.AngleDegrees()-30) > 1e-9 {
		t.Errorf("edit seeded angle %g°, want 30°", edit.AngleDegrees())
	}
	// Switching to two distances mid-edit starts from a committable second setback, not zero.
	edit.SetChamferTypeIndex(1)
	if edit.Distance2() <= 0 {
		t.Errorf("switching modes mid-edit left Distance2 at %g — the panel cannot commit", edit.Distance2())
	}
}
