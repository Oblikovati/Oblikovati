// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/model/feature"
)

// TestFilletToolConcaveStrategy checks the tool defaults to outward fill and carries the chosen
// concave strategy (via the combo index) into the committed fillet's definition.
func TestFilletToolConcaveStrategy(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	f := NewFilletTool()
	s.StartTool(f)
	if f.ConcaveStrategyIndex() != 0 {
		t.Errorf("new fillet tool concave index = %d, want 0 (outward fill default)", f.ConcaveStrategyIndex())
	}
	if len(FilletConcaveOptions()) != 2 {
		t.Errorf("FilletConcaveOptions = %v, want 2 labels", FilletConcaveOptions())
	}
	f.SetConcaveStrategyIndex(1) // inward
	if f.ConcaveStrategyIndex() != 1 {
		t.Errorf("after SetConcaveStrategyIndex(1), index = %d, want 1 (inward)", f.ConcaveStrategyIndex())
	}
	f.SetConcaveStrategyIndex(0) // back to the outward default for the commit below
	s.Click(1, 1)
	f.SetRadius(0.3)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def := f.AddedFeature().Definition().(*feature.FilletFeature).Definition(); def.ConcaveStrategy != types.FilletConcaveOutward {
		t.Errorf("committed fillet concave strategy = %v, want outward (the default)", def.ConcaveStrategy)
	}
}

// TestFilletToolEndToEnd drives the Fillet UI: start the tool, click a vertical edge of a
// 2×2×2 block, set the radius, OK — and asserts a valid solid with a cylinder face and the
// rolling-ball volume. r=0.5, edge length 2: 8 − (r²−πr²/4)·2.
func TestFilletToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	edge := verticalEdgeOf(t, block)
	s.SetPicker(stubPicker{sel: edge})

	f := NewFilletTool()
	s.StartTool(f)
	s.Click(50, 50)
	f.SetRadius(0.5)
	if !f.CanCommit() {
		t.Fatal("fillet not ready after edge + radius")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("filleted body not a valid solid: %+v", r)
	}
	cyls := 0
	for _, fc := range body.Faces() {
		if _, ok := fc.Geometry().(geom.Cylinder); ok {
			cyls++
		}
	}
	if cyls != 1 {
		t.Errorf("filleted body has %d cylinder faces, want 1", cyls)
	}
	want := 8 - (0.5*0.5-stdmath.Pi*0.25*0.25)*2
	if got := query.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-4}).Volume; relErrApp(got, want) > 1e-3 {
		t.Errorf("fillet volume = %g, want ≈ %g", got, want)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestFilletViaRibbonCommand drives the Fillet from its ribbon command/alias.
func TestFilletViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.Fillet"); err != nil {
		t.Fatalf("execute Modify.Fillet: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*FilletTool); !ok {
		t.Fatal("Fillet command did not start the fillet tool")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if v := query.BodyGeometryProperties(activePartDef(t, s).SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; v >= 8 {
		t.Errorf("fillet did not round material: volume %g, want < 8", v)
	}
}

// TestFilletUnbuildableSurfacesNotice checks that committing a fillet the kernel cannot build
// (here, a rolling-ball radius far larger than the 2×2×2 block admits — the ball centre falls
// outside the solid) fails loudly: the tool stays open and the session carries a notice the
// status bar shows — not a silent "nothing happened".
func TestFilletUnbuildableSurfacesNotice(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	f := NewFilletTool()
	s.StartTool(f)
	f.Pick(s, verticalEdgeOf(t, block))
	f.SetRadius(10) // a 10-unit rolling ball cannot round a 2×2×2 block's edge
	if err := s.OK(); err == nil {
		t.Fatal("filleting with an impossible radius should fail")
	}
	if s.ActiveTool() == nil {
		t.Error("a failed commit should keep the fillet tool open")
	}
	if s.Notice() == "" {
		t.Error("a failed commit should set a status-bar notice, not fail silently")
	}
	if v := query.BodyGeometryProperties(activePartDef(t, s).SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; relErrApp(v, 8) > 1e-6 {
		t.Errorf("body should be unchanged (vol 8) after a failed fillet, got %g", v)
	}
}

// TestSickConfigIsNotCommitted is the rule "no mutable operation commits a sick configuration":
// a fillet whose radius overruns the block previews sick, so CommitBlockedReason names the cause,
// s.OK() refuses, and — crucially — NO feature is appended to the design (the sick node must not
// persist in the tree). Fixing the radius then commits cleanly, adding exactly one feature.
func TestSickConfigIsNotCommitted(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	f := NewFilletTool()
	s.StartTool(f)
	f.Pick(s, verticalEdgeOf(t, block))
	f.SetRadius(10) // impossible: the rolling ball overruns the 2×2×2 block

	before := activePartDef(t, s).Features().Count()
	if s.CommitBlockedReason() == "" {
		t.Fatal("a sick fillet config should report a commit-blocked reason")
	}
	if err := s.OK(); err == nil {
		t.Fatal("OK on a sick config should be refused")
	}
	if after := activePartDef(t, s).Features().Count(); after != before {
		t.Errorf("a sick config must not append a feature: count %d → %d", before, after)
	}
	if s.ActiveTool() == nil {
		t.Fatal("tool should stay open after a refused commit")
	}

	f.SetRadius(0.5) // now valid
	if r := s.CommitBlockedReason(); r != "" {
		t.Fatalf("a valid config must not be blocked, got %q", r)
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK on a valid config: %v", err)
	}
	if after := activePartDef(t, s).Features().Count(); after != before+1 {
		t.Errorf("a valid commit should append exactly one feature: count %d → %d", before, after)
	}
}

// TestFilletToolNeedsEdge checks the tool is not committable until an edge is picked.
func TestFilletToolNeedsEdge(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	f := NewFilletTool()
	s.StartTool(f)
	if f.CanCommit() {
		t.Error("fillet ready with no edge picked")
	}
	s.Click(0, 0)
	if !f.CanCommit() {
		t.Error("fillet not ready after picking an edge")
	}
}
