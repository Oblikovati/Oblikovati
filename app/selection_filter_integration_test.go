// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// downLookingBox builds a box and installs a real RayPicker with a camera looking straight down,
// so a centre-pixel click resolves to the top face — the production pick path the selection
// workflows use, not a stub.
func downLookingBox(t *testing.T) *Session {
	t.Helper()
	s := extrudedBox(t, 2, 4) // [0,2]×[0,2]×[0,4]
	cam := scene.NewCamera(400, 400)
	cam.Eye, cam.Target, cam.Up = math.P3(1, 1, 20), math.P3(1, 1, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	s.SetPicker(NewRayPicker(cam, partBodies(s)))
	return s
}

// TestAmbientFilterGovernsFacePreselection drives the no-tool face pre-selection that
// sketch-on-face and feature pre-selection depend on, through the real picker (#1222): the
// default filter selects the face, disabling Faces stops it, Deselect All blocks everything, and
// Select All restores it.
func TestAmbientFilterGovernsFacePreselection(t *testing.T) {
	t.Parallel()
	s := downLookingBox(t)

	s.Click(200, 200)
	if _, ok := s.Selection().First().(FaceHandle); !ok {
		t.Fatalf("default ambient filter: want a pre-selected FaceHandle, got %T", s.Selection().First())
	}

	s.Selection().Clear()
	s.SelectionFilterState().SetEnabled(SelectFace, false)
	s.Click(200, 200)
	if _, ok := s.Selection().First().(FaceHandle); ok {
		t.Fatal("Faces disabled: a click must not pre-select a face")
	}

	s.Selection().Clear()
	s.SelectionFilterState().DisableAll()
	s.Click(200, 200)
	if got := s.Selection().First(); got != nil {
		t.Fatalf("Deselect All: a click must select nothing, got %T", got)
	}

	s.Selection().Clear()
	s.SelectionFilterState().EnableAll()
	s.Click(200, 200)
	if _, ok := s.Selection().First().(FaceHandle); !ok {
		t.Fatalf("Select All restored: want a FaceHandle, got %T", s.Selection().First())
	}
}

// TestToolFilterWinsOverDisabledAmbient confirms an active tool's filter still governs picking
// even when the user has ambiently disabled everything in the Selection Filter window — so the
// offset-work-plane-from-face workflow keeps working regardless of the ambient setting (#1222).
func TestToolFilterWinsOverDisabledAmbient(t *testing.T) {
	t.Parallel()
	s := downLookingBox(t)
	s.SelectionFilterState().DisableAll() // user blocked every kind ambiently

	tool := NewOffsetWorkPlaneTool() // Start sets a {WorkPlane, Face} filter
	s.StartTool(tool)
	s.Click(200, 200) // pick the top planar face through the real picker
	if !tool.BasePicked() {
		t.Fatal("an active tool's face filter must win over the disabled ambient filter")
	}
}
