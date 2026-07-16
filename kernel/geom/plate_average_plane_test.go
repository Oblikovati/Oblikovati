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
	dom, err := AveragePlane(tiltedPlaneAnchors())
	if err != nil {
		t.Fatalf("AveragePlane: %v", err)
	}
	expected := tiltedPlaneUnitNormal()
	if !dom.N.IsParallelTo(expected, 0) {
		t.Errorf("N = %+v not parallel (to within tol, up to sign) to known plane normal %+v", dom.N, expected)
	}
}

func TestAveragePlaneFrameIsOrthonormalRightHanded(t *testing.T) {
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
	_, err := AveragePlane(collinearAnchors())
	if err == nil {
		t.Fatal("AveragePlane: expected error for collinear anchors, got nil")
	}
	if !strings.Contains(err.Error(), "collinear") {
		t.Errorf("error %q should name the collinear/rank-deficient defect", err.Error())
	}
}

func TestAveragePlaneRejectsTooFewAnchors(t *testing.T) {
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
