// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The torus∩quadric closed form (ADR-0061 stage 5). A torus is affine in its azimuth direction, so a
// quadric whose quadratic form is invariant about the torus axis reduces there to one harmonic and the
// section is an arccos. These rows pin the three shapes that reduction takes and the one it refuses.

// testRing is the fixture ring: major radius 5, minor 1.5, about z.
func testRing(t *testing.T) Torus {
	t.Helper()
	ring, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	return ring
}

// assertSectionOnBothSurfaces walks every curve and requires each point to lie on both surfaces, and
// each closed curve to close.
func assertSectionOnBothSurfaces(t *testing.T, name string, curves []Curve3, a, b Surface) {
	t.Helper()
	for i, cv := range curves {
		lo, hi := cv.Domain()
		if d := float64(cv.PointAt(lo).DistanceTo(cv.PointAt(hi))); d > 1e-12 { // tol:numeric — a closed section's own rounding
			t.Errorf("%s curve %d closes with a %.3e gap", name, i, d)
		}
		for k := 0; k <= 400; k++ {
			p := cv.PointAt(lo + (hi-lo)*float64(k)/400)
			offA := stdmath.Abs(float64(SignedDistanceToSurface(a, p)))
			offB := stdmath.Abs(float64(SignedDistanceToSurface(b, p)))
			if offA > 1e-12 || offB > 1e-12 { // tol:numeric — the harmonic solve's own rounding
				t.Fatalf("%s curve %d at %g sits %.3e and %.3e off the two surfaces", name, i, float64(k)/400, offA, offB)
			}
		}
	}
}

// assertTangentsAreRegular requires every curve's analytic tangent to be finite, non-zero and to agree
// with a central difference — the folds included, which is where the cosine reparametrisation earns it.
func assertTangentsAreRegular(t *testing.T, name string, curves []Curve3) {
	t.Helper()
	for i, cv := range curves {
		lo, hi := cv.Domain()
		for k := 0; k <= 128; k++ {
			at := lo + (hi-lo)*float64(k)/128
			tan := cv.TangentAt(at)
			length := float64(tan.Length())
			if stdmath.IsNaN(length) || stdmath.IsInf(length, 0) || length == 0 {
				t.Fatalf("%s curve %d at %g: tangent %v is not a finite non-zero direction", name, i, at, tan)
			}
			a, b := stdmath.Max(lo, at-1e-7), stdmath.Min(hi, at+1e-7)
			fd := cv.PointAt(a).VectorTo(cv.PointAt(b))
			if cos := float64(fd.Dot(tan)) / (float64(fd.Length()) * length); cos < 1-1e-6 { // tol:numeric
				t.Errorf("%s curve %d at %g: the analytic tangent and the difference quotient differ (cos %.9f)", name, i, at, cos)
			}
		}
	}
}

// TestRingAgainstAnAxialDrillIsTwoFoldedLoops: a drill parallel to the ring's axis reaches the tube over
// part of its turn only — around the flanks, not over the top and bottom — so the section is one folded
// loop per window, and there are two: the seam it enters by and the seam it leaves by.
func TestRingAgainstAnAxialDrillIsTwoFoldedLoops(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	drill, _ := NewCylinder(math.P3(5, 0, 0), math.V3(0, 0, 1), 0.8)
	curves, handled := IntersectSurfacesAnalytic(ring, drill, ResolutionForSize(12))
	if !handled || len(curves) != 2 {
		t.Fatalf("handled=%v curves=%d, want two folded loops", handled, len(curves))
	}
	for i, cv := range curves {
		if _, ok := cv.(TorusQuadricLoop); !ok {
			t.Errorf("curve %d is %T, want a TorusQuadricLoop", i, cv)
		}
	}
	assertSectionOnBothSurfaces(t, "axial drill", curves, ring, drill)
	assertTangentsAreRegular(t, "axial drill", curves)
}

