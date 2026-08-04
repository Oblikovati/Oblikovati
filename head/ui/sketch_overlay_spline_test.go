//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2026: the overlay drew a spline by joining its defining points, i.e. the CONTROL POLYGON,
// while the model held (and extruded) a real NURBS curve. The sketch therefore contradicted the
// solid built from it, and users reasonably concluded "spline is not spline - it's polyline".

// splineSketch returns a sketch holding one 4-point fit spline with real curvature.
func splineSketch() (*sketch.Sketch, *sketch.Spline, []math.Point2) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	defining := []math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(2, -2), math.P2(3, 0)}
	return sk, sk.Splines().AddByPoints(defining, false), defining
}

// TestSplineOverlayDrawsTheCurveNotItsDefiningPoints is the headline regression: a 4-point
// spline drew as 3 chords, so its vertex count matched the defining points exactly.
func TestSplineOverlayDrawsTheCurveNotItsDefiningPoints(t *testing.T) {
	sk, sp, defining := splineSketch()
	got := vertexCount(sketchOverlay(sk, nil, nil, false))

	curve, _ := sketch.EntityPolyline(sp)
	if len(curve) <= len(defining) {
		t.Fatalf("test premise broken: the faceted curve has %d points, not more than the %d defining ones",
			len(curve), len(defining))
	}
	if got <= len(defining) {
		t.Errorf("the overlay emitted %d vertices for a %d-point spline — it is drawing the control polygon, not the curve (the curve facets to %d points)",
			got, len(defining), len(curve))
	}
}

// TestSplineOverlayFollowsTheModelledCurve pins the stronger property: what is drawn matches the
// same faceting region detection and picking use, so the sketch cannot disagree with the solid.
func TestSplineOverlayFollowsTheModelledCurve(t *testing.T) {
	sk, sp, _ := splineSketch()
	curve, _ := sketch.EntityPolyline(sp)

	// One draw item per lane; the spline is the only entity, so its vertices are all of them.
	if got, want := vertexCount(sketchOverlay(sk, nil, nil, false)), len(curve); got != want {
		t.Errorf("overlay drew %d vertices, want %d — the same faceting the model uses", got, want)
	}
}
