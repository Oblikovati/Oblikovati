// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/model/feature"
)

// #2050: several modify-family variants were routed over api/wire but had no ribbon path — not
// superseded wrappers, but capabilities the tool beside them could not express.

// TestMoveFaceToolBuildsARotation drives the rotate mode: the tool used to call AddMoveFace
// only, so rotating a face was API-only.
func TestMoveFaceToolBuildsARotation(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: face, Body: body}})

	tool := NewMoveFaceTool()
	s.StartTool(tool)
	s.Click(50, 50)
	tool.rotate = true
	if tool.CanCommit() {
		t.Fatal("rotate mode should not commit with a zero angle")
	}
	tool.angle = 10 * radPerDeg
	if !tool.CanCommit() {
		t.Fatal("rotate mode should commit with a face, an axis and an angle")
	}
	draft, ok := tool.DraftFeature(s)
	if !ok {
		t.Fatal("rotate mode produced no draft for the commit gate")
	}
	if _, isMove := draft.(*feature.MoveFaceFeature); !isMove {
		t.Fatalf("draft is %T, want a MoveFaceFeature", draft)
	}
}

// The rotate toggle switches which parameters the panel shows, so the axis and angle are
// reachable at all.
func TestMoveFaceToolExposesRotateParameters(t *testing.T) {
	t.Parallel()
	tool := NewMoveFaceTool()
	if got := len(tool.Params().Floats); got != 3 {
		t.Errorf("translate mode shows %d float rows, want 3 (Δ X/Y/Z)", got)
	}
	if len(tool.Params().Bools) != 1 {
		t.Error("the Rotate toggle is missing from the panel")
	}
	tool.Params().Bools[0].Set(true)
	if got := len(tool.Params().Floats); got != 7 {
		t.Errorf("rotate mode shows %d float rows, want 7 (axis point, direction, angle)", got)
	}
}

// TestReplaceFaceToolTargetsAWorkPlane: the tool used to call AddReplaceFace only, so a work
// plane could never be the target.
func TestReplaceFaceToolTargetsAWorkPlane(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)

	tool := NewReplaceFaceTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{Face: face, Body: body})
	if tool.CanCommit() {
		t.Fatal("replace face should need a target")
	}
	tool.SetPickingTarget(true)
	tool.Pick(s, WorkPlaneHandle{Plane: def.OriginPlanes()[0]})
	if _, onPlane := tool.TargetPlane(); !onPlane {
		t.Fatal("picking a work plane in target mode did not record it as the target")
	}
	if !tool.CanCommit() {
		t.Fatal("replace face should commit with faces and a plane target")
	}
	draft, ok := tool.DraftFeature(s)
	if !ok {
		t.Fatal("a plane-target replace produced no draft")
	}
	rf, isReplace := draft.(*feature.ReplaceFaceFeature)
	if !isReplace {
		t.Fatalf("draft is %T, want a ReplaceFaceFeature", draft)
	}
	if len(rf.TargetPlanes()) != 1 {
		t.Errorf("the draft froze %d target planes, want 1", len(rf.TargetPlanes()))
	}
}

// A face target still wins after a plane was picked, and vice versa — the two are exclusive.
func TestReplaceFaceTargetIsExclusive(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	tool := NewReplaceFaceTool()
	s.StartTool(tool)
	tool.SetPickingTarget(true)
	tool.Pick(s, WorkPlaneHandle{Plane: def.OriginPlanes()[0]})
	tool.Pick(s, FaceHandle{Face: face, Body: body})
	if _, onPlane := tool.TargetPlane(); onPlane {
		t.Error("picking a face target left the earlier plane target in place")
	}
}

// TestThickenToolCarriesTheOptions: the tool set none of the #1876 options, so the ribbon could
// only ever build the symmetric, joined, solid thicken.
func TestThickenToolCarriesTheOptions(t *testing.T) {
	t.Parallel()
	tool := NewThickenTool()
	tool.SetThickness(0.5)
	tool.SetDirectionIndex(1) // positive
	tool.SetOperationIndex(1) // cut
	tool.SetAsSurface(true)

	pf := tool.addThicken(feature.NewPartFeatures(nil))
	def := pf.Definition().(*feature.ThickenFeature)
	if def.Direction() != ops.ThickenPositive {
		t.Errorf("direction = %v, want ThickenPositive", def.Direction())
	}
	if def.Operation() != ops.Cut {
		t.Errorf("operation = %v, want Cut", def.Operation())
	}
	if !def.AsSurface() {
		t.Error("as-surface was not carried onto the definition")
	}
}

// A fresh thicken keeps the pre-#1876 defaults, so adding the options changed no behaviour for
// anyone who does not touch them.
func TestThickenToolDefaultsAreUnchanged(t *testing.T) {
	t.Parallel()
	pf := NewThickenTool().addThicken(feature.NewPartFeatures(nil))
	def := pf.Definition().(*feature.ThickenFeature)
	if def.Direction() != ops.ThickenSymmetric || def.Operation() != ops.Join || def.AsSurface() {
		t.Errorf("default thicken = %v/%v/asSurface=%v, want symmetric/join/false",
			def.Direction(), def.Operation(), def.AsSurface())
	}
}

// TestSimplifyToolReducesTheBody: the Simplify panel held only Derive and Shrinkwrap, so the
// reduction itself had no tool at all.
func TestSimplifyToolReducesTheBody(t *testing.T) {
	t.Parallel()
	// Chamfer a 2×2×2 block (vol 7.75), then simplify the chamfer away — a face whose opening
	// the neighbours CAN heal, so the reduction is the sharp box again.
	s, block := newPartWithBlock(t, 2)
	chamfered := chamferOneEdge(t, s, block)
	s.SetPicker(stubPicker{sel: chamferFaceHandleOf(t, chamfered)})

	tool := NewSimplifyTool()
	s.StartTool(tool)
	if tool.CanCommit() {
		t.Fatal("simplify with nothing to do should not commit")
	}
	s.Click(50, 50)
	if !tool.CanCommit() {
		t.Fatal("simplify should commit once a face is picked")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("simplified body not a valid solid: %+v", r)
	}
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(got-8) > 1e-6 {
		t.Errorf("simplified volume = %g, want the healed 8", got)
	}
}

// Fill voids alone is enough to commit: the reduction need not remove any face.
func TestSimplifyToolCommitsOnFillVoidsAlone(t *testing.T) {
	t.Parallel()
	tool := NewSimplifyTool()
	tool.SetFillVoids(true)
	if !tool.CanCommit() {
		t.Error("fill-voids alone should be a committable simplify")
	}
}

// Simplify is reachable from the ribbon, beside Derive and Shrinkwrap.
func TestSimplifyIsOnTheManageSimplifyPanel(t *testing.T) {
	t.Parallel()
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab("Manage")
	if !ok {
		t.Fatal("ribbon has no Manage tab")
	}
	panel, ok := tab.Panel("Simplify")
	if !ok {
		t.Fatal("Manage tab has no Simplify panel")
	}
	if _, ok := buttonNamed(panel, "Simplify"); !ok {
		t.Error("the Simplify panel has no Simplify button")
	}
}
