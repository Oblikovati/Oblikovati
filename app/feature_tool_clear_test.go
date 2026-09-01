// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The property panels' selector chips clear one pick each (⊗). These lock the clears:
// the pick empties, the tool stops being committable, and the breadcrumb name follows.

func TestRevolveClearProfileEmptiesSelection(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithSquare(t, 2)
	rv := NewRevolveTool()
	s.StartTool(rv)
	rv.Pick(s, profile)
	if !rv.CanCommit() {
		t.Fatal("a picked profile should make the revolve committable")
	}
	if rv.SourceSketchName() == "" {
		t.Error("SourceSketchName() = \"\" with a picked profile, want the sketch's name")
	}
	rv.ClearProfile()
	if _, ok := rv.PickedProfile(); ok || rv.CanCommit() {
		t.Error("after ClearProfile the revolve must have no profile and not be committable")
	}
	if name := rv.SourceSketchName(); name != "" {
		t.Errorf("SourceSketchName() = %q after clearing, want \"\"", name)
	}
}

func TestSweepClearsProfileAndPathIndependently(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithSquare(t, 2)
	sw := NewSweepTool()
	s.StartTool(sw)
	sw.Pick(s, profile)
	sw.Pick(s, PathHandle{Sketch: profile.Sketch, PathIndex: 0})
	if !sw.CanCommit() {
		t.Fatal("profile + path should make the sweep committable")
	}
	sw.ClearPath()
	if _, ok := sw.PickedPath(); ok {
		t.Error("after ClearPath the sweep must have no path")
	}
	if _, ok := sw.PickedProfile(); !ok {
		t.Error("ClearPath must not drop the profile pick")
	}
	sw.ClearProfile()
	if _, ok := sw.PickedProfile(); ok || sw.SourceSketchName() != "" {
		t.Error("after ClearProfile the sweep must have no profile and no source sketch name")
	}
}

func TestHoleClearFaceEmptiesSelection(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithSquare(t, 2)
	h := NewHoleTool()
	s.StartTool(h)
	h.Pick(s, FaceHandle{})
	if _, ok := h.PickedFace(); !ok {
		t.Fatal("picking a face handle should set the placement face")
	}
	h.ClearFace()
	if _, ok := h.PickedFace(); ok || h.CanCommit() {
		t.Error("after ClearFace the hole must have no face and not be committable")
	}
}

// TestEdgeAndFaceToolClears sweeps the simple multi-pick tools: each Clear empties its
// pick set so the chip returns to its required/empty state.
func TestEdgeAndFaceToolClears(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithSquare(t, 2)
	fl := NewFilletTool()
	s.StartTool(fl)
	fl.Pick(s, EdgeHandle{})
	fl.ClearEdges()
	if len(fl.Edges()) != 0 {
		t.Error("FilletTool.ClearEdges left edges picked")
	}
	ch := NewChamferTool()
	s.StartTool(ch)
	ch.Pick(s, EdgeHandle{})
	ch.ClearEdges()
	if len(ch.Edges()) != 0 {
		t.Error("ChamferTool.ClearEdges left edges picked")
	}
	for name, tool := range faceToolsForClearTest(s) {
		if n := tool(); n != 0 {
			t.Errorf("%s clear left %d faces picked", name, n)
		}
	}
}

// faceToolsForClearTest starts each face-pick tool, picks one face, clears it, and
// returns the remaining count per tool name.
func faceToolsForClearTest(s *Session) map[string]func() int {
	sh := NewShellTool()
	s.StartTool(sh)
	sh.Pick(s, FaceHandle{})
	fo := NewFaceOffsetTool()
	s.StartTool(fo)
	fo.Pick(s, FaceHandle{})
	dr := NewDraftTool()
	s.StartTool(dr)
	dr.Pick(s, FaceHandle{})
	df := NewDeleteFaceTool()
	s.StartTool(df)
	df.Pick(s, FaceHandle{})
	return map[string]func() int{
		"ShellTool.ClearFaces":      func() int { sh.ClearFaces(); return len(sh.Faces()) },
		"FaceOffsetTool.ClearFaces": func() int { fo.ClearFaces(); return len(fo.Faces()) },
		"DraftTool.ClearFaces":      func() int { dr.ClearFaces(); return len(dr.Faces()) },
		"DeleteFaceTool.ClearFaces": func() int { df.ClearFaces(); return len(df.Faces()) },
	}
}

// TestReplaceFaceClearsFacesAndTargetIndependently locks the two chips' clears.
func TestReplaceFaceClearsFacesAndTargetIndependently(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithSquare(t, 2)
	r := NewReplaceFaceTool()
	s.StartTool(r)
	r.Pick(s, FaceHandle{})
	r.SetPickingTarget(true)
	r.Pick(s, FaceHandle{})
	if _, ok := r.PickedTarget(); !ok {
		t.Fatal("target pick did not register")
	}
	r.ClearTarget()
	if _, ok := r.PickedTarget(); ok {
		t.Error("ClearTarget left a target picked")
	}
	if len(r.Faces()) == 0 {
		t.Error("ClearTarget must not drop the replace-faces pick")
	}
	r.ClearFaces()
	if len(r.Faces()) != 0 {
		t.Error("ClearFaces left faces picked")
	}
}

// TestSplitCoilOffsetPlaneLoftClears locks the single-pick and loft clears.
func TestSplitCoilOffsetPlaneLoftClears(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithSquare(t, 2)
	co := NewCoilTool()
	s.StartTool(co)
	co.Pick(s, profile)
	if co.SourceSketchName() == "" {
		t.Error("CoilTool.SourceSketchName() empty with a picked profile")
	}
	co.ClearProfile()
	if _, ok := co.PickedProfile(); ok || co.SourceSketchName() != "" {
		t.Error("CoilTool.ClearProfile left a profile (or its sketch name) behind")
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	op := NewOffsetWorkPlaneTool()
	s.StartTool(op)
	op.Pick(s, WorkPlaneHandle{Plane: wp})
	if !op.BasePicked() {
		t.Fatal("offset-plane tool did not capture the picked plane")
	}
	op.ClearBase()
	if op.BasePicked() {
		t.Error("OffsetWorkPlaneTool.ClearBase left the base picked")
	}
	lf := NewLoftTool()
	s.StartTool(lf)
	lf.Pick(s, profile)
	lf.Pick(s, PathHandle{Sketch: profile.Sketch})
	lf.ClearSections()
	lf.ClearGuides()
	if lf.SectionCount() != 0 || lf.RailCount() != 0 || lf.HasCenterline() || lf.MapCurveCount() != 0 {
		t.Error("LoftTool clears left sections or guides picked")
	}
}

// TestThreadClearFaceEmptiesSelection uses a real cylinder (Pick validates the face).
func TestThreadClearFaceEmptiesSelection(t *testing.T) {
	t.Parallel()
	s, cyl := newPartWithCylinder(t)
	tt := NewThreadTool()
	s.StartTool(tt)
	tt.Pick(s, FaceHandle{Face: cylinderFaceOf(t, cyl), Body: cyl})
	if !tt.HasFace() {
		t.Fatal("thread tool did not capture the cylindrical face")
	}
	tt.ClearFace()
	if tt.HasFace() {
		t.Error("ThreadTool.ClearFace left the face picked")
	}
}
