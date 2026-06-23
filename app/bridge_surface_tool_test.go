// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// partWithTwoPanels seeds a part with two curved NURBS surface panels to bridge (x∈[0,1], x∈[2,3]).
func partWithTwoPanels(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "bridge.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 0, func(i, j int) float64 { return 0.4 * float64(i*i) })})
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 2, func(i, j int) float64 { return 0.3 * float64((4-i)*(4-i)) })})
	def.Recompute()
	return s, def
}

func TestBridgeSurfaceToolConnectsPanels(t *testing.T) {
	s, def := partWithTwoPanels(t)
	tool := NewBridgeSurfaceTool() // G2 both sides
	s.StartTool(tool)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := def.SurfaceBodies().Count(); got != 3 {
		t.Fatalf("after bridge = %d surface bodies, want 3 (2 panels + bridge)", got)
	}
	bs, ok := nurbsFaceSurface(def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1))
	if !ok {
		t.Fatal("bridge body has no NURBS face")
	}
	if x := float64(bs.PointAt(0.5, 0.5).X); x < 1.2 || x > 1.8 {
		t.Errorf("bridge mid x = %g, want between the panels (~1.5)", x)
	}
}

func TestBridgeSurfaceToolParams(t *testing.T) {
	tool := NewBridgeSurfaceTool()
	if tool.Name() != "Bridge Surface" || tool.Prompt(nil) == "" || !tool.CanCommit() {
		t.Error("bridge tool should name, prompt and be committable by default")
	}
	p := tool.Params()
	if len(p.Choices) != 2 {
		t.Fatalf("bridge tool should expose 2 continuity choices, got %d", len(p.Choices))
	}
	p.Choices[0].Set(0)
	p.Choices[1].Set(1)
	if p.Choices[0].Get() != 0 || p.Choices[1].Get() != 1 {
		t.Error("continuity get/set round-trip mismatch")
	}
	tool.SetContinuity(-1, 0)
	if tool.CanCommit() {
		t.Error("out-of-range continuity should block commit")
	}
}

func TestBridgeSurfaceViaRibbonCommand(t *testing.T) {
	s, _ := partWithTwoPanels(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Bridge"); err != nil {
		t.Fatalf("execute Surface.Bridge: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Bridge Surface" {
		t.Errorf("Surface.Bridge started tool %q, want Bridge Surface", got)
	}
}
