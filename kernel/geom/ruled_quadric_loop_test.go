// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The folded ruled∩quadric window (ADR-0061 stage 5). A ball off a cylinder's axis is the smallest
// pair whose section folds: the ruling meets the ball over part of the sweep only, so the two roots
// of the ruling quadratic meet at the window's ends and the section is ONE closed loop.

// offAxisBallOnCylinder is the fixture: a cylinder of radius 3 up the z axis, and a ball of radius 2
// centred 4 out on x — so the ball straddles the wall, reaching 2 inside it and 6 outside.
func offAxisBallOnCylinder(t *testing.T) (Sphere, Cylinder) {
	t.Helper()
	sph, err := NewSphere(math.P3(4, 0, 0), 2)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	return sph, cyl
}

// TestFoldedWindowIsOneClosedLoopOnBothSurfaces: the section comes back as a single closed loop whose
// every point lies on both surfaces to rounding — the exactness claim the whole form rests on.
func TestFoldedWindowIsOneClosedLoopOnBothSurfaces(t *testing.T) {
	t.Parallel()
	sph, cyl := offAxisBallOnCylinder(t)
	curves, handled := IntersectSurfacesAnalytic(sph, cyl, ResolutionForSize(10))
	if !handled || len(curves) != 1 {
		t.Fatalf("handled=%v curves=%d, want one closed loop", handled, len(curves))
	}
	loop, ok := curves[0].(RuledQuadricLoop)
	if !ok {
		t.Fatalf("section is %T, want a RuledQuadricLoop", curves[0])
	}
	if d := float64(loop.PointAt(0).DistanceTo(loop.PointAt(1))); d != 0 {
		t.Errorf("the loop closes with a %g gap; the two halves must meet AT the fold, exactly", d)
	}
	for k := 0; k <= 512; k++ {
		p := loop.PointAt(float64(k) / 512)
		offSph := stdmath.Abs(float64(SignedDistanceToSurface(sph, p)))
		offCyl := stdmath.Abs(float64(SignedDistanceToSurface(cyl, p)))
		if offSph > 1e-12 || offCyl > 1e-12 { // tol:numeric — the quadratic solve's own rounding
			t.Fatalf("t=%g sits %.3e off the sphere and %.3e off the cylinder", float64(k)/512, offSph, offCyl)
		}
	}
}

// TestFoldedWindowTangentIsRegularThroughTheFolds is why the loop is parametrised by a cosine rather
// than by the azimuth: dv/du is INFINITE at a fold, and the cosine's speed vanishes there at exactly
// the rate that cancels it. The analytic tangent must therefore be finite, non-zero and agree with a
// central difference everywhere — the folds (t = 0 and t = 0.5) above all.
func TestFoldedWindowTangentIsRegularThroughTheFolds(t *testing.T) {
	t.Parallel()
	sph, cyl := offAxisBallOnCylinder(t)
	curves, _ := IntersectSurfacesAnalytic(sph, cyl, ResolutionForSize(10))
	loop := curves[0]
	for k := 0; k <= 64; k++ {
		at := float64(k) / 64
		tan := loop.TangentAt(at)
		length := float64(tan.Length())
		if stdmath.IsNaN(length) || stdmath.IsInf(length, 0) || length == 0 {
			t.Fatalf("t=%g: tangent %v is not a finite non-zero direction", at, tan)
		}
		lo, hi := stdmath.Max(0, at-1e-6), stdmath.Min(1, at+1e-6)
		fd := loop.PointAt(lo).VectorTo(loop.PointAt(hi)).Scale(math.Scalar(1 / (hi - lo)))
		cos := float64(fd.Dot(tan)) / (float64(fd.Length()) * length)
		if cos < 1-1e-6 { // tol:numeric — a central difference against the closed-form derivative
			t.Errorf("t=%g: the analytic tangent and the difference quotient differ (cos %.9f)", at, cos)
		}
	}
}

