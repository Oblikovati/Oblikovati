// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Regression for Oblikovati#2077. The self-intersection check is mesh-accurate, and its own
// docstring always promised that a crossing thinner than the faceting error escapes it — but the
// implementation compared nothing against that error. Two TANGENT curved faces are meshed by chords
// that leave the true surface in opposite directions, so their meshes cross by up to the sum of the
// two deviations with no real interpenetration behind them. Every tangent blend in the corpus
// (sphere-on-cone, torus-on-sphere, cylinder-on-sphere, the near-pinch cylinders) therefore reported
// a false self-intersection, which is a large part of why level-2 validation was never adopted.

// wedgeThroughPlane returns a horizontal triangle at z=0 and a vertical one that pierces it by
// exactly depth on each side — a straddling crossing of a known, dialable thickness.
func wedgeThroughPlane(depth float64) ([3]math.Point3, [3]math.Point3) {
	p := math.P3
	flat := [3]math.Point3{p(-10, -10, 0), p(10, -10, 0), p(0, 10, 0)}
	piercing := [3]math.Point3{p(-1, 0, -depth), p(1, 0, -depth), p(0, 0, depth)}
	return flat, piercing
}

// TestCrossingShallowerThanFacetingIsNotReported: the allowance is what the two meshes could have
// produced on their own, so a crossing at or below it carries no evidence and must be dropped.
func TestCrossingShallowerThanFacetingIsNotReported(t *testing.T) {
	flat, piercing := wedgeThroughPlane(0.01)
	bvh := newTriBVH([][3]math.Point3{piercing})
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{flat}, bvh, nil, 1e-9, 0.05); hit {
		t.Error("a 0.01-deep crossing was reported under a 0.05 faceting allowance")
	}
}

// TestCrossingDeeperThanFacetingIsStillReported: the allowance must not blunt the check. The same
// shape, driven deeper than the allowance, is a real interpenetration.
func TestCrossingDeeperThanFacetingIsStillReported(t *testing.T) {
	flat, piercing := wedgeThroughPlane(0.5)
	bvh := newTriBVH([][3]math.Point3{piercing})
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{flat}, bvh, nil, 1e-9, 0.05); !hit {
		t.Error("a 0.5-deep crossing was dropped under a 0.05 faceting allowance")
	}
}

// TestPlanarPairKeepsAZeroAllowance is the reason the allowance is per-face rather than a flat
// multiple of the chord tolerance. A planar face is tessellated EXACTLY, so a plane-on-plane
// crossing of any depth is real — this is the blade-twist defect (#2078), which a flat 2·tol
// allowance would have erased.
func TestPlanarPairKeepsAZeroAllowance(t *testing.T) {
	flat, piercing := wedgeThroughPlane(0.01)
	bvh := newTriBVH([][3]math.Point3{piercing})
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{flat}, bvh, nil, 1e-9, 0); !hit {
		t.Error("a 0.01-deep crossing was dropped even though neither face has any faceting error")
	}
}

// TestCoplanarOverlapSurvivesTheAllowance: a coplanar pair has no depth at all, so gating it on
// depth would erase every coplanar overlap — including the real ones. The allowance applies to the
// straddling branch only.
func TestCoplanarOverlapSurvivesTheAllowance(t *testing.T) {
	p := math.P3
	a := [3]math.Point3{p(0, 0, 0), p(10, 0, 0), p(0, 10, 0)}
	b := [3]math.Point3{p(1, 1, 0), p(9, 1, 0), p(1, 9, 0)} // same plane, large shared area
	bvh := newTriBVH([][3]math.Point3{b})
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{a}, bvh, nil, 1e-9, 1e6); !hit {
		t.Error("a coplanar overlap was erased by a depth allowance that cannot apply to it")
	}
}

// TestCrossingThicknessIsTheShallowerPoke: the measure is how far the crossing goes through, taken
// on the side that goes through LESS. Using the deeper side would let a triangle that merely grazes
// one face but reaches far past the other count as a deep crossing.
func TestCrossingThicknessIsTheShallowerPoke(t *testing.T) {
	p := math.P3
	flat := [3]math.Point3{p(-10, -10, 0), p(10, -10, 0), p(0, 10, 0)}
	lopsided := [3]math.Point3{p(-1, 0, -0.02), p(1, 0, -0.02), p(0, 0, 5)} // 0.02 below, 5 above
	if got := crossingThickness(flat, lopsided); stdmath.Abs(got-0.02) > 1e-12 {
		t.Errorf("crossingThickness = %g, want the shallower side 0.02", got)
	}
}

// TestCrossingThicknessIsSymmetric: which triangle is passed first must not change the verdict,
// or the same face pair would be judged differently depending on face ordering in the body.
func TestCrossingThicknessIsSymmetric(t *testing.T) {
	flat, piercing := wedgeThroughPlane(0.3)
	if a, b := crossingThickness(flat, piercing), crossingThickness(piercing, flat); stdmath.Abs(a-b) > 1e-12 {
		t.Errorf("crossingThickness is not symmetric: %g vs %g", a, b)
	}
}

// TestFacetingAllowanceScalesWithTheModel: the allowance is a length taken from the chord
// tolerance, so it follows the model (ADR-0042) instead of pinning a verdict to centimetres.
func TestFacetingAllowanceScalesWithTheModel(t *testing.T) {
	for _, k := range []float64{1e-3, 1, 1000} {
		flat, piercing := wedgeThroughPlane(0.01 * k)
		bvh := newTriBVH([][3]math.Point3{piercing})
		if _, hit := meshCrossesOffBoundary([][3]math.Point3{flat}, bvh, nil, 1e-12, 0.05*k); hit {
			t.Errorf("at scale %g the same sub-faceting crossing was reported", k)
		}
		if _, hit := meshCrossesOffBoundary([][3]math.Point3{flat}, bvh, nil, 1e-12, 0.001*k); !hit {
			t.Errorf("at scale %g the same real crossing was dropped", k)
		}
	}
}

// TestPlanarFaceGetsNoFacetingAllowance pins the per-face rule that makes the allowance safe: a
// plane is meshed exactly, so it must contribute nothing. If planar faces were given the chord
// tolerance too, plane-on-plane interpenetration up to 2·tol deep would go unreported.
func TestPlanarFaceGetsNoFacetingAllowance(t *testing.T) {
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(5, 5, 5), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := DefaultQuality()
	for _, f := range box.Faces() {
		if got := faceMeshDeviation(f, q); got != 0 {
			t.Fatalf("planar face %d was given a faceting allowance of %g, want 0", f.ID(), got)
		}
	}
}

// TestCurvedFaceGetsTheChordTolerance is the companion: a curved face's chords do leave the true
// surface, by up to the chord tolerance, so that is exactly its allowance.
func TestCurvedFaceGetsTheChordTolerance(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	q := DefaultQuality()
	curved := 0
	for _, f := range cyl.Faces() {
		if _, planar := f.Geometry().(geom.Plane); planar {
			continue
		}
		curved++
		if got := faceMeshDeviation(f, q); got != q.tol() {
			t.Errorf("curved face %d allowance = %g, want the chord tolerance %g", f.ID(), got, q.tol())
		}
	}
	if curved == 0 {
		t.Fatal("the cylinder fixture produced no curved face, so the assertion proved nothing")
	}
}
