// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/model/feature"
)

// The tools this epic added share a surface every tool has: a name, a prompt that guides the
// next step, a cancel that leaves no state behind, and a session bridge that resolves the
// running tool. This covers that surface for all of them in one place — the per-issue tests
// assert what each tool BUILDS; this asserts they behave like tools.

// TestNewToolsNameAndPromptTheirSteps checks each tool names itself and prompts differently
// before and after its inputs are gathered, so the status bar guides rather than repeats.
func TestNewToolsNameAndPromptTheirSteps(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)

	// Delete Body has no entry here: it is a browser action, not a tool (delete_body_test.go).
	for _, tc := range []struct {
		name string
		tool Tool
	}{
		{"Unwrap", NewUnwrapTool()},
		{"Simplify", NewSimplifyTool()},
		{"Feature Control Frame", NewModelFrameTool()},
		{"Datum Feature", NewModelDatumTool()},
		{"Angle to Plane", NewAngleWorkPlaneTool()},
		{"Edit Freeform Cage", NewFreeformCageEditTool()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tool.Name(); got != tc.name {
				t.Errorf("Name = %q, want %q", got, tc.name)
			}
			p, ok := tc.tool.(interface{ Prompt(*Session) string })
			if !ok {
				t.Fatal("the tool declares no Prompt, so the status bar cannot guide it")
			}
			first := p.Prompt(s)
			if strings.TrimSpace(first) == "" {
				t.Error("the opening prompt is empty")
			}
			// Feed the tool its first input; the prompt must move on.
			switch tool := tc.tool.(type) {
			case *UnwrapTool:
				tool.Pick(s, FaceHandle{Face: face, Body: body})
			case *SimplifyTool:
				tool.Pick(s, FaceHandle{Face: face, Body: body})
			case *ModelToleranceTool:
				tool.Pick(s, FaceHandle{Face: face, Body: body})
			case *AngleWorkPlaneTool:
				tool.Pick(s, WorkAxisHandle{Axis: originAxis(t, def, feature.OriginXAxis)})
			case *FreeformCageEditTool:
				return // drag-driven: one prompt, no steps
			}
			if p.Prompt(s) == first {
				t.Errorf("the prompt did not change after the first input: still %q", first)
			}
			tc.tool.Cancel(s)
		})
	}
}

// The session bridges resolve the running tool and answer nil for anything else, which is what
// keeps each dialog drawing only while its own tool is active.
func TestNewSessionBridgesResolveOnlyTheirTool(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	check := func(name string, got any, want bool) {
		t.Helper()
		isNil := got == nil || (got != nil && isNilPointer(got))
		if want == isNil {
			t.Errorf("%s: resolved=%v, want resolved=%v", name, !isNil, want)
		}
	}
	check("ActiveUnwrap idle", s.ActiveUnwrap(), false)
	check("ActiveSimplify idle", s.ActiveSimplify(), false)
	check("ActiveModelTolerance idle", s.ActiveModelTolerance(), false)
	check("ActiveAnglePlane idle", s.ActiveAnglePlane(), false)
	check("ActiveCageEdit idle", s.ActiveCageEdit(), false)

	s.StartTool(NewUnwrapTool())
	check("ActiveUnwrap running", s.ActiveUnwrap(), true)
	check("ActiveSimplify while unwrap runs", s.ActiveSimplify(), false)

	s.StartTool(NewAngleWorkPlaneTool())
	check("ActiveAnglePlane running", s.ActiveAnglePlane(), true)
	check("ActiveUnwrap after switching", s.ActiveUnwrap(), false)
}

// isNilPointer reports whether an interface holds a nil typed pointer, which is what a session
// bridge returns when its tool is not the running one.
func isNilPointer(v any) bool {
	switch p := v.(type) {
	case *UnwrapTool:
		return p == nil
	case *SimplifyTool:
		return p == nil
	case *ModelToleranceTool:
		return p == nil
	case *AngleWorkPlaneTool:
		return p == nil
	case *FreeformCageEditTool:
		return p == nil
	default:
		return v == nil
	}
}

// The cage tool's accessors clamp and round-trip, and its session bridge refuses when there is
// no free-form body to edit.
func TestFreeformCageToolAccessors(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if s.CanEditFreeformCage() {
		t.Error("a part with no freeform body should not offer cage editing")
	}
	if s.ApplyActiveCageLevel() || s.CreaseActiveCageHandle() {
		t.Error("the cage actions should refuse with no tool running")
	}

	tool := NewFreeformCageEditTool()
	s.StartTool(tool)
	if tool.LastVertex() != -1 {
		t.Errorf("LastVertex = %d before any drag, want -1", tool.LastVertex())
	}
	tool.SetSharpness(2.5)
	if tool.Sharpness() != 1 {
		t.Errorf("Sharpness = %g, want clamping to 1", tool.Sharpness())
	}
	tool.SetSharpness(-1)
	if tool.Sharpness() != 0 {
		t.Errorf("Sharpness = %g, want clamping to 0", tool.Sharpness())
	}
	if tool.CanCommit() {
		t.Error("the cage editor commits per drag, so CanCommit must stay false")
	}
	if err := tool.Commit(s); err != nil {
		t.Errorf("Commit should be a no-op, got %v", err)
	}
	tool.Cancel(s)
}