// TestFoldedWindowDeclinesNothingItCanAnswer pins the three answers the window form owes that are NOT
// a loop: a pair wholly clear, a pair wholly enclosed, and a tangency. Each is "they do not cross",
// which is an ANSWER — reporting it as unhandled would send a pair with no section to the marcher.
func TestFoldedWindowDeclinesNothingItCanAnswer(t *testing.T) {
	t.Parallel()
	_, cyl := offAxisBallOnCylinder(t)
	for _, c := range []struct {
		name   string
		centre math.Point3
		radius float64
	}{
		{"clear of the wall", math.P3(20, 0, 0), 2},
		{"inside the wall", math.P3(0, 0, 0), 1},
		{"tangent to the wall", math.P3(5, 0, 0), 2},
	} {
		sph, err := NewSphere(c.centre, c.radius)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		curves, handled := IntersectSurfacesAnalytic(sph, cyl, ResolutionForSize(30))
		if !handled || len(curves) != 0 {
			t.Errorf("%s: handled=%v curves=%d, want the empty ANSWER", c.name, handled, len(curves))
		}
	}
}

// TestFoldedWindowLeavesTheWrapFormAlone: a section that wraps every azimuth on either chart is the
// arc form's, and must keep coming back as one. The window form claims a pair only when BOTH charts
// fold, so a coaxial ball (two circles) and two crossing cylinders (two arcs) are untouched.
func TestFoldedWindowLeavesTheWrapFormAlone(t *testing.T) {
	t.Parallel()
	_, cyl := offAxisBallOnCylinder(t)
	coax, _ := NewSphere(math.P3(0, 0, 0), 5)
	curves, handled := IntersectSurfacesAnalytic(coax, cyl, ResolutionForSize(12))
	if !handled || len(curves) != 2 {
		t.Fatalf("coaxial ball: handled=%v curves=%d, want two circles", handled, len(curves))
	}
	for i, cv := range curves {
		if _, isCircle := cv.(Circle); !isCircle {
			t.Errorf("coaxial ball section %d is %T, want a Circle", i, cv)
		}
	}
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1.5)
	crossing, handled := IntersectSurfacesAnalytic(rod, cyl, ResolutionForSize(12))
	if !handled || len(crossing) != 2 {
		t.Fatalf("crossing cylinders: handled=%v curves=%d, want two arcs", handled, len(crossing))
	}
	for i, cv := range crossing {
		if _, isArc := cv.(RuledQuadricArc); !isArc {
			t.Errorf("crossing section %d is %T, want a RuledQuadricArc", i, cv)
		}
	}
}

// TestFoldedWindowSpansBothFoldAzimuths reads the WINDOW rather than the curve, which is what catches
// a fold bisection that converged on the wrong bracket. A cylinder's ruling at azimuth u stands at
// distance √(D² + R² − 2DR·cos u) from a ball centred D out from the axis, so it reaches a ball of
// radius r exactly while cos u ≥ (D² + R² − r²)/(2DR) — the law of cosines on the triangle
// (axis, ruling, ball centre), and the window is twice that arccosine.
func TestFoldedWindowSpansBothFoldAzimuths(t *testing.T) {
	t.Parallel()
	sph, cyl := offAxisBallOnCylinder(t)
	curves, _ := IntersectSurfacesAnalytic(sph, cyl, ResolutionForSize(10))
	loop := curves[0].(RuledQuadricLoop)
	const centreDist, wallRadius, ballRadius = 4.0, 3.0, 2.0
	cosHalf := (centreDist*centreDist + wallRadius*wallRadius - ballRadius*ballRadius) / (2 * centreDist * wallRadius)
	want := 2 * stdmath.Acos(cosHalf)
	if got := loop.U1 - loop.U0; stdmath.Abs(got-want) > 1e-9 { // tol:numeric — the fold bisection's own floor
		t.Errorf("the window spans %.9f rad, want %.9f", got, want)
	}
}
