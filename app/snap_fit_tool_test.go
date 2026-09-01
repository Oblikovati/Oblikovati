// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
)

// TestSnapFitToolEndToEnd drives the Snap Fit UI: keep the default beam/catch dimensions, OK —
// and asserts a valid solid hook was added to the empty part.
func TestSnapFitToolEndToEnd(t *testing.T) {
	t.Parallel()
	s := newPartSession(t)
	def := activePartDef(t, s)

	sf := NewSnapFitTool()
	s.StartTool(sf)
	if !sf.CanCommit() {
		t.Fatal("snap-fit tool not ready with its default dimensions")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if sf.AddedFeature() == nil || !sf.AddedFeature().Health().OK() {
		t.Fatalf("snap-fit feature not healthy: %+v", sf.AddedFeature())
	}
	body := def.SurfaceBodies().Item(0)
	if v := ops.Validate(body); !v.Valid || !body.IsSolid() {
		t.Fatalf("snap-fit body not a valid solid: %+v", v)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 0 {
		t.Errorf("snap-fit volume = %g, want a positive solid", v)
	}
}

// TestSnapFitToolParams exercises the property-dialog surface: name, the five dimension
// accessors (each rejects a non-positive value), and the Params model the head renders.
func TestSnapFitToolParams(t *testing.T) {
	t.Parallel()
	tl := NewSnapFitTool()
	if tl.Name() != "Snap Fit" {
		t.Errorf("name = %q, want Snap Fit", tl.Name())
	}
	tl.SetLength(8)
	tl.SetWidth(3)
	tl.SetThickness(1.5)
	tl.SetCatchLength(2)
	tl.SetCatchHeight(2)
	if tl.Length() != 8 || tl.Width() != 3 || tl.Thickness() != 1.5 || tl.CatchLength() != 2 || tl.CatchHeight() != 2 {
		t.Errorf("dims = %g/%g/%g/%g/%g, want 8/3/1.5/2/2",
			tl.Length(), tl.Width(), tl.Thickness(), tl.CatchLength(), tl.CatchHeight())
	}
	tl.SetLength(0) // non-positive entries are rejected, keeping the prior value
	tl.SetCatchHeight(-1)
	if tl.Length() != 8 || tl.CatchHeight() != 2 {
		t.Errorf("after bad entries length/catchHeight = %g/%g, want kept at 8/2", tl.Length(), tl.CatchHeight())
	}

	p := tl.Params()
	if len(p.Floats) != 5 {
		t.Fatalf("params = %d floats, want 5", len(p.Floats))
	}
	p.Floats[0].Set(10)
	if p.Floats[0].Get() != 10 {
		t.Errorf("param round-trip: beam length %g, want 10", p.Floats[0].Get())
	}
}

// TestSnapFitToolPreview covers the draft preview before and after the dimensions are valid.
func TestSnapFitToolPreview(t *testing.T) {
	t.Parallel()
	s := newPartSession(t)
	tl := NewSnapFitTool()
	s.StartTool(tl)
	if _, ok := tl.DraftFeature(s); !ok {
		t.Error("default dimensions are positive, so a draft should preview")
	}
	tl.SetThickness(0) // rejected, kept positive — still previewable
	if _, ok := tl.DraftFeature(s); !ok {
		t.Error("a non-positive entry is rejected, so the draft should still preview")
	}
}

// TestSnapFitToolCommitNoPart covers the no-active-part error path.
func TestSnapFitToolCommitNoPart(t *testing.T) {
	t.Parallel()
	if err := NewSnapFitTool().Commit(NewSession()); err == nil {
		t.Error("commit with no active part should error")
	}
}

// TestSnapFitViaRibbonCommand confirms the Create-panel ribbon command starts the tool.
func TestSnapFitViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s := newPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.SnapFit"); err != nil {
		t.Fatalf("execute Create.SnapFit: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*SnapFitTool); !ok {
		t.Fatal("Snap Fit command did not start the snap-fit tool")
	}
}
