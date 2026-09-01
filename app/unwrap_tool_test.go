// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/model/compdef"
)

// #2047: UnwrapDefinition and ModifyFeatures.AddUnwrap were implemented and routed over the API,
// but no ribbon command, tool, dialog or menu entry referenced them — a case-insensitive search
// for "unwrap" across app/commands_*.go and head/ui/ returned zero hits.

// TestUnwrapToolFlattensTheCylindricalFace drives the UI: start the tool, click the cylinder's
// curved face, OK — and asserts the appended sheet is the face's development, arc length ×
// axial height.
func TestUnwrapToolFlattensTheCylindricalFace(t *testing.T) {
	t.Parallel()
	s, cyl := newPartWithCylinder(t) // r = 0.5, h = 2
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	before := def.SurfaceBodies().Count()
	s.SetPicker(stubPicker{sel: FaceHandle{Face: cylinderFaceOf(t, cyl), Body: cyl}})

	tool := NewUnwrapTool()
	s.StartTool(tool)
	if tool.CanCommit() {
		t.Fatal("unwrap should need a face before it can commit")
	}
	s.Click(50, 50)
	if !tool.CanCommit() {
		t.Fatal("unwrap not ready after picking the cylindrical face")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	if got := def.SurfaceBodies().Count(); got != before+1 {
		t.Fatalf("part holds %d bodies after Unwrap, want %d (the flat patch is appended)", got, before+1)
	}
	patch := def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1)
	// The development of a full r=0.5 h=2 cylinder is 2πr × h.
	want := 2 * stdmath.Pi * 0.5 * 2
	if got := query.BodyGeometryProperties(patch, ops.DefaultQuality()).Area; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("flat patch area = %g, want the development %g (2πr·h)", got, want)
	}
}

// A face that is not a cylinder cannot be developed: the tool must report why and stay open
// rather than leaving a sick node in the tree.
func TestUnwrapToolRefusesANonCylindricalFace(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: face, Body: body}})

	tool := NewUnwrapTool()
	s.StartTool(tool)
	s.Click(50, 50)
	if err := s.OK(); err == nil {
		t.Error("unwrapping a planar face should error, not append a patch")
	}
	if s.ActiveTool() == nil {
		t.Error("a failed unwrap should keep the tool open so the user can pick another face")
	}
}

// A second pick replaces the first: the feature flattens exactly one face.
func TestUnwrapToolKeepsOnlyTheLastFace(t *testing.T) {
	t.Parallel()
	s, cyl := newPartWithCylinder(t)
	tool := NewUnwrapTool()
	s.StartTool(tool)
	face := cylinderFaceOf(t, cyl)
	tool.Pick(s, FaceHandle{Face: face, Body: cyl})
	tool.Pick(s, FaceHandle{Face: face, Body: cyl})
	if got := len(tool.Picks()); got != 1 {
		t.Errorf("unwrap holds %d picks, want 1", got)
	}
}

// Unwrap is reachable from the ribbon, which is the whole point of the issue. It sits on the
// Surface panel because its output is a sheet body, not a solid modification.
func TestUnwrapIsOnTheSurfacePanel(t *testing.T) {
	t.Parallel()
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab(tabSurfacesMesh)
	if !ok {
		t.Fatal("ribbon has no Surfaces & Mesh tab")
	}
	panel, ok := tab.Panel("Surface")
	if !ok {
		t.Fatal("Surfaces & Mesh tab has no Surface panel")
	}
	if _, ok := buttonNamed(panel, "Unwrap"); !ok {
		t.Error("the Surface panel has no Unwrap button")
	}
}
