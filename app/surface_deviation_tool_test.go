// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// partWithTwoOffsetSurfaces seeds a part with two coincident NURBS panels, the second lifted +dz, so
// their deviation is a known constant.
func partWithTwoOffsetSurfaces(t *testing.T, dz float64) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "dev.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 0, func(i, j int) float64 { return 0 })})
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 0, func(i, j int) float64 { return dz })})
	def.Recompute()
	return s, def
}

func TestSurfaceDeviationToolReportsOffset(t *testing.T) {
	const dz = 0.3
	s, _ := partWithTwoOffsetSurfaces(t, dz)
	tool := NewSurfaceDeviationTool()
	s.StartTool(tool) // Start computes the deviation
	r := tool.Report()
	if r == nil {
		t.Fatal("deviation tool produced no report")
	}
	if stdmath.Abs(r.AbsMax-dz) > 1e-6 {
		t.Errorf("deviation AbsMax = %g, want ≈ %g (the offset)", r.AbsMax, dz)
	}
	if len(s.DeviationItems()) == 0 {
		t.Error("deviation tool should produce a colour overlay")
	}
}

func TestSurfaceDeviationToolParams(t *testing.T) {
	tool := NewSurfaceDeviationTool()
	if tool.Name() != "Surface Deviation" || tool.CanCommit() {
		t.Error("deviation is a display tool (no commit)")
	}
	p := tool.Params()
	p.Floats[0].Set(0.05)
	if p.Floats[0].Get() != 0.05 {
		t.Error("tolerance get/set round-trip mismatch")
	}
}

func TestSurfaceDeviationViaRibbonCommand(t *testing.T) {
	s, _ := partWithTwoOffsetSurfaces(t, 0.2)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Inspect.SurfaceDeviation"); err != nil {
		t.Fatalf("execute Inspect.SurfaceDeviation: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Surface Deviation" {
		t.Errorf("Inspect.SurfaceDeviation started tool %q, want Surface Deviation", got)
	}
}
