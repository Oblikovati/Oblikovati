// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/kernel/brep"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/doc"
	"oblikovati/model/feature"
)

// newPartWithCylinder sets up a part whose active body is an analytic solid cylinder (which
// carries a real geom.Cylinder face to thread).
func newPartWithCylinder(t *testing.T) (*Session, *topo.Body) {
	t.Helper()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5, 2.0)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	feature.NewBaseFeatures(def.Features()).AddBase(cyl)
	def.Recompute()
	return s, def.SurfaceBodies().Item(0)
}

// cylinderFaceOf returns the body's first cylindrical face.
func cylinderFaceOf(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("no cylindrical face found")
	return nil
}

// TestThreadToolEndToEnd drives the Thread UI the ribbon way: start the Thread tool, click a
// cylindrical face, choose M8 / 1.25 / cut, OK — and asserts a real modeled thread reduced the
// volume and the tool closed.
func TestThreadToolEndToEnd(t *testing.T) {
	s, cyl := newPartWithCylinder(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	before := ops.BodyGeometryProperties(cyl, ops.DefaultQuality()).Volume
	face := cylinderFaceOf(t, cyl)

	// Ribbon: click "Thread" → the tool; pick the cylindrical face; configure; OK.
	s.SetPicker(stubPicker{sel: FaceHandle{Face: face, Body: cyl}})
	tool := NewThreadTool()
	s.StartTool(tool)
	s.Click(100, 100)
	if !tool.HasFace() {
		t.Fatal("thread tool did not capture the picked cylindrical face")
	}
	tool.SetStandardIndex(0) // ISO
	// pick M8 (index 6 in the metric table) and its coarse pitch (1.25), then a cut thread.
	tool.SetSizeIndex(6)
	tool.SetPitchIndex(0)
	tool.SetCut(true)
	if d, err := tool.Designation(); err != nil || d != "M8x1.25" {
		t.Fatalf("designation = %q, %v; want M8x1.25", d, err)
	}
	if !tool.CanCommit() {
		t.Fatal("thread tool not ready after face + pick")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("thread OK: %v", err)
	}

	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("threaded body not a valid solid: %+v", r)
	}
	after := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if after >= before {
		t.Errorf("modeled cut thread should remove material: %.4f → %.4f", before, after)
	}
	if s.ActiveTool() != nil {
		t.Error("thread tool should close after OK")
	}
}

// TestThreadToolRejectsPlanarFace checks the tool ignores a non-cylindrical pick.
func TestThreadToolRejectsPlanarFace(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})
	tool := NewThreadTool()
	s.StartTool(tool)
	s.Click(100, 100)
	if tool.HasFace() {
		t.Error("thread tool should not accept a planar face")
	}
	if tool.CanCommit() {
		t.Error("thread tool should not be committable without a cylindrical face")
	}
}
