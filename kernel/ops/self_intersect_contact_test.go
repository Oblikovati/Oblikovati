// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// Regression for Oblikovati#2075. The Möller test compared a fixed 1e-9 against quantities nobody
// had normalised: the plane distances were n·p−d (inflated by |n| ≈ 2·area) and the overlap
// interval was projected on the unnormalised n1×n2 (inflated by |n1||n2|). For large triangles the
// inflation is several orders of magnitude, so an overlap far too short to be a crossing still
// cleared the threshold — and two faces that merely TOUCH along a line report as interpenetrating.
// That is what made a mitered sheet-metal corner invalid: the miter-gap cut face abuts the wall it
// cuts, sharing exactly the line y=3 and no area at all.

// touchingPair returns two large triangles whose crossing intervals on their planes' intersection
// line meet over only 1e-10 of true length — contact, not a crossing. The triangles are big
// (~30 units) precisely because that is what the old unnormalised test amplified.
func touchingPair() ([3]math.Point3, [3]math.Point3) {
	p := math.P3
	inPlaneZ := [3]math.Point3{p(0, -10, 0), p(0, 10, 0), p(30, 10, 0)} // crosses y=0 over x ∈ [0, 15]
	const touch = 15 - 1e-10
	inPlaneY := [3]math.Point3{p(touch, 0, -10), p(touch, 0, 10), p(40, 0, 10)} // crosses z=0 from x = touch
	return inPlaneZ, inPlaneY
}

// TestFacesMeetingAlongALineAreContactNotInterpenetration is the #2075 shape in the small.
func TestFacesMeetingAlongALineAreContactNotInterpenetration(t *testing.T) {
	a, b := touchingPair()
	if w, hit := trianglesIntersect(a, b); hit {
		t.Errorf("triangles sharing 1e-10 of their intersection line report interpenetration at %v", w)
	}
}

// TestGenuineCrossingIsStillReported: the fix must not blunt the check. The same pair, moved so the
// intervals genuinely overlap by half a unit, is a real crossing.
func TestGenuineCrossingIsStillReported(t *testing.T) {
	p := math.P3
	a := [3]math.Point3{p(0, -10, 0), p(0, 10, 0), p(30, 10, 0)} // crosses y=0 over x ∈ [0, 15]
	b := [3]math.Point3{p(10, 0, -10), p(10, 0, 10), p(40, 0, 10)}
	w, hit := trianglesIntersect(a, b)
	if !hit {
		t.Fatal("a genuine crossing was missed")
	}
	if x := float64(w.X); x < 10 || x > 15 {
		t.Errorf("witness %v is outside the overlapping run x ∈ [10, 15]", w)
	}
}

// TestCrossingVerdictIsIndependentOfModelScale: the same shape, built ten times bigger, must be
// judged the same way. An unnormalised threshold fails exactly here — it is why the sheet-metal
// corner (millimetres) and the fixtures (centimetres) disagreed about the same geometry.
func TestCrossingVerdictIsIndependentOfModelScale(t *testing.T) {
	scaleTri := func(t [3]math.Point3, k float64) [3]math.Point3 {
		var out [3]math.Point3
		for i, v := range t {
			out[i] = math.P3(float64(v.X)*k, float64(v.Y)*k, float64(v.Z)*k)
		}
		return out
	}
	p := math.P3
	contactA, contactB := touchingPair()
	crossA := [3]math.Point3{p(0, -10, 0), p(0, 10, 0), p(30, 10, 0)}
	crossB := [3]math.Point3{p(10, 0, -10), p(10, 0, 10), p(40, 0, 10)}
	// The range runs to 1e-6 deliberately. Both normalisations claim scale invariance, and only the
	// small end can disprove it: leaving the plane distances unnormalised deflates them by |n| ≈ 2·area,
	// which shrinks as the square of the model, so a genuine crossing on a small part stops
	// registering as straddling at all and the interpenetration is silently missed.
	for _, k := range []float64{1e-6, 1e-3, 0.1, 1, 10, 1000} {
		if w, hit := trianglesIntersect(scaleTri(contactA, k), scaleTri(contactB, k)); hit {
			t.Errorf("at scale %g the same contact reports interpenetration at %v", k, w)
		}
		if _, hit := trianglesIntersect(scaleTri(crossA, k), scaleTri(crossB, k)); !hit {
			t.Errorf("at scale %g a genuine crossing was missed", k)
		}
	}
}

// TestCrossRatioHasAPlateau justifies crossRatio. The verdict must hold across many orders of
// magnitude, or the constant is tuned to the fixture rather than derived from the problem.
//
// The plateau is bounded by the fixture itself, and both bounds are real rather than chosen: below
// ~2.8e-12 (the contact's 1e-10 over a ~36-unit triangle) no ratio can call the contact anything
// but a crossing, and above ~1.4e-2 (the crossing's 0.5 over the same triangle) none can call the
// crossing anything but contact. Between them the answer is stable over seven orders of magnitude,
// and crossRatio = 1e-9 sits in the middle of that span.
func TestCrossRatioHasAPlateau(t *testing.T) {
	p := math.P3
	a := [3]math.Point3{p(0, -10, 0), p(0, 10, 0), p(30, 10, 0)}
	crossing := [3]math.Point3{p(10, 0, -10), p(10, 0, 10), p(40, 0, 10)}
	contact, _ := func() ([3]math.Point3, [3]math.Point3) { x, y := touchingPair(); return y, x }()
	for _, ratio := range []float64{1e-11, 1e-10, 1e-9, 1e-8, 1e-6, 1e-4} {
		if overlapExceeds(a, contact, ratio) {
			t.Errorf("at ratio %g, line contact was read as a crossing", ratio)
		}
		if !overlapExceeds(a, crossing, ratio) {
			t.Errorf("at ratio %g, a real crossing was read as contact", ratio)
		}
	}
}

// overlapExceeds re-runs the interval step at a chosen ratio, so the sweep tests the threshold
// rather than a re-implementation of it.
func overlapExceeds(t1, t2 [3]math.Point3, ratio float64) bool {
	eps := ratio * minTriScale(t1, t2)
	n2, d2 := triPlaneEq(t2)
	s1, ok1 := signedDistances(t1, n2, d2, eps)
	n1, d1 := triPlaneEq(t1)
	s2, ok2 := signedDistances(t2, n1, d1, eps)
	if !ok1 || !ok2 {
		return false
	}
	line, err := math.UnitVector3FromVector(n1.Cross(n2))
	if err != nil {
		return false
	}
	_, hit := intervalOverlap(t1, s1, t2, s2, line.AsVector(), eps)
	return hit
}

func minTriScale(t1, t2 [3]math.Point3) float64 {
	if a, b := triScale(t1), triScale(t2); a < b {
		return a
	} else {
		return b
	}
}
