// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestMarcherCentreCurvePlaneCylinder checks the heart of method (B): between a top plane and a
// coaxial cylinder (radius 2), the rolling-ball (r=0.5) centre curve is the EXACT circle of radius
// R−r=1.5 at height z_top−r=2.5, and the blend surface is the torus (major 1.5, minor 0.5) tangent
// to both supports — the curved-neighbour blend our analytic catalog could not route (#1797/#1806).
func TestMarcherCentreCurvePlaneCylinder(t *testing.T) {
	plane, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	inside := func(p math.Point3) bool { // the solid: inside the cylinder AND below the cap
		return float64(p.Z) < 3 && stdmath.Hypot(float64(p.X), float64(p.Y)) < 2
	}
	m := &Marcher{Inside: inside, Res: geom.ResolutionForSize(6)}

	centre, ok := m.centreCurve(plane, cyl, 0.5)
	if !ok {
		t.Fatal("no centre curve found for plane+cylinder rolling ball")
	}
	circ, isCircle := centre.(geom.Circle)
	if !isCircle {
		t.Fatalf("centre curve is %T, want a circle", centre)
	}
	if stdmath.Abs(circ.Radius-1.5) > 1e-9 || stdmath.Abs(float64(circ.Center.Z)-2.5) > 1e-9 {
		t.Errorf("centre circle radius=%g centreZ=%g, want 1.5 / 2.5", circ.Radius, circ.Center.Z)
	}

	surf, ok := analyticBlendSurface(centre, 0.5)
	if !ok {
		t.Fatal("no analytic blend surface for a circular centre curve")
	}
	tor, isTorus := surf.(geom.Torus)
	if !isTorus {
		t.Fatalf("blend surface is %T, want a torus", surf)
	}
	if stdmath.Abs(tor.MajorRadius-1.5) > 1e-9 || stdmath.Abs(tor.MinorRadius-0.5) > 1e-9 {
		t.Errorf("torus major=%g minor=%g, want 1.5 / 0.5", tor.MajorRadius, tor.MinorRadius)
	}
	// tangency: the torus tube touches the cap plane at z=3 and the cylinder wall at radius 2.
	assertBlendTangent(t, tor, 3.0, 2.0)
}

// assertBlendTangent samples the torus tube's extreme point on each principal direction: its top
// must reach the cap plane and its outer equator the cylinder radius.
func assertBlendTangent(t *testing.T, tor geom.Torus, capZ, cylR float64) {
	t.Helper()
	top := tor.MajorRadius*0 + float64(tor.Center.Z) + tor.MinorRadius
	if stdmath.Abs(top-capZ) > 1e-9 {
		t.Errorf("torus top z=%g, want tangent to cap at %g", top, capZ)
	}
	if outer := tor.MajorRadius + tor.MinorRadius; stdmath.Abs(outer-cylR) > 1e-9 {
		t.Errorf("torus outer radius=%g, want tangent to cylinder at %g", outer, cylR)
	}
}

// TestFitCanalMatchesTorusOracle forces the general B-spline canal fit on the plane+cylinder case
// (whose exact blend is the torus) and checks the fitted surface lies on that torus within a fit
// tolerance — validating approx.go against an analytic oracle for the non-primitive-centre path.
func TestFitCanalMatchesTorusOracle(t *testing.T) {
	plane, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	inside := func(p math.Point3) bool {
		return float64(p.Z) < 3 && stdmath.Hypot(float64(p.X), float64(p.Y)) < 2
	}
	m := &Marcher{Inside: inside, Res: geom.ResolutionForSize(6)}
	centre, _ := m.centreCurve(plane, cyl, 0.5)
	lo, hi := centre.Domain()
	// A real segment spans a bounded sub-arc (a quarter-cylinder rim is 90°); the full 360° circle
	// as one bicubic is under-resolved. Fit the first quarter, as a bounded edge would.
	t1 := lo + 0.25*(hi-lo)

	surf, st := fitCanal(centre, plane, cyl, 0.5, lo, t1, 1e-9, inside)
	if st != StatusOk {
		t.Fatalf("fitCanal status = %v", st)
	}
	const major, minor = 1.5, 0.5 // the oracle torus about (0,0,2.5), axis z
	maxErr := 0.0
	for iu := 0; iu <= 8; iu++ {
		for iv := 0; iv <= 6; iv++ {
			p := surf.PointAt(float64(iu)/8, float64(iv)/6)
			rho := stdmath.Hypot(float64(p.X), float64(p.Y)) - major
			d := stdmath.Hypot(rho, float64(p.Z)-2.5) - minor // signed distance to the torus tube
			if a := stdmath.Abs(d); a > maxErr {
				maxErr = a
			}
		}
	}
	if maxErr > 5e-3 {
		t.Errorf("fitted canal deviates from the torus oracle by %g, want < 5e-3", maxErr)
	}
}
