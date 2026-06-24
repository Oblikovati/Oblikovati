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
