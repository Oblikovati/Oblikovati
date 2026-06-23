// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// partWithOneCurvedSurface seeds a part with a single curved NURBS surface body.
func partWithOneCurvedSurface(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "ext.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 0, func(i, j int) float64 { return 0.5 * float64(i*i) })})
	def.Recompute()
	return s, def
}

func TestExtendSurfaceToolG2ContinuesCurvature(t *testing.T) {
	s, def := partWithOneCurvedSurface(t)
	tool := NewExtendSurfaceTool() // defaults: U-max, curvature (G2), distance 1
	s.StartTool(tool)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	bs, ok := nurbsFaceSurface(def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1))
	if !ok {
		t.Fatal("extended body has no NURBS face")
	}
	if _, hi := bs.UDomain(); hi <= 1+1e-9 {
		t.Errorf("extended u-domain max = %g, want > 1", hi)
	}
	// Curvature continuous across the original boundary u=1 (the F13 continuity measure).
	below := bs.SurfaceDersAt(1-1e-6, 0.5, 2, 0)
	above := bs.SurfaceDersAt(1+1e-6, 0.5, 2, 0)
	if !below[1][0].IsEqualTo(above[1][0], 1e-3) {
		t.Errorf("extend G2 tangent jump at join: %v vs %v", below[1][0], above[1][0])
	}
	if !below[2][0].IsEqualTo(above[2][0], 1e-3) {
		t.Errorf("extend G2 curvature jump at join: %v vs %v", below[2][0], above[2][0])
	}
}

func TestExtendSurfaceToolParams(t *testing.T) {
	tool := NewExtendSurfaceTool()
	if tool.Prompt(nil) == "" || !tool.CanCommit() {
		t.Error("extend tool should prompt and (with default distance) be committable")
	}
	p := tool.Params()
	p.Floats[0].Set(2.5)
	p.Choices[0].Set(2)
	p.Choices[1].Set(0)
	if p.Floats[0].Get() != 2.5 || p.Choices[0].Get() != 2 || p.Choices[1].Get() != 0 {
		t.Error("param get/set round-trip mismatch")
	}
	tool.distance = 0
	if tool.CanCommit() {
		t.Error("zero distance should block commit")
	}
}

func TestUntrimCommandRecoversFace(t *testing.T) {
	s, def := partWithOneCurvedSurface(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Untrim"); err != nil {
		t.Fatalf("execute Surface.Untrim: %v", err)
	}
	body := def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1)
	if len(body.Faces()) != 1 || len(body.Edges()) != 4 {
		t.Errorf("untrimmed body = %d faces, %d edges; want 1 face, 4 edges", len(body.Faces()), len(body.Edges()))
	}
	if _, ok := body.Faces()[0].Geometry().(geom.BSplineSurface); !ok {
		t.Error("untrimmed face should keep its NURBS surface")
	}
}

func TestExtendNurbsViaRibbonCommand(t *testing.T) {
	s, _ := partWithOneCurvedSurface(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.ExtendNurbs"); err != nil {
		t.Fatalf("execute Surface.ExtendNurbs: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Extend Surface" {
		t.Errorf("Surface.ExtendNurbs started tool %q, want Extend Surface", got)
	}
}
