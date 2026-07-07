// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// plusZFaceOf returns the body's +Z-facing planar face (the top of the block).
func plusZFaceOf(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.9 {
			return f
		}
	}
	t.Fatal("no +Z face found")
	return nil
}

// plusXFaceOf returns the body's +X-facing planar face (a vertical side of the block).
func plusXFaceOf(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).X > 0.9 {
			return f
		}
	}
	t.Fatal("no +X face found")
	return nil
}

// TestDraftToolEndToEnd drives the Draft UI: start the tool, click a side face of a 2×2×2
// block, set an inward angle, OK — and asserts the side tapered (volume dropped, still a
// valid solid). atan(0.25) ≈ 14.04°, inward ⇒ removes a 4·tan = 1 wedge ⇒ vol 7.
func TestDraftToolEndToEnd(t *testing.T) {
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	s.SetPicker(stubPicker{sel: FaceHandle{Face: plusXFaceOf(t, block), Body: block}})

	d := NewDraftTool()
	s.StartTool(d)
	s.Click(50, 50)
	d.SetAngleDegrees(-stdmath.Atan(0.25) * 180 / stdmath.Pi)
	if !d.CanCommit() {
		t.Fatal("draft not ready after face + angle")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := activePartDef(t, s)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("drafted body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 7) > 1e-6 {
		t.Errorf("draft volume = %g, want 7", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestDraftViaRibbonCommand drives the Draft from its ribbon command.
func TestDraftViaRibbonCommand(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: plusXFaceOf(t, block), Body: block}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.Draft"); err != nil {
		t.Fatalf("execute Modify.Draft: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*DraftTool); !ok {
		t.Fatal("Draft command did not start the draft tool")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := activePartDef(t, s)
	// The default angle is +3° (taper outward), so the volume changes from the 8 baseline.
	if v := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; stdmath.Abs(v-8) < 1e-6 {
		t.Errorf("draft did not change the body: volume %g, want ≠ 8", v)
	}
}

// TestDraftToolPullAndNeutralPicks drives the pull-direction and neutral-plane pick slots: the
// tool routes clicks by pick mode, and the committed draft carries the picked +X face normal as
// its pull direction and the +Z (top) face as its neutral parting plane (#1801).
func TestDraftToolPullAndNeutralPicks(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	sideX := FaceHandle{Face: plusXFaceOf(t, block), Body: block}
	topZ := FaceHandle{Face: plusZFaceOf(t, block), Body: block}

	d := NewDraftTool()
	s.StartTool(d)
	d.Pick(s, sideX) // faces to taper (default mode)

	d.SetPickingPull(true)
	if !d.PickingPull() {
		t.Fatal("pull pick mode not armed")
	}
	d.Pick(s, sideX) // pull-direction face

	d.SetPickingNeutral(true)
	if !d.PickingNeutral() {
		t.Fatal("neutral pick mode not armed")
	}
	d.Pick(s, topZ) // neutral parting plane
	if !d.PullSet() || !d.NeutralSet() {
		t.Fatalf("pull/neutral not registered: pull=%v neutral=%v", d.PullSet(), d.NeutralSet())
	}

	d.SetAngleDegrees(-stdmath.Atan(0.25) * 180 / stdmath.Pi)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := d.AddedFeature().Definition().(*feature.FaceDraftFeature).Definition()
	if def.PullDir.X < 0.9 {
		t.Errorf("pull direction = %+v, want +X normal", def.PullDir)
	}
	if def.Neutral == nil {
		t.Fatal("neutral plane not committed")
	}
	if nz := float64(def.Neutral.Normal().Z); stdmath.Abs(nz) < 0.9 {
		t.Errorf("neutral normal Z = %g, want ±1 (the +Z top face)", nz)
	}
}

// TestDraftToolClearPullNeutral checks the clear handlers drop the pull/neutral inputs and
// return the tool to picking faces.
func TestDraftToolClearPullNeutral(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	side := FaceHandle{Face: plusXFaceOf(t, block), Body: block}
	top := FaceHandle{Face: plusZFaceOf(t, block), Body: block}

	d := NewDraftTool()
	s.StartTool(d)
	d.SetPickingPull(true)
	d.Pick(s, side)
	d.SetPickingNeutral(true)
	d.Pick(s, top)

	d.ClearPull()
	if d.PullSet() || d.PickingPull() {
		t.Error("pull not cleared / mode not reset")
	}
	d.ClearNeutral()
	if d.NeutralSet() || d.PickingNeutral() {
		t.Error("neutral not cleared / mode not reset")
	}
}

// TestDraftToolNeedsFace checks the tool is not committable until a face is picked.
func TestDraftToolNeedsFace(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: plusXFaceOf(t, block), Body: block}})
	d := NewDraftTool()
	s.StartTool(d)
	if d.CanCommit() {
		t.Error("draft ready with no face picked")
	}
	s.Click(0, 0)
	if !d.CanCommit() {
		t.Error("draft not ready after picking a face")
	}
}
