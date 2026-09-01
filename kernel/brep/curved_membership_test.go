// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestInwardAtPointsIntoTheKeptRegion pins the side convention the whole classifier rests on: a loop is
// walked with its region on the LEFT seen from outside the surface, so n × T points into it. The equator
// walked CLOCKWISE about +Z (as [lowerHemisphereCap] walks it) keeps the SOUTHERN cap, so its inward
// direction at every station has a negative Z component.
func TestInwardAtPointsIntoTheKeptRegion(t *testing.T) {
	t.Parallel()
	cap := lowerHemisphereCap(t, 5)
	le := cap.loops[0].edges[0]
	for _, u := range []float64{0, 0.17, 0.5, 0.83} {
		inward, ok := inwardAt(cap.surface, le, le.t0+(le.t1-le.t0)*u)
		if !ok {
			t.Fatalf("no inward direction at u=%g of the equator", u)
		}
		if inward.Z >= 0 {
			t.Errorf("inward at u=%g is %v, want a southward (negative Z) direction", u, inward)
		}
	}
}

// TestClosestParamOnEdgeFindsTheGlobalMinimum: the golden-section refinement must land on the true
// closest point of a full circle, not on whichever coarse scan station happened to be nearest. A point
// out along +X sees the circle's +X station as its closest, whatever parameter that station carries.
func TestClosestParamOnEdgeFindsTheGlobalMinimum(t *testing.T) {
	t.Parallel()
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 4)
	le := loopEdge{curve: circle, t0: 0, t1: 1}
	p := math.P3(9, 0, 0)
	got := le.curve.PointAt(le.t0 + (le.t1-le.t0)*closestParamOnEdge(le, p))
	if d := float64(got.DistanceTo(math.P3(4, 0, 0))); d > 1e-6 {
		t.Errorf("closest point on the circle to %v is %v, want (4,0,0) (off by %g)", p, got, d)
	}
}

// TestPointInCurvedFaceAcrossALoopCorner: a query whose closest boundary point is the VERTEX two edges
// share is classified from the blended (pseudonormal) inward direction of both. Splitting the equator in
// two must not change what the lower cap claims, above all right at the split.
func TestPointInCurvedFaceAcrossALoopCorner(t *testing.T) {
	t.Parallel()
	cap := lowerHemisphereCap(t, 5)
	circle := cap.loops[0].edges[0].curve
	cap.loops = []curvedLoop{{edges: []loopEdge{{curve: circle, t0: 1, t1: 0.5}, {curve: circle, t0: 0.5, t1: 0}}}}
	sphere := cap.surface.(geom.Sphere)
	corner, _ := sphere.ParamAt(circle.PointAt(0.5)) // the longitude of the split vertex
	for _, c := range []struct {
		v    float64
		want bool
	}{{-0.05, true}, {0.05, false}, {-1.4, true}, {1.4, false}} {
		p := sphere.PointAt(corner, c.v) // on the vertex's own meridian, so the closest foot IS the corner
		if got := pointInCurvedFace(cap, p); got != c.want {
			t.Errorf("the split-equator lower cap claims %v (v=%g) = %v, want %v", p, c.v, got, c.want)
		}
	}
}
