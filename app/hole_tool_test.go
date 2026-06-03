// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// newPartWithBlock sets up a part whose active body is a side×side×height block (built
// by an extrude), returning the session and the block body.
func newPartWithBlock(t *testing.T, side, height float64) (*Session, *topo.Body) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(side, 0))
	c2 := sk.Points().Add(math.P2(side, side))
	c3 := sk.Points().Add(math.P2(0, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return height })
	def.Recompute()
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("block setup: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	return s, def.SurfaceBodies().Item(0)
}

// topFaceOf returns the body's +Z-facing planar face (the block's top).
func topFaceOf(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.9 {
			return f
		}
	}
	t.Fatal("no top face found")
	return nil
}

// TestHoleToolEndToEnd drives the Hole UI: start the tool, click the block's top face,
// set diameter and depth, OK — and asserts a through hole removed the right volume.
func TestHoleToolEndToEnd(t *testing.T) {
	s, block := newPartWithBlock(t, 4, 2) // 4×4×2 block, vol 32
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	hole := NewHoleTool()
	s.StartTool(hole) // ribbon: click "Hole"
	s.Click(100, 100) // viewport: click the top face
	hole.SetDiameter(2)
	hole.SetDepth(3) // > thickness 2 ⇒ through
	if !hole.CanCommit() {
		t.Fatal("hole tool not ready after face + diameter + depth")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after hole, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("drilled body not a valid solid: %+v", r)
	}
	// Removed = a Ø2 cylinder (32-gon faceted) over the full thickness 2 ⇒ block − that.
	ngonArea := 0.5 * 32 * stdmath.Sin(2*stdmath.Pi/32) // r=1
	want := 32 - ngonArea*2
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 0.01 {
		t.Errorf("drilled volume = %g, want ≈%g", got, want)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

func TestHoleViaRibbonCommand(t *testing.T) {
	s, block := newPartWithBlock(t, 4, 2)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.PressKey(KeyEvent{Key: "H"}); err != nil { // "H" alias
		t.Fatalf("alias: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*HoleTool); !ok {
		t.Fatal("Hole alias did not start the hole tool")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil { // default Ø1 × 2 blind hole
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if v := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; v >= 32 {
		t.Errorf("hole did not remove material: volume %g, want < 32", v)
	}
}

func TestHoleToolNeedsFace(t *testing.T) {
	s, block := newPartWithBlock(t, 4, 2)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})
	hole := NewHoleTool()
	s.StartTool(hole)
	if hole.CanCommit() {
		t.Error("hole ready with no face picked")
	}
	s.Click(0, 0)
	if !hole.CanCommit() {
		t.Error("hole not ready after picking a face")
	}
}
