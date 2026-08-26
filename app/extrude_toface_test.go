// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"slices"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// planeAtZ (a horizontal sketch plane at height z, the "to face" termination target) is shared
// with loft_tool_test.go.

// TestExtrudeToFaceStepsAndTarget pins the engine-driven two-step flow: extrude picks a profile,
// and once "to face" is chosen it switches to accepting a termination face/plane, gates commit on
// it, and folds it into the extent (#1222 selection engine, the To-Face proof case).
func TestExtrudeToFaceStepsAndTarget(t *testing.T) {
	tool := NewExtrudeTool()
	if k := tool.AcceptedKinds(); len(k) != 1 || k[0] != SelectProfile {
		t.Fatalf("initial AcceptedKinds = %v, want [SelectProfile]", k)
	}
	tool.Pick(nil, ProfileHandle{})
	tool.SetExtentType(feature.ToFaceExtent)

	k := tool.AcceptedKinds()
	if !kindsContain(k, SelectFace) || !kindsContain(k, SelectWorkPlane) {
		t.Fatalf("to-face AcceptedKinds = %v, want face + work plane", k)
	}
	if tool.CanCommit() {
		t.Fatal("to-face must not be committable before a termination target is picked")
	}
	tool.Pick(nil, WorkPlaneHandle{Plane: feature.NewFixedWorkPlane(planeAtZ(3))})
	if !tool.CanCommit() {
		t.Fatal("to-face should be committable once a termination is picked")
	}
	if tool.buildExtent().ToPlane == nil {
		t.Fatal("buildExtent must carry the to-face termination plane")
	}
}

// TestExtrudeToFaceTerminatesAtPlane drives the whole flow and asserts the solid actually
// terminates at the picked plane (height = the plane's z), not at a typed distance.
func TestExtrudeToFaceTerminatesAtPlane(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	tool := NewExtrudeTool()
	s.StartTool(tool)
	tool.Pick(s, profile)
	tool.SetExtentType(feature.ToFaceExtent)
	tool.Pick(s, WorkPlaneHandle{Plane: feature.NewFixedWorkPlane(planeAtZ(3))})
	if err := s.OK(); err != nil {
		t.Fatalf("OK (to-face extrude): %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	if h := float64(body.RangeBox().Diagonal().Z); h < 2.99 || h > 3.01 {
		t.Errorf("to-face extrude height = %v, want ~3 (terminated at the picked plane)", h)
	}
}

// TestExtrudeToFacePicksIncludeTermination: the termination face shows in the tool's Picks so the
// engine highlights it like any other selection.
func TestExtrudeToFacePicksIncludeTermination(t *testing.T) {
	s := downLookingBoxWithPlanes(t)
	tool := NewExtrudeTool()
	s.StartTool(tool)
	tool.Pick(s, ProfileHandle{})
	tool.SetExtentType(feature.ToFaceExtent)
	sel, _ := s.PickAt(200, 200, NewSelectionFilter(SelectFace))
	tool.Pick(s, sel)
	picks := tool.Picks()
	if len(picks) == 0 {
		t.Fatal("Picks must include the picked termination face")
	}
	if _, ok := picks[len(picks)-1].(FaceHandle); !ok {
		t.Fatalf("last pick = %T, want the termination FaceHandle", picks[len(picks)-1])
	}
}

func kindsContain(ks []SelectionKind, want SelectionKind) bool {
	return slices.Contains(ks, want)
}
