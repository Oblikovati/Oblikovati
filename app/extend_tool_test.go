// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/feature"
	"oblikovati/model/sketch"
)

// patchWithBottomEdge gives a part holding a 4×4 surface patch and its bottom boundary edge.
func patchWithBottomEdge(t *testing.T) (*Session, *compdef.PartComponentDefinition, *topo.Edge) {
	t.Helper()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(4, 0))
	c2 := sk.Points().Add(math.P2(4, 4))
	c3 := sk.Points().Add(math.P2(0, 4))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewBoundaryPatchFeatures(def.Features()).Add(sk, 0, feature.PatchFree)
	def.Recompute()
	var bottom *topo.Edge
	for _, e := range def.SurfaceBodies().Item(0).Edges() {
		if e.StartVertex().Point().Y == 0 && e.EndVertex().Point().Y == 0 {
			bottom = e
		}
	}
	if bottom == nil {
		t.Fatal("no bottom edge on the patch")
	}
	return s, def, bottom
}

// TestExtendToolEndToEnd drives the Extend UI: pick the bottom edge, set a distance, OK — the
// patch grows to y∈[-2,4].
func TestExtendToolEndToEnd(t *testing.T) {
	s, def, bottom := patchWithBottomEdge(t)
	s.SetPicker(stubPicker{sel: EdgeHandle{Edge: bottom}})

	ext := NewExtendTool()
	s.StartTool(ext)
	s.Click(100, 100)
	ext.SetDistance(2)
	if !ext.CanCommit() {
		t.Fatal("extend tool not ready after edge + distance")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	box := def.SurfaceBodies().Item(0).RangeBox()
	if stdmath.Abs(float64(box.Min.Y)+2) > 1e-6 || stdmath.Abs(float64(box.Max.Y)-4) > 1e-6 {
		t.Errorf("extended y-span = [%v,%v], want [-2,4]", box.Min.Y, box.Max.Y)
	}
}

func TestExtendViaRibbonCommand(t *testing.T) {
	s, _, bottom := patchWithBottomEdge(t)
	s.SetPicker(stubPicker{sel: EdgeHandle{Edge: bottom}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Extend"); err != nil {
		t.Fatalf("execute Surface.Extend: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*ExtendTool); !ok {
		t.Fatal("Extend command did not start the extend tool")
	}
}
