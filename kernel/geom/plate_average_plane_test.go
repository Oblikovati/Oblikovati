// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// tiltedPlaneAnchors returns 6 points that satisfy z = x+y exactly — a plane tilted off every
// coordinate axis, so a passing test can't be an accident of an axis-aligned special case. The
// set varies x and y independently, spanning two distinct in-plane directions, which is what
// AveragePlane needs to resolve a well-defined normal (M6 P1 brief, Step 1).
func tiltedPlaneAnchors() []math.Point3 {
	return []math.Point3{
		math.P3(0, 0, 0),
		math.P3(2, 0, 2),
		math.P3(0, 3, 3),
		math.P3(2, 3, 5),
		math.P3(1, 1, 2),
		math.P3(3, 1, 4),
	}
}

// tiltedPlaneUnitNormal is the analytic unit normal of z=x+y (implicit form x+y-z=0), the
// ground truth tiltedPlaneAnchors was built from — the oracle AveragePlane's fitted N is
// checked against, not our own output.
func tiltedPlaneUnitNormal() math.Vector3 {
	n := math.V3(1, 1, -1)
	return n.Scale(1 / n.Length())
}

// collinearAnchors are 4 points on a straight line: only one direction carries spread, so no
// second in-plane axis exists and no plane is well-defined. AveragePlane must reject this
// rather than silently emit an arbitrary normal.
func collinearAnchors() []math.Point3 {
	return []math.Point3{
		math.P3(0, 0, 0),
		math.P3(1, 0, 0),
		math.P3(2, 0, 0),
		math.P3(3, 0, 0),
	}
}

func TestAveragePlaneMatchesKnownNormal(t *testing.T) {
	t.Parallel()
	dom, err := AveragePlane(tiltedPlaneAnchors())
	if err != nil {
		t.Fatalf("AveragePlane: %v", err)
	}
	expected := tiltedPlaneUnitNormal()
	if !dom.N.IsParallelTo(expected, 0) {
		t.Errorf("N = %+v not parallel (to within tol, up to sign) to known plane normal %+v", dom.N, expected)
	}
}

// wallScaleTiltedPlaneAnchors is tiltedPlaneAnchors scaled x1e4 — wall/room-scale coordinates,
// not toy 0-3 units. Regression coverage for the Jacobi convergence floor (M6 P1 review): a
// BARE absolute residual floor is unreachable by scatter-matrix entries at this scale (~1e8,
// since entries carry units of length²), so jacobiEigen3 must scale its convergence check by
// the matrix's own Frobenius norm to still resolve the correct normal here.
func wallScaleTiltedPlaneAnchors() []math.Point3 {
	const scale = 1e4
	base := tiltedPlaneAnchors()
	scaled := make([]math.Point3, len(base))
	for i, p := range base {
		scaled[i] = math.P3(p.X*scale, p.Y*scale, p.Z*scale)
	}
	return scaled
}

func TestAveragePlaneMatchesKnownNormalAtWallScale(t *testing.T) {
	t.Parallel()
	dom, err := AveragePlane(wallScaleTiltedPlaneAnchors())
	if err != nil {
		t.Fatalf("AveragePlane: %v", err)
	}
	expected := tiltedPlaneUnitNormal()
	if !dom.N.IsParallelTo(expected, 0) {
		t.Errorf("N = %+v not parallel (to within tol, up to sign) to known plane normal %+v", dom.N, expected)
	}
}

func TestAveragePlaneFrameIsOrthonormalRightHanded(t *testing.T) {
	t.Parallel()
	dom, err := AveragePlane(tiltedPlaneAnchors())
	if err != nil {
		t.Fatalf("AveragePlane: %v", err)
	}
	const unitTol = 1e-9 // tol:numeric — length-1 / orthogonality check on a dimensionless basis
	if d := dom.U.Length(); mathAbs(d-1) > unitTol {
		t.Errorf("U not unit length: %v", d)
	}
	if d := dom.V.Length(); mathAbs(d-1) > unitTol {
		t.Errorf("V not unit length: %v", d)
	}
	if d := dom.U.Dot(dom.V); mathAbs(d) > unitTol {
		t.Errorf("U,V not orthogonal: U·V = %v", d)
	}
	if cross := dom.U.Cross(dom.V); !cross.IsEqualTo(dom.N, unitTol) {
		t.Errorf("frame not right-handed: U×V = %+v, want N = %+v", cross, dom.N)
	}
}