// TestRingAgainstABallIsExact drives the ball in two positions: centred on the tube's own centre circle,
// where it reaches the tube at EVERY tube angle and the section is two full-turn arcs, and set off in
// all three coordinates, where it reaches part of the turn and the section is one folded loop.
func TestRingAgainstABallIsExact(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	onTube, _ := NewSphere(math.P3(5, 0, 0), 2)
	curves, handled := IntersectSurfacesAnalytic(ring, onTube, ResolutionForSize(12))
	if !handled || len(curves) != 2 {
		t.Fatalf("ball on the tube centre: handled=%v curves=%d, want two full-turn arcs", handled, len(curves))
	}
	for i, cv := range curves {
		if _, ok := cv.(TorusQuadricArc); !ok {
			t.Errorf("curve %d is %T, want a TorusQuadricArc", i, cv)
		}
	}
	assertSectionOnBothSurfaces(t, "ball on tube", curves, ring, onTube)
	assertTangentsAreRegular(t, "ball on tube", curves)

	general, _ := NewSphere(math.P3(3, 2, 1), 2.5)
	folded, handled := IntersectSurfacesAnalytic(ring, general, ResolutionForSize(12))
	if !handled || len(folded) != 1 {
		t.Fatalf("ball off centre: handled=%v curves=%d, want one folded loop", handled, len(folded))
	}
	assertSectionOnBothSurfaces(t, "ball off centre", folded, ring, general)
	assertTangentsAreRegular(t, "ball off centre", folded)
}

// TestCoaxialQuadricAgainstARingIsCircles: a coaxial cylinder's constraint on the ring carries NO
// azimuth dependence — it is a function of the tube angle alone — so the section is whole circles at the
// tube angles that satisfy it, not two curves. It is the degenerate end of the same reduction, and it
// falls out of asking whether the harmonic has any reach rather than out of asking what the surface is.
func TestCoaxialQuadricAgainstARingIsCircles(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	shaft, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 4)
	curves, handled := IntersectSurfacesAnalytic(ring, shaft, ResolutionForSize(12))
	if !handled || len(curves) != 2 {
		t.Fatalf("coaxial shaft: handled=%v curves=%d, want the two circles at radius 4", handled, len(curves))
	}
	for i, cv := range curves {
		c, ok := cv.(Circle)
		if !ok {
			t.Fatalf("curve %d is %T, want a Circle", i, cv)
		}
		if stdmath.Abs(c.Radius-4) > 1e-9 { // tol:numeric — the level-root bisection's own floor
			t.Errorf("circle %d has radius %.9f, want the shaft's 4", i, c.Radius)
		}
	}
	assertSectionOnBothSurfaces(t, "coaxial shaft", curves, ring, shaft)

	clear, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	empty, handled := IntersectSurfacesAnalytic(ring, clear, ResolutionForSize(12))
	if !handled || len(empty) != 0 {
		t.Errorf("a coaxial shaft inside the ring's hole: handled=%v curves=%d, want the empty ANSWER", handled, len(empty))
	}
}

// TestSkewQuadricAgainstARingDeclines: a rod ACROSS the ring has a quadratic form that is not invariant
// about the ring's axis, so the azimuth dependence is a second harmonic and its roots are a quartic
// rather than an arccos. The form must decline rather than answer, and the gate is on the TENSOR, not
// on the surface's type — a cone coaxial with the ring passes it, the same cone tilted does not.
func TestSkewQuadricAgainstARingDeclines(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1)
	if _, handled := IntersectSurfacesAnalytic(ring, rod, ResolutionForSize(12)); handled {
		t.Error("a rod across the ring must decline: its section is not one harmonic")
	}
	tilted, _ := NewCylinder(math.P3(5, 0, 0), math.V3(0.2, 0, 1), 0.8)
	if _, handled := IntersectSurfacesAnalytic(ring, tilted, ResolutionForSize(12)); handled {
		t.Error("a TILTED drill must decline too — the reduction is gated on the tensor, not on the type")
	}
}

// TestARingAndAFarBallDoNotMeet: a ball clear of the ring is an ANSWER — empty and handled — not a
// refusal. Reporting it as unhandled would send a pair with no section to the marcher.
func TestARingAndAFarBallDoNotMeet(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	far, _ := NewSphere(math.P3(30, 0, 0), 2)
	curves, handled := IntersectSurfacesAnalytic(ring, far, ResolutionForSize(40))
	if !handled || len(curves) != 0 {
		t.Errorf("handled=%v curves=%d, want the empty ANSWER", handled, len(curves))
	}
}
