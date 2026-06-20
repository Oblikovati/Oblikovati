// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestRestToolEndToEnd drives the Rest UI: pick a region, set a depth, OK — and asserts the
// raised pad added material (block 72 + 2×2×1 = 76).
func TestRestToolEndToEnd(t *testing.T) {
	s, def, region := partWithTopRegion(t)
	s.SetPicker(stubPicker{sel: region})

	r := NewRestTool()
	s.StartTool(r)
	s.Click(100, 100)
	r.SetDepth(1)
	if !r.CanCommit() {
		t.Fatal("rest tool not ready after picking a region + depth")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if r.AddedFeature() == nil || !r.AddedFeature().Health().OK() {
		t.Fatalf("rest feature not healthy: %+v", r.AddedFeature())
	}
	body := def.SurfaceBodies().Item(0)
	if v := ops.Validate(body); !v.Valid || !body.IsSolid() {
		t.Fatalf("rest body not a valid solid: %+v", v)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(v, 76) > 0.01 {
		t.Errorf("rest volume = %g, want ≈76 (72 + 2×2×1)", v)
	}
}

// TestRestToolRecesses confirms the recessed mode cuts a pocket (block 72 − 2×2×1 = 68).
func TestRestToolRecesses(t *testing.T) {
	s, def, region := partWithTopRegion(t)
	s.SetPicker(stubPicker{sel: region})

	r := NewRestTool()
	s.StartTool(r)
	s.Click(100, 100)
	r.SetDepth(1)
	r.SetRecessed(true)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if v := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; relErrApp(v, 68) > 0.01 {
		t.Errorf("recessed volume = %g, want ≈68 (72 − 2×2×1)", v)
	}
}

// TestRestToolParams exercises the property-dialog surface: name, depth/recessed accessors,
// and the Params model the head renders.
func TestRestToolParams(t *testing.T) {
	tl := NewRestTool()
	if tl.Name() != "Rest" {
		t.Errorf("name = %q, want Rest", tl.Name())
	}
	if tl.CanCommit() {
		t.Error("rest with no picked region should not be committable")
	}
	tl.SetDepth(3)
	tl.SetRecessed(true)
	if tl.Depth() != 3 || !tl.Recessed() {
		t.Errorf("depth/recessed = %g/%v, want 3/true", tl.Depth(), tl.Recessed())
	}
	p := tl.Params()
	if len(p.Floats) != 1 || len(p.Bools) != 1 {
		t.Fatalf("params = %d floats / %d bools, want 1 / 1", len(p.Floats), len(p.Bools))
	}
	p.Floats[0].Set(5)
	p.Bools[0].Set(false)
	if p.Floats[0].Get() != 5 || p.Bools[0].Get() {
		t.Errorf("param round-trip: depth %g recessed %v, want 5/false", p.Floats[0].Get(), p.Bools[0].Get())
	}
}

// TestRestToolPreviewAndPick covers the draft preview, Ctrl-toggle multi-pick, and Cancel.
func TestRestToolPreviewAndPick(t *testing.T) {
	s, _, region := partWithTopRegion(t)
	tl := NewRestTool()
	s.StartTool(tl)

	if _, ok := tl.DraftFeature(s); ok {
		t.Error("no draft should preview before a region is picked")
	}
	tl.PickWithMods(s, region, CtrlMod) // first Ctrl-pick adds it
	if len(tl.PickedProfiles()) != 1 {
		t.Fatalf("picked = %d, want 1 after Ctrl-pick", len(tl.PickedProfiles()))
	}
	tl.SetDepth(1)
	if _, ok := tl.DraftFeature(s); !ok {
		t.Error("a draft should preview once a region + depth are set")
	}
	tl.PickWithMods(s, region, CtrlMod) // Ctrl-pick again toggles it off
	if len(tl.PickedProfiles()) != 0 {
		t.Errorf("picked = %d, want 0 after toggling the region off", len(tl.PickedProfiles()))
	}
	tl.Cancel(s)
}

// TestRestToolCommitNoPart covers the no-active-part error path.
func TestRestToolCommitNoPart(t *testing.T) {
	if err := NewRestTool().Commit(NewSession()); err == nil {
		t.Error("commit with no active part should error")
	}
}

// TestRestViaRibbonCommand confirms the Create-panel ribbon command starts the tool.
func TestRestViaRibbonCommand(t *testing.T) {
	s, _, _ := partWithTopRegion(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Rest"); err != nil {
		t.Fatalf("execute Create.Rest: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*RestTool); !ok {
		t.Fatal("Rest command did not start the rest tool")
	}
}
