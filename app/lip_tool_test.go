// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
)

// TestLipToolEndToEnd drives the Lip UI: start the tool, click a vertical edge of a 2×2×2
// block, set the bead width/height, OK — and asserts the raised lip added material to a
// still-valid solid.
func TestLipToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	before := query.BodyGeometryProperties(block, ops.DefaultQuality()).Volume
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})

	lt := NewLipTool()
	s.StartTool(lt)
	s.Click(50, 50)
	lt.SetWidth(0.3)
	lt.SetHeight(0.3)
	if !lt.CanCommit() {
		t.Fatal("lip tool not ready after edge + dimensions")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if lt.AddedFeature() == nil || !lt.AddedFeature().Health().OK() {
		t.Fatalf("lip feature not healthy: %+v", lt.AddedFeature())
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if v := ops.Validate(body); !v.Valid || !body.IsSolid() {
		t.Fatalf("lip body not a valid solid: %+v", v)
	}
	if after := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; after <= before {
		t.Errorf("lip volume %g did not rise from %g — a raised lip adds material", after, before)
	}
}

// TestLipToolGrooves confirms the groove mode cuts material (volume drops).
func TestLipToolGrooves(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	before := query.BodyGeometryProperties(block, ops.DefaultQuality()).Volume
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})

	lt := NewLipTool()
	s.StartTool(lt)
	s.Click(50, 50)
	lt.SetWidth(0.3)
	lt.SetHeight(0.3)
	lt.SetGroove(true)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if v := ops.Validate(body); !v.Valid || !body.IsSolid() {
		t.Fatalf("grooved body not a valid solid: %+v", v)
	}
	if after := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; after >= before {
		t.Errorf("groove volume %g did not drop from %g — a groove cuts material", after, before)
	}
}

// TestLipToolParams exercises the property-dialog surface: name, width/height/groove accessors
// (the dimensions reject a non-positive value), and the Params model the head renders.
func TestLipToolParams(t *testing.T) {
	t.Parallel()
	tl := NewLipTool()
	if tl.Name() != "Lip" {
		t.Errorf("name = %q, want Lip", tl.Name())
	}
	if tl.CanCommit() {
		t.Error("lip with no picked edge should not be committable")
	}
	tl.SetWidth(2)
	tl.SetHeight(3)
	tl.SetGroove(true)
	if tl.Width() != 2 || tl.Height() != 3 || !tl.Groove() {
		t.Errorf("width/height/groove = %g/%g/%v, want 2/3/true", tl.Width(), tl.Height(), tl.Groove())
	}
	tl.SetWidth(0) // a non-positive dimension is rejected, keeping the prior value
	if tl.Width() != 2 {
		t.Errorf("width after 0 = %g, want kept at 2", tl.Width())
	}
	p := tl.Params()
	if len(p.Floats) != 2 || len(p.Bools) != 1 {
		t.Fatalf("params = %d floats / %d bools, want 2 / 1", len(p.Floats), len(p.Bools))
	}
	p.Floats[1].Set(4)
	p.Bools[0].Set(false)
	if p.Floats[1].Get() != 4 || p.Bools[0].Get() {
		t.Errorf("param round-trip: height %g groove %v, want 4/false", p.Floats[1].Get(), p.Bools[0].Get())
	}
}

// TestLipToolPreviewAndCancel covers the draft preview (before/after a pick) and Cancel.
func TestLipToolPreviewAndCancel(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	edge := verticalEdgeOf(t, block)
	tl := NewLipTool()
	s.StartTool(tl)

	if _, ok := tl.DraftFeature(s); ok {
		t.Error("no draft should preview before an edge is picked")
	}
	tl.Pick(s, edge)
	tl.Pick(s, edge) // a repeat pick must not duplicate the edge
	if len(tl.Edges()) != 1 {
		t.Fatalf("edges = %d, want 1 (no duplicate)", len(tl.Edges()))
	}
	if _, ok := tl.DraftFeature(s); !ok {
		t.Error("a draft should preview once an edge + dimensions are set")
	}
	tl.Cancel(s)
}

// TestLipToolCommitNoPart covers the no-active-part error path.
func TestLipToolCommitNoPart(t *testing.T) {
	t.Parallel()
	if err := NewLipTool().Commit(NewSession()); err == nil {
		t.Error("commit with no active part should error")
	}
}

// TestLipViaRibbonCommand confirms the Modify-panel ribbon command starts the tool.
func TestLipViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.Lip"); err != nil {
		t.Fatalf("execute Modify.Lip: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*LipTool); !ok {
		t.Fatal("Lip command did not start the lip tool")
	}
}
