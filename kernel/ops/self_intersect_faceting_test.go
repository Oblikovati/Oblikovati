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

// TestCoplanarOverlapIsJudgedByAreaNotDepth: a coplanar pair has no depth at all, so a DEPTH gate
// would read zero and erase every coplanar overlap, real ones included. The allowance still applies
// — as a length on the overlap's side, against its area — so a substantial overlap survives.
func TestCoplanarOverlapIsJudgedByAreaNotDepth(t *testing.T) {
	p := math.P3
	a := [3]math.Point3{p(0, 0, 0), p(10, 0, 0), p(0, 10, 0)}
	b := [3]math.Point3{p(1, 1, 0), p(9, 1, 0), p(1, 9, 0)} // same plane, ~32 of shared area
	bvh := newTriBVH([][3]math.Point3{b})
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{a}, bvh, nil, 1e-9, 0.05); !hit {
		t.Error("a 32-area coplanar overlap was erased by a 0.05 faceting allowance")
	}
}

// TestSliverCoplanarOverlapIsDiscarded is the #2077 residue. The ratio filter compares the overlap
// to the smaller TRIANGLE, so a sliver passes it on an overlap of no consequence — a torus
// tessellated against a plane it touches produced overlaps down to 1e-16 cm2 that way. An overlap
// that does not span the faceting allowance on a side is noise.
func TestSliverCoplanarOverlapIsDiscarded(t *testing.T) {
	p := math.P3
	// Two long, extremely thin coplanar slivers: each is 10 long and 1e-6 across, so the overlap is
	// ~1e-5 of area while being a large FRACTION of either sliver.
	a := [3]math.Point3{p(0, 0, 0), p(10, 0, 0), p(0, 1e-6, 0)}
	b := [3]math.Point3{p(0.5, 0, 0), p(9.5, 0, 0), p(0.5, 1e-6, 0)}
	bvh := newTriBVH([][3]math.Point3{b})
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{a}, bvh, nil, 1e-12, 0.05); hit {
		t.Error("a sliver overlap far below the faceting allowance was reported")
	}
	// With no allowance — two PLANAR faces, tessellated exactly — the same overlap is real.
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{a}, bvh, nil, 1e-12, 0); !hit {
		t.Error("an exact coplanar overlap was dropped even though neither face has faceting error")
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

// coplanarPairOfArea returns two coplanar triangles sharing approximately the given area, scaled
// about the origin by k.
func coplanarPairOfArea(area, k float64) ([3]math.Point3, [3]math.Point3) {
	p := math.P3
	h := stdmath.Sqrt(2 * area) // a right isoceles overlap of legs h has area h²/2
	a := [3]math.Point3{p(0, 0, 0), p(10*k, 0, 0), p(0, 10*k, 0)}
	b := [3]math.Point3{p(0, 0, 0), p(h*k, 0, 0), p(0, h*k, 0)}
	return a, b
}

// TestCoplanarAllowanceIsAnAreaNotALength pins the DIMENSION of the coplanar gate. The allowance is
// a length (a faceting deviation), so an area must be compared against its square. Comparing the
// area against the raw length instead happens to agree on the fixtures at either extreme, and only
// an overlap BETWEEN the two thresholds can tell them apart: at allow = 0.05 the square is 0.0025,
// so an overlap of 0.01 is real under the correct rule and noise under the dimensionally wrong one.
func TestCoplanarAllowanceIsAnAreaNotALength(t *testing.T) {
	a, b := coplanarPairOfArea(0.01, 1)
	bvh := newTriBVH([][3]math.Point3{b})
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{a}, bvh, nil, 1e-12, 0.05); !hit {
		t.Error("an overlap of 0.01 was discarded under a 0.05 allowance, whose square is only 0.0025")
	}
}

// TestCoplanarAllowanceIsScaleFree is the same point as a property. Areas grow as k² and the
// allowance as k, so area vs allow² holds its verdict at every scale while area vs allow does not
// (ADR-0042).
func TestCoplanarAllowanceIsScaleFree(t *testing.T) {
	for _, k := range []float64{1e-2, 1, 1e2} {
		real, noise := 0.01*k*k, 1e-6*k*k
		for _, c := range []struct {
			area float64
			want bool
		}{{real, true}, {noise, false}} {
			a, b := coplanarPairOfArea(c.area/(k*k), k)
			bvh := newTriBVH([][3]math.Point3{b})
			if _, hit := meshCrossesOffBoundary([][3]math.Point3{a}, bvh, nil, 1e-12*k, 0.05*k); hit != c.want {
				t.Errorf("at scale %g an overlap of area %g reported %v, want %v", k, c.area, hit, c.want)
			}
		}
	}
}

// TestPlaneAxesAreOrthonormal is the guard for the scale bug #2075 left behind in this branch. The
// axes used to be the raw cross products n×ref and n×(n×ref), whose magnitudes are |n| and |n|², so
// the projected 2D space was scaled by the triangle's own area — and by a DIFFERENT factor on each
// axis. Every length and area measured there was meaningless, including selfIntersectEps.
func TestPlaneAxesAreOrthonormal(t *testing.T) {
	for _, n := range []math.Vector3{
		math.V3(0, 0, 1), math.V3(0, 0, 1e-6), math.V3(0, 0, 1e6), math.V3(3, -4, 12),
	} {
		u, v := planeAxes(n)
		lu, lv := float64(u.Length()), float64(v.Length())
		if stdmath.Abs(lu-1) > 1e-12 || stdmath.Abs(lv-1) > 1e-12 {
			t.Errorf("planeAxes(%v) lengths = %g, %g, want 1 and 1", n, lu, lv)
		}
		if d := stdmath.Abs(float64(u.Dot(v))); d > 1e-12 {
			t.Errorf("planeAxes(%v) axes are not perpendicular: u·v = %g", n, d)
		}
		if d := stdmath.Abs(float64(u.Dot(n))) + stdmath.Abs(float64(v.Dot(n))); d > 1e-9*float64(n.Length()) {
			t.Errorf("planeAxes(%v) axes do not lie in the plane: |u·n|+|v·n| = %g", n, d)
		}
	}
}

// TestCoplanarAreaIsMeasuredInTrueUnits: the whole point of the orthonormal axes. The shared area
// two coplanar triangles report must be the area they actually share, whatever the normal's
// magnitude — which is proportional to the triangle's area and so varies wildly across a mesh.
func TestCoplanarAreaIsMeasuredInTrueUnits(t *testing.T) {
	p := math.P3
	big := [3]math.Point3{p(0, 0, 0), p(10, 0, 0), p(0, 10, 0)}
	small := [3]math.Point3{p(0, 0, 0), p(2, 0, 0), p(0, 2, 0)} // area 2, wholly inside big
	n, _ := triPlaneEq(small)
	_, area, hit := coplanarOverlap(big, small, n)
	if !hit {
		t.Fatal("a triangle wholly inside another did not report a coplanar overlap")
	}
	if stdmath.Abs(area-2) > 1e-9 {
		t.Errorf("shared area = %g, want 2 (the small triangle's own area)", area)
	}
}
