// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
)

// TestRuledSurfaceToolEndToEnd drives the Ruled Surface UI: pick a closed region, set the
// distance, OK — and asserts the band exists (one planar quad per profile edge).
func TestRuledSurfaceToolEndToEnd(t *testing.T) {
	s, def, region := partWithSquareRegion(t)
	s.SetPicker(stubPicker{sel: region})

	tool := NewRuledSurfaceTool()
	s.StartTool(tool)
	s.Click(1, 1)
	tool.SetDistance(2)
	if !tool.CanCommit() {
		t.Fatal("ruled tool not ready after picking a profile")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after ruled surface: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	band := def.SurfaceBodies().Item(0)
	if band.IsSolid() || len(band.Faces()) != 4 {
		t.Errorf("band solid=%v faces=%d, want surface with 4 faces", band.IsSolid(), len(band.Faces()))
	}
}

// A tangent ruling resolves its inputs then defers (#339) — the tool must not report that
// Warning as an error.
func TestRuledSurfaceToolTangentDefersAsWarning(t *testing.T) {
	s, _, region := partWithSquareRegion(t)
	s.SetPicker(stubPicker{sel: region})

	tool := NewRuledSurfaceTool()
	s.StartTool(tool)
	s.Click(1, 1)
	tool.SetDirection(int(feature.RuledTangent))
	if err := s.OK(); err != nil {
		t.Fatalf("OK on a deferred mode must not error, got: %v", err)
	}
	if got := tool.AddedFeature().Health().Status; got != health.Warning {
		t.Errorf("tangent ruling health = %v, want Warning (deferred)", got)
	}
}

// TestSurfaceOffsetToolEndToEnd patches a region, then offsets the running surface: the
// patch is replaced by its translated copy (still one one-face sheet body).
func TestSurfaceOffsetToolEndToEnd(t *testing.T) {
	s, def, region := partWithSquareRegion(t)
	feature.NewBoundaryPatchFeatures(def.Features()).Add(region.Sketch, region.ProfileIndex, feature.PatchFree)
	def.Recompute()

	tool := NewSurfaceOffsetTool()
	s.StartTool(tool)
	tool.SetDistance(1.5)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after offset: %d bodies, want 1 (offset replaces the running surface)", def.SurfaceBodies().Count())
	}
	if body := def.SurfaceBodies().Item(0); body.IsSolid() || len(body.Faces()) != 1 {
		t.Errorf("offset body solid=%v faces=%d, want a one-face sheet", body.IsSolid(), len(body.Faces()))
	}
}

// TestMidSurfaceToolEndToEnd extracts the mid-plane of a 6×6×2 plate: only the z face pair
// (separation 2) is under the 3-unit threshold, so one patch of thickness 2 replaces the solid.
func TestMidSurfaceToolEndToEnd(t *testing.T) {
	s, _ := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)

	tool := NewMidSurfaceTool()
	s.StartTool(tool)
	tool.SetMaxThickness(3)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after mid-surface: %d bodies, want 1 patch", def.SurfaceBodies().Count())
	}
	mid := tool.AddedFeature().Definition().(*feature.MidSurfaceFeature)
	if n := mid.Thicknesses().Count(); n != 1 || mid.Thicknesses().Item(0).Value != 2 {
		t.Errorf("thicknesses count=%d, want one pair of thickness 2", n)
	}
}

// TestSurfaceEditToolsViaRibbonCommands asserts each new command starts its tool.
func TestSurfaceEditToolsViaRibbonCommands(t *testing.T) {
	s, _, _ := partWithSquareRegion(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	for id, want := range map[string]string{
		"Surface.Ruled":      "Ruled Surface",
		"Surface.Offset":     "Offset Surface",
		"Surface.MidSurface": "Mid-Surface",
	} {
		if err := s.Execute(id); err != nil {
			t.Fatalf("execute %s: %v", id, err)
		}
		if got := s.ActiveTool().Name(); got != want {
			t.Errorf("%s started tool %q, want %q", id, got, want)
		}
		s.CancelTool()
	}
}

// TestSurfaceEditToolsDraftFeature pins the #1626 commit-gate seam for the three M10 surface
// tools: no draft below each tool's gate, a non-nil draft once commit-ready.
func TestSurfaceEditToolsDraftFeature(t *testing.T) {
	_, _, region := partWithSquareRegion(t)
	ruled := NewRuledSurfaceTool()
	if _, ok := ruled.DraftFeature(nil); ok {
		t.Error("ruled: no draft before a profile is picked")
	}
	ruled.Pick(nil, region)
	if draft, ok := ruled.DraftFeature(nil); !ok || draft == nil {
		t.Error("ruled: want a non-nil draft once a profile is picked")
	}

	offset := NewSurfaceOffsetTool()
	offset.SetDistance(0)
	if _, ok := offset.DraftFeature(nil); ok {
		t.Error("offset: no draft at zero distance")
	}
	offset.SetDistance(1.5)
	if draft, ok := offset.DraftFeature(nil); !ok || draft == nil {
		t.Error("offset: want a non-nil draft at a non-zero distance")
	}

	mid := NewMidSurfaceTool()
	mid.SetMaxThickness(0)
	if _, ok := mid.DraftFeature(nil); ok {
		t.Error("mid: no draft at a non-positive threshold")
	}
	mid.SetMaxThickness(3)
	if draft, ok := mid.DraftFeature(nil); !ok || draft == nil {
		t.Error("mid: want a non-nil draft at a positive threshold")
	}
}
