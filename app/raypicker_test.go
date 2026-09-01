// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/scene"
)

// extrudedBox returns a session whose active part holds a real extruded box solid,
// built by driving the Extrude tool — so picking tests run against genuine geometry.
func extrudedBox(t *testing.T, side, height float64) *Session {
	t.Helper()
	s, profile := newPartWithSquare(t, side)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(0, 0)
	ext.SetDistance(height)
	if err := s.OK(); err != nil {
		t.Fatalf("build box: %v", err)
	}
	return s
}

func partBodies(s *Session) func() []*topo.Body {
	return func() []*topo.Body {
		def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
		return def.SurfaceBodies().All()
	}
}

func TestRayPickerSelectsTopFaceOfBox(t *testing.T) {
	t.Parallel()
	s := extrudedBox(t, 2, 4) // box [0,2]×[0,2]×[0,4]

	// Camera above the box looking straight down → the center pixel ray hits the top
	// (z=4) cap first.
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(1, 1, 20)
	cam.Target = math.P3(1, 1, 0)
	cam.Up = math.V3(0, 1, 0)
	picker := NewRayPicker(cam, partBodies(s))
	s.SetPicker(picker)
	s.Selection().SetFilter(NewSelectionFilter(SelectFace))

	s.Click(200, 200) // center pixel
	if s.Selection().Count() != 1 {
		t.Fatalf("ray pick selected %d, want 1 face", s.Selection().Count())
	}
	fh, ok := s.Selection().First().(FaceHandle)
	if !ok {
		t.Fatal("selected item is not a face")
	}
	// The picked face is the top cap: all its vertices are at z = 4.
	for _, v := range fh.Face.Vertices() {
		if v.Point().Z < 3.99 {
			t.Errorf("picked face is not the top cap (vertex z=%v)", v.Point().Z)
		}
	}
}

func TestRayPickerHonorsBodyFilter(t *testing.T) {
	t.Parallel()
	s := extrudedBox(t, 3, 4) // a larger box still centered under (1,1)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(1, 1, 20)
	cam.Target = math.P3(1, 1, 0)
	picker := NewRayPicker(cam, partBodies(s))
	s.SetPicker(picker)
	// With a body-only filter, a face hit resolves to its owning body.
	s.Selection().SetFilter(NewSelectionFilter(SelectBody))
	s.Click(200, 200)
	if _, ok := s.Selection().First().(BodyHandle); !ok {
		t.Errorf("body-filter pick = %T, want BodyHandle", s.Selection().First())
	}
}

func TestRayPickerMissSelectsNothing(t *testing.T) {
	t.Parallel()
	s := extrudedBox(t, 2, 4)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(1, 1, 20)
	cam.Target = math.P3(1, 1, 0)
	s.SetPicker(NewRayPicker(cam, partBodies(s)))
	s.Selection().SetFilter(NewSelectionFilter(SelectFace))
	s.Click(0, 0) // corner pixel ray misses the small box
	if s.Selection().Count() != 0 {
		t.Errorf("corner click selected %d, want 0 (miss)", s.Selection().Count())
	}
}

func TestRayPickerSetCameraAndDirectQuery(t *testing.T) {
	t.Parallel()
	s := extrudedBox(t, 2, 6) // a taller box
	p := NewRayPicker(scene.NewCamera(10, 10), partBodies(s))
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(1, 1, 20)
	cam.Target = math.P3(1, 1, 0)
	p.SetCamera(cam)
	if _, ok := p.Pick(200, 200, NewSelectionFilter(SelectFace)); !ok {
		t.Error("SetCamera not applied / direct Pick missed")
	}
	// Sanity: the kernel query agrees the box is hit from above.
	o, d := cam.RayThrough(200, 200)
	if _, _, ok := query.RayCastFaces(partBodies(s)()[0], o, d, ops.DefaultQuality()); !ok {
		t.Error("kernel RayCastFaces missed the box from above")
	}
}
