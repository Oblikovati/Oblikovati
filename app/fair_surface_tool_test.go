// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// partWithOneSurface seeds a session's active part with one curved NURBS surface body.
func partWithOneSurface(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "fair.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 0, func(i, j int) float64 { return 0.2 * float64(i) })})
	def.Recompute()
	return s, def
}

func TestFairSurfaceToolParams(t *testing.T) {
	t.Parallel()
	tool := NewFairSurfaceTool()
	if tool.Name() != "Fair Surface" || !tool.CanCommit() {
		t.Error("fair tool should name and be committable by default")
	}
	p := tool.Params()
	p.Floats[0].Set(0.8)
	p.Ints[0].Set(30)
	p.Choices[0].Set(1)
	if p.Floats[0].Get() != 0.8 || p.Ints[0].Get() != 30 || p.Choices[0].Get() != 1 {
		t.Error("fair param get/set round-trip mismatch")
	}
	tool.strength = 0
	if tool.CanCommit() {
		t.Error("zero strength should block commit")
	}
}

func TestFairSurfaceToolCommits(t *testing.T) {
	t.Parallel()
	s, _ := partWithOneSurface(t)
	tool := NewFairSurfaceTool()
	tool.hold = 1
	s.StartTool(tool)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if tool.AddedFeature() == nil {
		t.Error("fair commit should add a feature")
	}
}

func TestFairViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, _ := partWithOneSurface(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Fair"); err != nil {
		t.Fatalf("execute Surface.Fair: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Fair Surface" {
		t.Errorf("Surface.Fair started tool %q, want Fair Surface", got)
	}
}

// TestFairSurfaceToolDraftFeature pins the #1626 commit-gate seam: no draft at zero strength,
// a non-nil draft once commit-ready.
func TestFairSurfaceToolDraftFeature(t *testing.T) {
	t.Parallel()
	tool := NewFairSurfaceTool()
	tool.strength = 0
	if _, ok := tool.DraftFeature(nil); ok {
		t.Error("DraftFeature must not build at a non-positive strength")
	}
	tool.strength = 0.5
	if draft, ok := tool.DraftFeature(nil); !ok || draft == nil {
		t.Fatalf("DraftFeature = (%v, %v), want a non-nil draft once commit-ready", draft, ok)
	}
}
