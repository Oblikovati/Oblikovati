// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The conformance invariant (ADR-0054/#2167): whenever two circular curves ARE the same
// circle — same centre, axis line, radius — they must discretize to the identical point
// set, no matter how each stored its RefDir or normal sign. These tests pin that property
// at the source, independent of the boolean that consumes it.

const conformTol = 1e-12

func unit(x, y, z float64) math.UnitVector3 {
	u, err := math.UnitVector3FromVector(math.V3(x, y, z))
	if err != nil {
		panic(err)
	}
	return u
}

// TestCircleSegmentsCanonical checks the segment count is a multiple of 4, at least 8, and
// grows as the tolerance tightens — and, above all, is a pure function of radius+quality
// (never of RefDir), so two coincident circles get the same count.
func TestCircleSegmentsCanonical(t *testing.T) {
	if n := CircleSegments(3, 0.05, 10*stdmath.Pi/180); n%4 != 0 || n < 8 {
		t.Fatalf("CircleSegments = %d, want a multiple of 4 ≥ 8", n)
	}
	coarse := CircleSegments(3, 0.05, 10*stdmath.Pi/180)
	fine := CircleSegments(3, 1e-3, 1*stdmath.Pi/180)
	if fine <= coarse {
		t.Fatalf("finer quality gave %d segments, not more than the coarse %d", fine, coarse)
	}
}

// TestCircleConformsAcrossRefDir is the core invariant: two circles with identical centre,
// normal, radius, and SEAM but DIFFERENT RefDir sample to identical points — RefDir must
// not perturb the discretization.
func TestCircleConformsAcrossRefDir(t *testing.T) {
	seam := math.P3(3.5, 2, 3)
	a := Circle{Center: math.P3(1, 2, 3), Normal: unit(0, 0, 1), RefDir: unit(1, 0, 0), Radius: 2.5}
	b := Circle{Center: math.P3(1, 2, 3), Normal: unit(0, 0, 1), RefDir: unit(0, 1, 0), Radius: 2.5}
	pa, _ := CircleConformalSamples(a, seam, 1e-3, 1*stdmath.Pi/180)
	pb, _ := CircleConformalSamples(b, seam, 1e-3, 1*stdmath.Pi/180)
	assertSamePoints(t, "RefDir-invariant circle", pa, pb)
}

// TestCircleConformsAcrossNormalSign checks a circle and its normal-flipped twin (the
// stacked-solids case: a lower solid's +z rim and an upper solid's −z rim are one circle)
// sample to the identical point set when anchored on the same seam.
func TestCircleConformsAcrossNormalSign(t *testing.T) {
	seam := math.P3(3, 0, 6)
	a := Circle{Center: math.P3(0, 0, 6), Normal: unit(0, 0, 1), RefDir: unit(1, 0, 0), Radius: 3}
	b := Circle{Center: math.P3(0, 0, 6), Normal: unit(0, 0, -1), RefDir: unit(0, 1, 0), Radius: 3}
	pa, _ := CircleConformalSamples(a, seam, 0.05, 10*stdmath.Pi/180)
	pb, _ := CircleConformalSamples(b, seam, 0.05, 10*stdmath.Pi/180)
	assertSamePoints(t, "normal-sign-invariant circle", pa, pb)
}

// TestCircleInteriorConformsAcrossSeam is the real cross-operand case: two coincident
// circles anchored on DIFFERENT seams still share every interior (non-seam) sample, so the
// two boolean operands conform on the shared rim regardless of where each seam falls.
func TestCircleInteriorConformsAcrossSeam(t *testing.T) {
	c := Circle{Center: math.P3(0, 0, 6), Normal: unit(0, 0, 1), RefDir: unit(1, 0, 0), Radius: 3}
	pa, _ := CircleConformalSamples(c, math.P3(3, 0, 6), 0.05, 10*stdmath.Pi/180)  // seam at angle 0
	pb, _ := CircleConformalSamples(c, math.P3(-3, 0, 6), 0.05, 10*stdmath.Pi/180) // seam at angle π
	for i := 1; i < len(pa)-1; i++ {                                               // interior of A (drop its seam ends) must all appear in B
		if !hasPoint(pb, pa[i]) {
			t.Fatalf("interior sample %d (%v) missing from the other-seam circle — not conformal", i, pa[i])
		}
	}
}

// TestArcInteriorSubsetOfCircle is the #2167 shape: a major arc coaxial with a full circle
// (the cylinder rim and the D-prism arc are the same circle) has EVERY interior point
// coincident with a full-circle sample, so the two conform along the shared arc. The arc's
// own endpoints are extra (a chord meets it), which the boolean's co-refinement imprints.
func TestArcInteriorSubsetOfCircle(t *testing.T) {
	const r, theta = 3.0, 0.6
	circle := Circle{Center: math.P3(0, 0, 6), Normal: unit(0, 0, 1), RefDir: unit(1, 0, 0), Radius: r}
	arc, err := NewArc3d(math.P3(0, 0, 6), math.V3(0, 0, 1), math.V3(1, 0, 0), r, theta, 2*stdmath.Pi-2*theta)
	if err != nil {
		t.Fatal(err)
	}
	cPts, _ := CircleConformalSamples(circle, math.P3(r, 0, 6), 0.05, 10*stdmath.Pi/180)
	aPts, aParams := ArcConformalSamples(arc, 0.05, 10*stdmath.Pi/180)
	if len(aPts) < 3 {
		t.Fatalf("arc sampled to %d points, want ≥3", len(aPts))
	}
	// Endpoints are the arc's own, at ±theta.
	if got := aPts[0]; got.DistanceTo(arc.PointAt(0)) > conformTol {
		t.Fatalf("arc[0] = %v, want the arc start %v", got, arc.PointAt(0))
	}
	if got := aPts[len(aPts)-1]; got.DistanceTo(arc.PointAt(1)) > conformTol {
		t.Fatalf("arc[last] = %v, want the arc end %v", got, arc.PointAt(1))
	}
	if aParams[0] != 0 || aParams[len(aParams)-1] != 1 {
		t.Fatalf("arc params run %g..%g, want 0..1", aParams[0], aParams[len(aParams)-1])
	}
	// Every INTERIOR arc point must coincide with a full-circle sample.
	for i := 1; i < len(aPts)-1; i++ {
		if !hasPoint(cPts, aPts[i]) {
			t.Fatalf("arc interior point %d (%v) has no matching circle sample — not conformal", i, aPts[i])
		}
	}
}

// assertSamePoints fails unless a and b are the same length and pairwise coincident.
func assertSamePoints(t *testing.T, name string, a, b []math.Point3) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("%s: %d vs %d points", name, len(a), len(b))
	}
	for i := range a {
		if a[i].DistanceTo(b[i]) > conformTol {
			t.Fatalf("%s: point %d differs: %v vs %v", name, i, a[i], b[i])
		}
	}
}

// hasPoint reports whether pts contains a point within conformTol of p.
func hasPoint(pts []math.Point3, p math.Point3) bool {
	for _, q := range pts {
		if q.DistanceTo(p) <= conformTol {
			return true
		}
	}
	return false
}
