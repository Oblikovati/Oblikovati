// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// sketch3DPicker builds a session whose picker sees one 3D sketch, with the camera
// looking straight down the Z axis at the given target.
func sketch3DPicker(t *testing.T, target math.Point3) (*Session, *sketch.Sketch3D) {
	t.Helper()
	s, _ := emptyPartSession(t)
	sk3, err := s.CreateSketch3D()
	if err != nil {
		t.Fatalf("create 3D sketch: %v", err)
	}
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(target.X, target.Y, 20)
	cam.Target = target
	cam.Up = math.V3(0, 1, 0)
	s.SetPicker(NewRayPicker(cam, partBodies(s)).
		WithSketches3D(func() []*sketch.Sketch3D { return []*sketch.Sketch3D{sk3} }))
	s.Selection().SetFilter(NewSelectionFilter(SelectSketchEntity))
	return s, sk3
}

// TestRayPickerSelectsSketch3DLine checks a viewport click lands on a 3D-sketch line —
// what feeds the 3D constraint tools their picks (issue #142).
func TestRayPickerSelectsSketch3DLine(t *testing.T) {
	t.Parallel()
	s, sk3 := sketch3DPicker(t, math.P3(1, 1, 0))
	l := sk3.AddLine3D(math.P3(0, 1, 0), math.P3(2, 1, 0)) // passes under the centre pixel

	s.Click(200, 200)
	if s.Selection().Count() != 1 {
		t.Fatalf("3D line pick selected %d, want 1", s.Selection().Count())
	}
	h, ok := s.Selection().First().(SketchEntityHandle)
	if !ok || h.Entity != l {
		t.Fatalf("selected %T (%v), want the 3D line", s.Selection().First(), ok)
	}
}

// TestRayPickerSelectsSketch3DCurveAndPoint checks sampled-curve picking (a helix seen
// down its axis) and standalone-point picking.
func TestRayPickerSelectsSketch3DCurveAndPoint(t *testing.T) {
	t.Parallel()
	zAxis, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("axis: %v", err)
	}
	// The helix winds around (0,0,z) with radius 1 — looking down at (1,0,0) hits its
	// start-side flank.
	s, sk3 := sketch3DPicker(t, math.P3(1, 0, 0))
	h := sk3.AddHelix3D(math.P3(0, 0, 0), zAxis, 1, 0.5, 0, 4, false)
	s.Click(200, 200)
	if got := s.Selection().First(); s.Selection().Count() != 1 || got.(SketchEntityHandle).Entity != h {
		t.Fatalf("helix pick selected %v (count %d), want the helix", got, s.Selection().Count())
	}

	s2, sk2 := sketch3DPicker(t, math.P3(5, 5, 2))
	p := sk2.AddPoint3D(math.P3(5, 5, 2))
	s2.Click(200, 200)
	if got := s2.Selection().First(); s2.Selection().Count() != 1 || got.(SketchEntityHandle).Entity != p {
		t.Fatalf("point pick selected %v (count %d), want the standalone point", got, s2.Selection().Count())
	}
}

// TestRayPickerSketch3DRespectsFilter: with a face-only filter, a 3D-sketch entity
// under the cursor must not be picked.
func TestRayPickerSketch3DRespectsFilter(t *testing.T) {
	t.Parallel()
	s, sk3 := sketch3DPicker(t, math.P3(1, 1, 0))
	sk3.AddLine3D(math.P3(0, 1, 0), math.P3(2, 1, 0))
	s.Selection().SetFilter(NewSelectionFilter(SelectFace))
	s.Click(200, 200)
	if s.Selection().Count() != 0 {
		t.Fatalf("filtered pick selected %d, want 0", s.Selection().Count())
	}
}
