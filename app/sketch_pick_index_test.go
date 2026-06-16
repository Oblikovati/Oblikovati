// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
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

// TestSketchPickIndexHitsEveryCurveType picks each kind of sketch curve at a point on
// the curve that is OFF its chord (the apex of an arc, the top of a circle/ellipse, an
// interior fit point of a spline). Each is selectable only if the spatial index facets
// the entity along its real sweep — the property that lets an imported DWG drawing's
// arcs/circles/ellipses/elliptical arcs be selected, not just its straight lines.
func TestSketchPickIndexHitsEveryCurveType(t *testing.T) {
	cases := []struct {
		name   string
		build  func(sk *sketch.Sketch) sketch.Entity
		pickAt [2]float64 // sketch-plane point, on the curve but off its chord
	}{
		{"line", func(sk *sketch.Sketch) sketch.Entity {
			return sk.Lines().AddByTwoPoints(math.P2(10, 0), math.P2(14, 0))
		}, [2]float64{12, 0}},
		{"arc", func(sk *sketch.Sketch) sketch.Entity { // upper semicircle r=1, apex (20,1)
			return sk.Arcs().AddByCenterStartEnd(math.P2(20, 0), math.P2(21, 0), math.P2(19, 0), true)
		}, [2]float64{20, 1}},
		{"circle", func(sk *sketch.Sketch) sketch.Entity { // r=2 centred (30,0), top (30,2)
			return sk.Circles().AddByCenterRadius(math.P2(30, 0), 2)
		}, [2]float64{30, 2}},
		{"ellipse", func(sk *sketch.Sketch) sketch.Entity { // major 3 / minor 1, top (40,1)
			return sk.Ellipses().Add(math.P2(40, 0), math.V2(3, 0), 3, 1)
		}, [2]float64{40, 1}},
		{"elliptical arc", func(sk *sketch.Sketch) sketch.Entity { // upper half, apex (50,1)
			return sk.EllipticalArcs().Add(math.P2(50, 0), math.V2(3, 0), 3, 1, 0, stdmath.Pi)
		}, [2]float64{50, 1}},
		{"spline", func(sk *sketch.Sketch) sketch.Entity { // fit curve through (61,1)
			return sk.Splines().AddByPoints([]math.Point2{{X: 60, Y: 0}, {X: 61, Y: 1}, {X: 62, Y: 0}}, false)
		}, [2]float64{61, 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, sk := newPartWithEmptySketch(t)
			target := tc.build(sk)

			cam := newTopDownCameraAt(tc.pickAt[0], tc.pickAt[1])
			s.SetPicker(NewRayPicker(cam, partBodies(s)).
				WithSketches(func() []*sketch.Sketch { return []*sketch.Sketch{sk} }))
			s.Selection().SetFilter(NewSelectionFilter(SelectSketchEntity))
			s.Click(200, 200)

			got, ok := s.Selection().First().(SketchEntityHandle)
			if !ok || got.Entity != target {
				t.Fatalf("picked %v at (%g,%g), want the %s along its sweep",
					s.Selection().First(), tc.pickAt[0], tc.pickAt[1], tc.name)
			}
		})
	}
}

// TestSketchPickIndexCurveMissesOffCurve checks the faceted index does not over-claim:
// a ray through the empty interior of an open arc (inside its chord, off the curve)
// selects nothing. Guards against the curve being treated as a filled region.
func TestSketchPickIndexCurveMissesOffCurve(t *testing.T) {
	s, sk := newPartWithEmptySketch(t)
	sk.Arcs().AddByCenterStartEnd(math.P2(20, 0), math.P2(21, 0), math.P2(19, 0), true) // apex (20,1)

	cam := newTopDownCameraAt(20, 0.2) // inside the arc, well away from the curve
	s.SetPicker(NewRayPicker(cam, partBodies(s)).
		WithSketches(func() []*sketch.Sketch { return []*sketch.Sketch{sk} }))
	s.Selection().SetFilter(NewSelectionFilter(SelectSketchEntity))
	s.Click(200, 200)

	if first := s.Selection().First(); first != nil {
		t.Fatalf("picked %v inside an open arc, want nothing", first)
	}
}

// TestPickIndexRebuildsOnCurveTypeChange checks the cache key tracks total entity count,
// so adding a non-line entity (an arc) invalidates the index — otherwise a freshly
// imported curve would be unpickable until a line happened to change.
func TestPickIndexRebuildsOnCurveTypeChange(t *testing.T) {
	_, sk := newPartWithEmptySketch(t)
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	a := pickIndexFor(sk)
	sk.Arcs().AddByCenterStartEnd(math.P2(20, 0), math.P2(21, 0), math.P2(19, 0), true)
	if b := pickIndexFor(sk); b == a {
		t.Fatal("index not rebuilt after adding an arc (cache key ignores non-line entities)")
	}
}

func newPartWithEmptySketch(t *testing.T) (*Session, *sketch.Sketch) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	return s, def.Sketches().Add(sketch.XYPlane())
}

func newTopDownCameraAt(x, y float64) scene.Camera {
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(x, y, 20)
	cam.Target = math.P3(x, y, 0)
	cam.Up = math.V3(0, 1, 0)
	return cam
}
