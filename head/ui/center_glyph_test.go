//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// hasVertexNear reports whether any drawn vertex lies within eps of target.
func hasVertexNear(pos []math.Point3, target math.Point3, eps float64) bool {
	for _, p := range pos {
		if float64(p.DistanceTo(target)) <= eps {
			return true
		}
	}
	return false
}

// arcCircleSketch returns a fresh XY sketch (no standalone points, no projections) holding one arc
// and one circle at known centres — the #2159 geometry whose centres must be glyphed.
func arcCircleSketch(t *testing.T, arcCenter, circleCenter math.Point2) *sketch.Sketch {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "centers.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	start := math.P2(arcCenter.X+1, arcCenter.Y)
	end := math.P2(arcCenter.X, arcCenter.Y+1)
	sk.Arcs().AddByCenterStartEnd(arcCenter, start, end, true)
	sk.Circles().AddByCenterRadius(circleCenter, 1)
	return sk
}

// TestPointsOverlayGlyphsArcAndCircleCenters is the #2159 regression: an arc's and a circle's
// centre are real, constrainable sketch points but are not in the typed Points collection, so
// before the fix the overlay drew no glyph for them. An arc's centre sits off the curve in empty
// space, so with no marker the user could not aim a coincident-to-origin at it ("the arc has no
// centre"). Both centres must now carry a target marker.
func TestPointsOverlayGlyphsArcAndCircleCenters(t *testing.T) {
	const h = 0.1
	arcCenter, circleCenter := math.P2(5, 5), math.P2(-3, 2)
	sk := arcCircleSketch(t, arcCenter, circleCenter)

	item, ok := pointsOverlay(sk.Plane(), sk, h)
	if !ok || len(item.Positions) == 0 {
		t.Fatal("pointsOverlay must glyph the arc and circle centres (#2159)")
	}
	if !hasVertexNear(item.Positions, sk.Plane().ToModel(arcCenter), h) {
		t.Errorf("no marker near the arc centre %v — an arc centre must be a hover target", arcCenter)
	}
	if !hasVertexNear(item.Positions, sk.Plane().ToModel(circleCenter), h) {
		t.Errorf("no marker near the circle centre %v", circleCenter)
	}
}

// TestPointsOverlayEmptyWithoutPointsOrCircles guards the discriminator: a sketch with only a line
// (no points, no circular centres) draws no point markers, so the arc/circle case above is what
// adds them, not some always-on behaviour.
func TestPointsOverlayEmptyWithoutPointsOrCircles(t *testing.T) {
	s := app.NewSession()
	pd, _ := compdef.AddPart(s.Workspace(), "line.opd", true)
	sk := pd.Content().(*compdef.PartComponentDefinition).Sketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 1))
	if _, ok := pointsOverlay(sk.Plane(), sk, 0.1); ok {
		t.Error("a line-only sketch must draw no point markers")
	}
}