func TestAveragePlaneRoundTripsInPlanePoints(t *testing.T) {
	t.Parallel()
	dom, err := AveragePlane(tiltedPlaneAnchors())
	if err != nil {
		t.Fatalf("AveragePlane: %v", err)
	}
	for _, p := range tiltedPlaneAnchors() {
		u, v := dom.Project(p)
		if got := dom.Lift(u, v, 0); !got.IsEqualTo(p, 0) {
			t.Errorf("round trip for %+v: got %+v", p, got)
		}
	}
}

func TestAveragePlaneRoundTripsOffPlanePoint(t *testing.T) {
	t.Parallel()
	dom, err := AveragePlane(tiltedPlaneAnchors())
	if err != nil {
		t.Fatalf("AveragePlane: %v", err)
	}
	inPlane := tiltedPlaneAnchors()[0]
	off := inPlane.TranslateBy(tiltedPlaneUnitNormal().Scale(5))
	u, v := dom.Project(off)
	w := dom.Origin.VectorTo(off).Dot(dom.N) // the signed height Project() itself drops
	if got := dom.Lift(u, v, w); !got.IsEqualTo(off, 0) {
		t.Errorf("off-plane round trip: got %+v, want %+v", got, off)
	}
}

func TestAveragePlaneRejectsCollinearAnchors(t *testing.T) {
	t.Parallel()
	_, err := AveragePlane(collinearAnchors())
	if err == nil {
		t.Fatal("AveragePlane: expected error for collinear anchors, got nil")
	}
	if !strings.Contains(err.Error(), "collinear") {
		t.Errorf("error %q should name the collinear/rank-deficient defect", err.Error())
	}
}

// TestPlaneFrameFromEigenAcceptsHealthySecondSpread and
// TestPlaneFrameFromEigenRejectsWhenSecondSpreadCollapses call planeFrameFromEigen directly
// with synthetic eigenvalue triples, bypassing jacobiEigen3/AveragePlane entirely. This closes
// a review gap (M6 P1): tiltedPlaneAnchors' real sqrt(values[lo])/majorSpread ratio is ~4.8e-9
// — just ABOVE domainDegenerateTol (1e-9) — so a guard mistakenly comparing values[lo] instead
// of values[mid] would STILL pass on that fixture, and the rank-guard test couldn't prove
// planeFrameFromEigen reads the SECOND-largest eigenvalue (values[mid]) rather than the
// smallest (values[lo] — which is *supposed* to be ~0 for any valid planar fit; that's the
// normal direction, not evidence of degeneracy). These two triples pin values[lo] at an
// unambiguous ~0 (1e-20, far below the 1e-9 floor by itself) so a mid->lo swap changes the
// outcome: the healthy triple below is accepted by the correct (mid) guard and WRONGLY
// rejected by a lo-based guard, since values[lo] <= values[mid] always holds (sortEigenIndices
// invariant) and a valid planar fit's lo is always ~0 — the guard direction that discriminates
// is "buggy over-rejects", not "buggy under-rejects".
func TestPlaneFrameFromEigenAcceptsHealthySecondSpread(t *testing.T) {
	t.Parallel()
	values := [3]float64{1e-20, 5, 10} // lo, mid, hi: two healthy in-plane spreads (mid, hi)
	_, _, _, err := planeFrameFromEigen(values, identity3(), ResolutionForSize(1))
	if err != nil {
		t.Fatalf("planeFrameFromEigen: expected accept for healthy second spread, got %v", err)
	}
}

func TestPlaneFrameFromEigenRejectsWhenSecondSpreadCollapses(t *testing.T) {
	t.Parallel()
	values := [3]float64{1e-20, 1e-20, 10} // lo, mid both ~0: only one in-plane spread (hi)
	_, _, _, err := planeFrameFromEigen(values, identity3(), ResolutionForSize(1))
	if err == nil {
		t.Fatal("planeFrameFromEigen: expected reject when second-largest spread collapses, got nil")
	}
	if !strings.Contains(err.Error(), "collinear") {
		t.Errorf("error %q should name the collinear/rank-deficient defect", err.Error())
	}
}

func TestAveragePlaneRejectsTooFewAnchors(t *testing.T) {
	t.Parallel()
	_, err := AveragePlane([]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)})
	if err == nil {
		t.Fatal("AveragePlane: expected error for <3 anchors, got nil")
	}
}

// mathAbs avoids importing the stdlib math package under a second name purely for one
// absolute-value call in this test file (kernel/geom already imports "oblikovati.org/math"
// as the unqualified "math").
func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
