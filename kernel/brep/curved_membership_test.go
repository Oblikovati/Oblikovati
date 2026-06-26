// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// pointInCurvedFace (M2 Phase 1, Oblikovati/Oblikovati#1334) must classify points on a sphere cap: the
// lower hemisphere (enclosed by the equator loop) contains its own pole, not the opposite one.

// lowerHemisphereCap builds the curvedFace for a sphere's lower (z<0) cap: the sphere surface with the
// equator as its boundary loop, reversed so the lower cap is the enclosed region (as capSplit emits it).
func lowerHemisphereCap(t *testing.T, r float64) curvedFace {
	t.Helper()
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), r)
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	return curvedFace{
		surface: sphere,
		loops:   []curvedLoop{{edges: []loopEdge{{curve: circle, t0: 1, t1: 0}}}},
	}
}

func TestPointInCurvedFaceSphereCap(t *testing.T) {
	const r = 5
	cap := lowerHemisphereCap(t, r)
	sphere := cap.surface.(geom.Sphere)
	inside := sphere.PointAt(0, -1.2) // a point in the southern (lower) hemisphere
	outside := sphere.PointAt(0, 1.2) // a point in the northern (upper) hemisphere
	if !pointInCurvedFace(cap, inside) {
		t.Errorf("lower-cap point %v classified outside", inside)
	}
	if pointInCurvedFace(cap, outside) {
		t.Errorf("upper-hemisphere point %v classified inside the lower cap", outside)
	}
}

// TestPointInCurvedFaceBoundaryless: a boundary-less face contains every surface point.
func TestPointInCurvedFaceBoundaryless(t *testing.T) {
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), 3)
	f := curvedFace{surface: sphere}
	if !pointInCurvedFace(f, sphere.PointAt(0.7, 0.3)) {
		t.Error("a boundary-less face must contain every point on its surface")
	}
}

// TestPointInCurvedFaceHuggingBoundary is #1407 criterion 3: a point pressed right up against the loop
// boundary — where the boundary passes close to it and the geodesic turning is rapid — is still
// classified correctly, because the winding sum refines adaptively there instead of trusting a fixed
// number of samples that a thin feature could slip between.
func TestPointInCurvedFaceHuggingBoundary(t *testing.T) {
	const r = 5
	cap := lowerHemisphereCap(t, r)
	sphere := cap.surface.(geom.Sphere)
	for _, c := range []struct {
		name string
		v    float64
		want bool
	}{
		{"just inside the equator (lower)", -0.001, true},
		{"just outside the equator (upper)", 0.001, false},
	} {
		p := sphere.PointAt(2.0, c.v) // latitude a hair from the equator, hugging the boundary loop
		if got := pointInCurvedFace(cap, p); got != c.want {
			t.Errorf("%s: point %v classified inside=%v, want %v", c.name, p, got, c.want)
		}
	}
}

// TestEdgeWindingAngleRefinesLargeSubtend covers the adaptive split: an arc that subtends a large angle
// from a nearby point is integrated to the true turning, not under-counted as one coarse step. A
// half-circle seen from its own centre subtends exactly π; one bisected chord step would read far less.
func TestEdgeWindingAngleRefinesLargeSubtend(t *testing.T) {
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 4)
	le := loopEdge{curve: circle, t0: 0, t1: 0.5} // the upper half-circle (t ∈ [0, 0.5] of a full turn)
	normal := math.V3(0, 0, 1)
	got := edgeWindingAngle(le, math.P3(0, 0, 0), normal, le.t0, le.t1, le.curve.PointAt(le.t0), le.curve.PointAt(le.t1), 0)
	if d := got - stdmath.Pi; d < -1e-6 || d > 1e-6 {
		t.Errorf("half-circle winding from the centre = %v, want pi (adaptive refinement under-integrated)", got)
	}
}
