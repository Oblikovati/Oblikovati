// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
)

// The #703 dress-up depth wiring: the interactive tools must reach the M09 model
// capabilities — variable-radius fillet sets, thread class/tapered/model diameter,
// faces-only split, and the offset/thicken approximation request.

// TestFilletToolVariableRadius blends a vertical block edge from 0.3 to 0.8: the commit
// goes through AddFilletSets (one variable set) and the solid stays valid.
func TestFilletToolVariableRadius(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})

	f := NewFilletTool()
	s.StartTool(f)
	s.Click(50, 50)
	f.SetVariable(true)
	f.SetStartRadius(0.3)
	f.SetEndRadius(0.8)
	if !f.CanCommit() {
		t.Fatal("variable fillet not ready after edge + radii")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := f.AddedFeature().Definition().(*feature.FilletFeature).Definition()
	if len(def.EdgeSets) != 1 || def.EdgeSets[0].Radius != nil {
		t.Fatalf("definition has %d sets (variable=%v), want 1 variable set",
			len(def.EdgeSets), len(def.EdgeSets) == 1 && def.EdgeSets[0].Radius == nil)
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("variable-filleted body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; got >= 8 {
		t.Errorf("volume after variable fillet = %g, want < 8 (material removed)", got)
	}
}

// Re-editing a committed variable fillet must seed the panel's variable state.
func TestFilletEditSeedsVariableState(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	f := NewFilletTool()
	s.StartTool(f)
	s.Click(50, 50)
	f.SetVariable(true)
	f.SetStartRadius(0.3)
	f.SetEndRadius(0.8)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	tool, ok := s.editToolFor(f.AddedFeature())
	if !ok {
		t.Fatal("a single-set fillet must re-open in the fillet panel")
	}
	seeded := tool.(*FilletTool)
	if !seeded.Variable() || seeded.StartRadius() != 0.3 || seeded.EndRadius() != 0.8 {
		t.Errorf("seeded variable=%v start=%g end=%g, want true/0.3/0.8",
			seeded.Variable(), seeded.StartRadius(), seeded.EndRadius())
	}
}

// TestSplitToolFacesOnly imprints the mid-plane: the body count and volume stay, the
// crossed faces split (4 sides × 2 + top + bottom = 10 faces).
func TestSplitToolFacesOnly(t *testing.T) {
	t.Parallel()
	s, def, wp := partWithMidPlane(t, 6) // 6×6×2 block, vol 72

	split := NewSplitTool()
	s.StartTool(split)
	split.Pick(s, WorkPlaneHandle{Plane: wp})
	split.SetSplitFaces()
	if split.KeepLabel() != "Split faces (imprint only)" {
		t.Errorf("KeepLabel = %q, want the faces-only label", split.KeepLabel())
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after split faces: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if n := len(body.Faces()); n != 10 {
		t.Errorf("after split faces: %d faces, want 10", n)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; got != 72 {
		t.Errorf("volume after split faces = %g, want exactly 72 (no material removed)", got)
	}
}

// TestThreadToolParityOptions commits class/tapered/model-diameter into the definition,
// and blocks the cut+tapered combination.
func TestThreadToolParityOptions(t *testing.T) {
	t.Parallel()
	s, cyl := newPartWithCylinder(t)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: cylinderFaceOf(t, cyl), Body: cyl}})

	tool := NewThreadTool()
	s.StartTool(tool)
	s.Click(1, 1)
	tool.SetClassIndex(1) // the standard's first external class
	tool.SetTapered(true)
	tool.SetModelDiameterIndex(3) // tap drill
	tool.SetCut(true)
	if tool.CanCommit() {
		t.Fatal("a cut tapered thread must be blocked")
	}
	tool.SetCut(false)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := tool.AddedFeature().Definition().(*feature.ThreadFeature).Definition()
	if def.Class == "" || !def.Tapered || def.ModelDiameter != types.ThreadTapDrillDiameter {
		t.Errorf("definition class=%q tapered=%v modelDiameter=%v, want class set / tapered / tapDrill",
			def.Class, def.Tapered, def.ModelDiameter)
	}
}

// TestFaceOffsetToolApproximation records the #331 request on the committed feature.
func TestFaceOffsetToolApproximation(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 6)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})

	tool := NewFaceOffsetTool()
	s.StartTool(tool)
	s.Click(3, 3)
	tool.SetApproximationIndex(2) // never too thick
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	got := tool.AddedFeature().Definition().(*feature.FaceOffsetFeature).Approximation()
	if got != types.NeverTooThickApproximation {
		t.Errorf("approximation = %v, want neverTooThick", got)
	}
}

// TestThickenToolApproximation records the #331 request on the committed thicken.
func TestThickenToolApproximation(t *testing.T) {
	t.Parallel()
	s, def, region := partWithSquareRegion(t)
	feature.NewBoundaryPatchFeatures(def.Features()).Add(region.Sketch, region.ProfileIndex, feature.PatchFree)
	def.Recompute()

	tool := NewThickenTool()
	s.StartTool(tool)
	tool.SetApproximationIndex(1) // mean
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	got := tool.AddedFeature().Definition().(*feature.ThickenFeature).Approximation()
	if got != types.MeanApproximation {
		t.Errorf("approximation = %v, want mean", got)
	}
}
