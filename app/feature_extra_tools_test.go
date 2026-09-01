// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// blockVolume returns the part's first body volume (analytic comparisons need it).
func blockVolume(def *compdef.PartComponentDefinition) float64 {
	b := def.SurfaceBodies().Item(0)
	return float64(ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume)
}

// TestBossToolEndToEnd raises a Ø2 × 1.5 stud on the block top: the volume grows by the
// faceted stud's prism (the boss tool drives the same drillTool facets as the hole).
func TestBossToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 6) // 6×6×2 block, volume 72
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})

	tool := NewBossTool()
	s.StartTool(tool)
	s.Click(3, 3)
	tool.diameter, tool.height = 2, 1.5
	if !tool.CanCommit() {
		t.Fatal("boss tool not ready after picking a face")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := blockVolume(def); got <= 72 {
		t.Errorf("volume after boss = %g, want > 72 (stud joined on top)", got)
	}
}

// TestHullToolEndToEnd hulls an L-shaped part (block + stud) into its convex hull: the
// hull volume strictly exceeds the input volume (concavity filled).
func TestHullToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	feature.NewBossFeatures(def.Features()).Add(topFaceOf(t, block).ReferenceKey(),
		func() float64 { return 2 }, func() float64 { return 3 })
	def.Recompute()
	before := blockVolume(def)

	s.StartTool(NewHullTool())
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after hull: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	if got := blockVolume(def); got <= before {
		t.Errorf("hull volume = %g, want > %g (concavity filled)", got, before)
	}
}

// TestDirectEditToolSizeOverFaces pushes the block top up by 1 via the Size operation:
// 6×6×2 grows to 6×6×3 (volume 72 → 108).
func TestDirectEditToolSizeOverFaces(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})

	tool := NewDirectEditTool()
	s.StartTool(tool)
	s.Click(3, 3)
	tool.SetOperation(1) // Size
	tool.vec, tool.distance = [3]float64{0, 0, 1}, 1
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := blockVolume(def); stdmath.Abs(got-108) > 1e-9 {
		t.Errorf("volume after size = %g, want 108", got)
	}
}

// TestDirectEditToolScaleNeedsNoFaces scales the whole block ×1.5 about the origin:
// volume 72 → 72·1.5³ = 243.
func TestDirectEditToolScaleNeedsNoFaces(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)

	tool := NewDirectEditTool()
	s.StartTool(tool)
	tool.SetOperation(4) // Scale
	tool.scale = 1.5
	if !tool.CanCommit() {
		t.Fatal("scale needs no face picks")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := blockVolume(def); stdmath.Abs(got-243) > 1e-9 {
		t.Errorf("volume after scale = %g, want 243", got)
	}
}

// TestSketchDrivenPatternToolEndToEnd patterns a boss at three sketch points: each extra
// point adds one stud's volume.
func TestSketchDrivenPatternToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	boss := feature.NewBossFeatures(def.Features()).Add(topFaceOf(t, block).ReferenceKey(),
		func() float64 { return 1 }, func() float64 { return 1 })
	def.Recompute()
	before := blockVolume(def)
	driver := def.Sketches().Add(sketch.XYPlane())
	driver.Points().Add(math.P2(1.5, 1.5)) // the source location
	driver.Points().Add(math.P2(4.5, 1.5))
	driver.Points().Add(math.P2(1.5, 4.5))

	tool := NewFeatureSketchDrivenPatternTool()
	s.StartTool(tool)
	tool.Pick(s, FeatureHandle{Feature: boss})
	tool.Pick(s, SketchHandle{Sketch: driver})
	if !tool.CanCommit() {
		t.Fatal("pattern tool not ready after feature + sketch picks")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := blockVolume(def); got <= before {
		t.Errorf("volume after sketch-driven pattern = %g, want > %g (copies placed)", got, before)
	}
}

// TestExtraToolsDraftFeature asserts Boss, Direct Edit, Hull and the Sketch-Driven
// Pattern build the draft the commit gate inspects (#1626): no draft before the tool is
// commit-ready, a non-nil draft once it is.
func TestExtraToolsDraftFeature(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	top := FaceHandle{Face: topFaceOf(t, block), Body: block}

	boss := NewBossTool()
	if _, ok := boss.DraftFeature(s); ok {
		t.Error("boss: draft ready with no face picked")
	}
	boss.Pick(s, top)
	if draft, ok := boss.DraftFeature(s); !ok || draft == nil {
		t.Errorf("boss: no draft once commit-ready (ok=%v)", ok)
	}

	edit := NewDirectEditTool()
	if _, ok := edit.DraftFeature(s); ok {
		t.Error("direct edit: draft ready with no face picked")
	}
	edit.Pick(s, top)
	if draft, ok := edit.DraftFeature(s); !ok || draft == nil {
		t.Errorf("direct edit: no draft once commit-ready (ok=%v)", ok)
	}

	if draft, ok := NewHullTool().DraftFeature(s); !ok || draft == nil {
		t.Errorf("hull: no draft (input-free, always ready; ok=%v)", ok)
	}

	pattern := NewFeatureSketchDrivenPatternTool()
	pattern.Pick(s, FeatureHandle{Feature: def.Features().Item(0)})
	if _, ok := pattern.DraftFeature(s); ok {
		t.Error("sketch-driven pattern: draft ready with no driving sketch")
	}
	driver := def.Sketches().Add(sketch.XYPlane())
	driver.Points().Add(math.P2(1.5, 1.5))
	pattern.Pick(s, SketchHandle{Sketch: driver})
	if draft, ok := pattern.DraftFeature(s); !ok || draft == nil {
		t.Errorf("sketch-driven pattern: no draft once commit-ready (ok=%v)", ok)
	}
}

// TestFeatureExtraToolsViaRibbonCommands asserts each new command starts its tool.
func TestFeatureExtraToolsViaRibbonCommands(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	for id, want := range map[string]string{
		"Modify.Boss":                "Boss",
		"Modify.DirectEdit":          "Direct Edit",
		"Modify.Hull":                "Hull",
		"Modify.SketchDrivenPattern": "Sketch-Driven Pattern",
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
