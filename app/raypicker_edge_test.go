// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/scene"
)

// TestRayPickerSelectsEdge aims a ray straight at a vertical edge of a box and checks the
// picker returns that edge — the path the chamfer/fillet tools rely on (previously the
// RayPicker had no edge picking, so edge selection silently did nothing in the head).
func TestRayPickerSelectsEdge(t *testing.T) {
	s := extrudedBox(t, 2, 4) // box [0,2]×[0,2]×[0,4]
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(10, 0, 2)
	cam.Target = math.P3(2, 0, 2)
	p := NewRayPicker(cam, partBodies(s))

	dir, _ := math.UnitVector3FromVector(math.V3(-1, 0, 0))
	e := p.nearestEdge(math.P3(10, 0, 2), dir.AsVector()) // straight at the (x=2,y=0) edge
	if e == nil {
		t.Fatal("nearestEdge found no edge aiming at the (2,0) vertical edge")
	}
	a, b := e.StartVertex().Point(), e.EndVertex().Point()
	if !(a.X == 2 && a.Y == 0 && b.X == 2 && b.Y == 0) {
		t.Errorf("picked edge endpoints %v..%v, want the vertical edge at (2,0)", a, b)
	}
}

// TestRayPickerEdgeFilter checks the edge pick is gated by the filter: a face-only filter
// near an edge does not return an edge (it falls through to the face).
func TestRayPickerEdgeFilter(t *testing.T) {
	s := extrudedBox(t, 2, 4)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(10, 0, 2)
	cam.Target = math.P3(2, 0, 2)
	p := NewRayPicker(cam, partBodies(s))
	s.SetPicker(p)

	s.Selection().SetFilter(NewSelectionFilter(SelectEdge))
	if sel, ok := p.Pick(200, 200, NewSelectionFilter(SelectEdge)); ok {
		if _, isEdge := sel.(EdgeHandle); !isEdge {
			t.Errorf("edge filter returned %T, want EdgeHandle", sel)
		}
	}
}

// TestRaySegmentDistance checks the ray↔segment closest distance and parameter.
func TestRaySegmentDistance(t *testing.T) {
	dir := math.V3(1, 0, 0)
	// A segment crossing the ray at (5,0,0): distance 0 at t=5.
	if d, tt, ok := raySegmentDistance(math.P3(0, 0, 0), dir, math.P3(5, 1, 0), math.P3(5, -1, 0)); !ok || d > 1e-9 || stdmath.Abs(tt-5) > 1e-9 {
		t.Errorf("crossing: dist=%g t=%g ok=%v, want 0 at t=5", d, tt, ok)
	}
	// A segment offset 2 in Y from the ray: closest distance 2.
	if d, _, ok := raySegmentDistance(math.P3(0, 0, 0), dir, math.P3(5, 2, 0), math.P3(5, 3, 0)); !ok || stdmath.Abs(d-2) > 1e-9 {
		t.Errorf("offset: dist=%g, want 2", d)
	}
}
