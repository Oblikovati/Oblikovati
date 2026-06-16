// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

func TestPriorityFilterMapping(t *testing.T) {
	cases := []struct {
		p      SelectionPriority
		kind   SelectionKind
		accept bool
	}{
		{PriorityGeneral, SelectFace, true}, // accepts everything
		{PriorityGeneral, SelectEdge, true},
		{PriorityPart, SelectBody, true},
		{PriorityPart, SelectFace, false},
		{PriorityFace, SelectFace, true},
		{PriorityFace, SelectEdge, false},
		{PriorityEdge, SelectEdge, true},
		{PriorityEdge, SelectFace, false},
	}
	for _, c := range cases {
		if got := priorityFilter(c.p).Accepts(c.kind); got != c.accept {
			t.Errorf("priority %d accepts %d = %v, want %v", c.p, c.kind, got, c.accept)
		}
	}
}

func TestPickFilterUsesToolFilterThenPriority(t *testing.T) {
	s := NewSession()
	s.SetSelectionPriority(PriorityEdge)
	// No tool → the priority filter (edge-only).
	if s.pickFilter().Accepts(SelectFace) {
		t.Error("with no tool, pickFilter should be the edge-priority filter (no faces)")
	}
	// A tool active → its own filter wins (the default tool filter accepts all here).
	s.StartTool(stubSketchTool{})
	if !s.pickFilter().Accepts(SelectFace) {
		t.Error("with a tool active, pickFilter should be the tool's filter, not the priority")
	}
}

// rayPickerSession builds a box viewed head-on at a face, with the RayPicker installed, so a
// screen-centre click lands on the +X face centre.
func rayPickerSession(t *testing.T) *Session {
	t.Helper()
	s := extrudedBox(t, 2, 4) // [0,2]×[0,2]×[0,4]; +X face centre ≈ (2,1,2)
	cam := scene.NewCamera(400, 400)
	cam.Eye, cam.Target, cam.Up = math.P3(20, 1, 2), math.P3(0, 1, 2), math.V3(0, 0, 1)
	s.SetCamera(cam)
	s.SetPicker(NewRayPicker(cam, partBodies(s)))
	return s
}

func TestSelectionPriorityRibbonCombo(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if err := s.Execute("Select.Priority.Edge"); err != nil {
		t.Fatalf("execute Select.Priority.Edge: %v", err)
	}
	if s.SelectionPriority() != PriorityEdge {
		t.Errorf("the Edge-priority combo command did not set the priority: got %d", s.SelectionPriority())
	}
	if _, ok := s.Commands().ByID("Select.Priority.General"); !ok {
		t.Error("the General-priority combo command should be registered")
	}
}

func TestSelectionPriorityBiasesClick(t *testing.T) {
	s := rayPickerSession(t)

	// General priority: a click on a face selects the face.
	s.SetSelectionPriority(PriorityGeneral)
	s.Click(200, 200)
	if _, ok := s.Selection().First().(FaceHandle); !ok {
		t.Fatalf("General priority click = %T, want a FaceHandle", s.Selection().First())
	}

	// Part priority: the same click selects the whole body.
	s.Selection().Clear()
	s.SetSelectionPriority(PriorityPart)
	s.Click(200, 200)
	if _, ok := s.Selection().First().(BodyHandle); !ok {
		t.Fatalf("Part priority click = %T, want a BodyHandle", s.Selection().First())
	}
}
