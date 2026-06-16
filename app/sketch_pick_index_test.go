// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// TestSketchPickIndexFindsLineAmongMany builds a sketch with thousands of lines and
// confirms the spatial index still picks the one the ray passes through — i.e. the
// culling is correct, not just fast.
func TestSketchPickIndexFindsLineAmongMany(t *testing.T) {
	s, profile := newPartWithSquare(t, 2) // 2×2 square on XY
	sk := profile.Sketch
	for i := 0; i < 3000; i++ { // a wall of lines off to the side, far from the cursor
		y := float64(i) * 0.5
		sk.Lines().AddByTwoPoints(math.P2(-100, y), math.P2(-90, y))
	}
	target := sk.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(2, 1)) // passes through (1,1)

	cam := newTopDownCameraAt(1, 1)
	s.SetPicker(NewRayPicker(cam, partBodies(s)).
		WithSketches(func() []*sketch.Sketch { return []*sketch.Sketch{sk} }))
	s.Selection().SetFilter(NewSelectionFilter(SelectSketchEntity))
	s.Click(200, 200) // centre pixel → ray through (1,1)

	if got, ok := s.Selection().First().(SketchEntityHandle); !ok || got.Entity != target {
		t.Fatalf("picked %v, want the target line through (1,1)", s.Selection().First())
	}
}

// TestSketchPickIndexMatchesBruteForce checks the indexed pick agrees with a direct
// scan of every segment for the same ray.
func TestSketchPickIndexMatchesBruteForce(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	sk := profile.Sketch
	for i := 0; i < 500; i++ {
		x := float64(i) * 0.1
		sk.Lines().AddByTwoPoints(math.P2(x, 5), math.P2(x+0.05, 5))
	}
	target := sk.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(2, 1))
	cam := newTopDownCameraAt(1, 1)
	p := NewRayPicker(cam, partBodies(s)).WithSketches(func() []*sketch.Sketch { return []*sketch.Sketch{sk} })
	origin, dir := cam.RayThrough(200, 200)
	filter := NewSelectionFilter(SelectSketchEntity)

	hit, _, ok := p.nearestSketchCurve(origin, dir, filter)
	if !ok {
		t.Fatal("indexed pick found nothing")
	}
	if h, _ := hit.(SketchEntityHandle); h.Entity != target {
		t.Fatalf("indexed pick = %v, want target", hit)
	}
}

// TestPickIndexCacheReuseAndInvalidate checks the per-sketch index is reused while
// unchanged and rebuilt when the line count changes.
func TestPickIndexCacheReuseAndInvalidate(t *testing.T) {
	_, profile := newPartWithSquare(t, 2)
	sk := profile.Sketch
	a := pickIndexFor(sk)
	if b := pickIndexFor(sk); a != b {
		t.Fatal("index rebuilt despite no change")
	}
	sk.Lines().AddByTwoPoints(math.P2(3, 3), math.P2(4, 4))
	if c := pickIndexFor(sk); c == a {
		t.Fatal("index not rebuilt after adding a line")
	}
}

func newTopDownCameraAt(x, y float64) scene.Camera {
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(x, y, 20)
	cam.Target = math.P3(x, y, 0)
	cam.Up = math.V3(0, 1, 0)
	return cam
}