// The angle-plane tool's picks and clear behave, and it refuses to commit half-gathered.
func TestAngleWorkPlaneToolAccessors(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	tool := NewAngleWorkPlaneTool()
	s.StartTool(tool)
	if tool.AxisPicked() || tool.BasePicked() {
		t.Error("nothing should be picked on a fresh angle-plane tool")
	}
	if err := tool.Commit(s); err == nil {
		t.Error("committing with nothing gathered should error")
	}
	face, body := boxTopFace(t, def)
	tool.Pick(s, FaceHandle{Face: face, Body: body}) // a planar face is a valid base
	if !tool.BasePicked() {
		t.Error("a face pick did not record the base plane")
	}
	tool.SetAngleDegrees(45)
	if got := tool.AngleDegrees(); got < 44.9 || got > 45.1 {
		t.Errorf("AngleDegrees = %g, want 45", got)
	}
	tool.ClearPicks()
	if tool.AxisPicked() || tool.BasePicked() {
		t.Error("ClearPicks left a pick behind")
	}
	tool.Cancel(s)
}

// The model-tolerance tool's accessors round-trip and its geometry clear works.
func TestModelToleranceToolAccessors(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	tool := NewModelFrameTool()
	s.StartTool(tool)
	if tool.DatumMode() {
		t.Error("the frame tool should not be in datum mode")
	}
	if len(tool.Picks()) != 0 {
		t.Error("a fresh tool reports picks")
	}
	face, body := boxTopFace(t, def)
	tool.Pick(s, FaceHandle{Face: face, Body: body})
	if !tool.GeometryPicked() || len(tool.Picks()) != 1 {
		t.Error("the face pick was not recorded")
	}
	tool.SetCharacteristicIndex(-1) // out of range is ignored
	if tool.CharacteristicIndex() < 0 {
		t.Error("an out-of-range characteristic index was accepted")
	}
	tool.SetDatums("A")
	if tool.Datums() != "A" {
		t.Errorf("Datums = %q, want A", tool.Datums())
	}
	tool.SetValue(0.25)
	if tool.Value() != 0.25 {
		t.Errorf("Value = %g, want 0.25", tool.Value())
	}
	datum := NewModelDatumTool()
	datum.SetLabel(" B ")
	if datum.Label() != "B" {
		t.Errorf("Label = %q, want the trimmed B", datum.Label())
	}
	tool.ClearGeometry()
	if tool.GeometryPicked() {
		t.Error("ClearGeometry left the reference behind")
	}
	tool.Cancel(s)
}

// The unwrap and simplify tools clear their picks and refuse an empty commit.
func TestUnwrapAndSimplifyAccessors(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)

	u := NewUnwrapTool()
	s.StartTool(u)
	u.Pick(s, FaceHandle{Face: face, Body: body})
	if !u.FacePicked() || len(u.Picks()) != 1 {
		t.Error("unwrap did not record the face")
	}
	u.ClearFace()
	if u.FacePicked() || u.CanCommit() {
		t.Error("ClearFace left the face behind")
	}
	u.Cancel(s)

	sp := NewSimplifyTool()
	s.StartTool(sp)
	if len(sp.Picks()) != 0 || sp.FillVoids() {
		t.Error("a fresh simplify reports picks or void filling")
	}
	sp.Pick(s, FaceHandle{Face: face, Body: body})
	if len(sp.Picks()) != 1 {
		t.Errorf("simplify reports %d picks for the unified highlight, want 1", len(sp.Picks()))
	}
	sp.Pick(s, FaceHandle{Face: face, Body: body}) // a duplicate is ignored
	if sp.FaceCount() != 1 || len(sp.Faces()) != 1 {
		t.Errorf("simplify holds %d faces, want 1", sp.FaceCount())
	}
	sp.ClearFaces()
	if sp.FaceCount() != 0 || sp.CanCommit() {
		t.Error("ClearFaces left a face behind")
	}
	sp.Cancel(s)
}

// The datum guided-pick tool narrows the selection filter to the kinds its constructor needs,
// and stops accepting everything else while it runs.
func TestDatumPickToolNarrowsTheSelectionFilter(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	tool := newMidplaneWorkPlaneTool()
	s.StartTool(tool)
	f := s.Selection().Filter()
	if f == nil || !f.Accepts(SelectWorkPlane) {
		t.Fatal("the midplane tool did not filter selection to work planes")
	}
	if f.Accepts(SelectEdge) {
		t.Error("the midplane tool accepts edges, which it cannot use")
	}
	if tool.CanCommit() {
		t.Error("a midplane needs two planes before it can commit")
	}
	s.CancelTool()
	if s.ActiveTool() != nil {
		t.Error("cancel left the datum tool running")
	}
}

// The cage overlay is empty for a part with no free-form body rather than panicking.
func TestCagePreviewIsEmptyWithoutABody(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if items := NewFreeformCageEditTool().Preview(s); len(items) != 0 {
		t.Errorf("cage preview drew %d items with no freeform body", len(items))
	}
	if _, _, ok := activeFreeformCage(s); ok {
		t.Error("activeFreeformCage found a body on a part with none")
	}
	if s.CageEditActive() || s.CageDragActive() {
		t.Error("no cage editing should be active")
	}
	if s.BeginCageDrag(0, 0) {
		t.Error("BeginCageDrag should refuse with no tool running")
	}
	s.UpdateCageDrag(1, 1) // must not panic with no drag in flight
	s.CommitCageDrag()
	if s.CreaseCageEdgesAround(0, 1) {
		t.Error("CreaseCageEdgesAround should refuse with no body")
	}
}
