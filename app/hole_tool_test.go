// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// newPartWithBlock sets up a part whose active body is a side×side×2 block (built by an
// extrude), returning the session and the block body.
func newPartWithBlock(t *testing.T, side float64) (*Session, *topo.Body) {
	t.Helper()
	const height = 2.0
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
	s, block := newPartWithBlock(t, 4) // 4×4×2 block, vol 32
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

// TestHoleToolThroughAll drives the Hole UI with the Through All option: pick the top face,
// set the diameter, tick Through All (no depth needed), OK — and asserts the result has a
// TRUE cylinder wall (one curved face) rather than a faceted prism.
func TestHoleToolThroughAll(t *testing.T) {
	s, block := newPartWithBlock(t, 4) // 4×4×2 block, vol 32
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	hole := NewHoleTool()
	s.StartTool(hole)
	s.Click(100, 100)
	hole.SetDiameter(2)
	hole.SetThroughAll(true) // no depth set on purpose
	if !hole.CanCommit() {
		t.Fatal("through-all hole not ready with face + diameter (depth not required)")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("drilled body not a valid solid: %+v", r)
	}
	cyl := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cyl++
		}
	}
	if cyl != 1 {
		t.Errorf("through-all hole produced %d cylinder faces, want 1 (true wall)", cyl)
	}
}

// TestHoleToolCounterbore drives the Hole UI in counterbore mode: pick the face, set bore +
// recess, tick Through All, OK — and asserts two true cylinder walls (recess + bore).
func TestHoleToolCounterbore(t *testing.T) {
	s, block := newPartWithBlock(t, 8) // 8×8×2 block
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	hole := NewHoleTool()
	s.StartTool(hole)
	s.Click(100, 100)
	hole.SetDiameter(2)
	hole.SetCounterbore(true)
	hole.SetCounterDiameter(4)
	hole.SetCounterDepth(0.5)
	hole.SetThroughAll(true)
	if !hole.CanCommit() {
		t.Fatal("counterbore not ready with face + bore + recess")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("counterbored body not a valid solid: %+v", r)
	}
	cyl := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cyl++
		}
	}
	if cyl != 2 {
		t.Errorf("counterbore produced %d cylinder faces, want 2 (recess + bore)", cyl)
	}
}

// TestHoleToolCountersink drives the Hole UI in countersink mode: pick the face, set bore +
// sink Ø + angle, Through All, OK — and asserts a true cone wall plus a cylinder bore wall.
func TestHoleToolCountersink(t *testing.T) {
	s, block := newPartWithBlock(t, 10) // 10×10×2 block
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	hole := NewHoleTool()
	s.StartTool(hole)
	s.Click(100, 100)
	hole.SetDiameter(1)
	hole.SetCountersink(true)
	hole.SetCounterDiameter(2)
	hole.SetSinkAngle(stdmath.Pi / 2) // 90°
	hole.SetThroughAll(true)
	if !hole.CanCommit() {
		t.Fatal("countersink not ready with face + bore + sink")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("countersunk body not a valid solid: %+v", r)
	}
	cone, cyl := 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cone++
		case geom.Cylinder:
			cyl++
		}
	}
	if cone != 1 || cyl != 1 {
		t.Errorf("countersink produced %d cone / %d cylinder faces, want 1 / 1", cone, cyl)
	}
}

// TestHoleToolConicalPoint drives the Hole UI for a blind drilled hole with the default 118°
// drill point: pick the face, set a depth shallower than the block, OK — and asserts a true
// cone tip plus a cylinder bore.
func TestHoleToolConicalPoint(t *testing.T) {
	s, block := newPartWithBlock(t, 8) // 8×8×2 block
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	hole := NewHoleTool()
	s.StartTool(hole)
	s.Click(100, 100)
	hole.SetDiameter(1)
	hole.SetDepth(1) // blind (< thickness 2); default 118° point
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("conical-point body not a valid solid: %+v", r)
	}
	cone, cyl := 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cone++
		case geom.Cylinder:
			cyl++
		}
	}
	if cone != 1 || cyl != 1 {
		t.Errorf("conical-point hole produced %d cone / %d cylinder faces, want 1 / 1", cone, cyl)
	}
}

func TestHoleViaRibbonCommand(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
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
	s, block := newPartWithBlock(t, 4)
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
