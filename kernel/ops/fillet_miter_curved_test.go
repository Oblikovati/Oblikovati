// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// fakeMiterCorner is a named fixture holding one curved-miter corner's derived geometry (the exact
// torus + cylinder arm surfaces, the corner-ball centre, and the two endpoint targets) read from the
// curved-miter-seam-derivation.md measured fixtures. It lets the equidistance regression exercise the
// seam MATH on raw geometry without importing STEP.
type fakeMiterCorner struct {
	name   string
	arms   curvedMiterArms
	center math.Point3
	vertex math.Point3 // the corner vertex (branch bias anchor)
	r      float64
}

// p5MiterCorner is OCCT blend/simple P5 (curved SHARED cylinder miter): the top-rim arc arm is a
// torus C(50,50,145)/major 45/minor 5, the vertical line arm an r=5 cylinder about the ρ=45 offset
// ruling (65, 7.5736, ·). Corner ball centre (65, 7.5736, 145). Values are [validated] in the
// derivation to be equidistant = r from both spines to machine precision.
func p5MiterCorner(t *testing.T) fakeMiterCorner {
	y := 50 - stdmath.Sqrt(1800) // ρ=R−r=45 offset ruling of the equal-parallel cyl∩cyl arm (derivation D3)
	tor, err := geom.NewTorusWithRef(math.P3(50, 50, 145), math.V3(0, 0, 1), math.V3(1, 0, 0), 45, 5)
	if err != nil {
		t.Fatalf("P5 torus arm: %v", err)
	}
	cyl, err := geom.NewCylinderWithRef(math.P3(65, y, 145), math.V3(0, 0, 1), math.V3(0, -1, 0), 5)
	if err != nil {
		t.Fatalf("P5 cylinder arm: %v", err)
	}
	return fakeMiterCorner{
		name: "P5", arms: curvedMiterArms{tor: tor, cyl: cyl}, r: 5,
		center: math.P3(65, y, 145), vertex: math.P3(65, 2.303, 150),
	}
}

// w4MiterCorner is OCCT blend/simple W4 (planar-shared / curved-OUTER cylinder miter): the arc arm is
// a torus C(3,0.2,0.9999)/major 0.8/minor 0.2 about the −ŷ axis, the line arm an r=0.2 cylinder about
// (·,0.2,0.2). Corner ball centre ≈ (2.987, 0.2, 0.2).
func w4MiterCorner(t *testing.T) fakeMiterCorner {
	tor, err := geom.NewTorusWithRef(math.P3(3, 0.2, 0.9999), math.V3(0, -1, 0), math.V3(0, 0, 1), 0.8, 0.2)
	if err != nil {
		t.Fatalf("W4 torus arm: %v", err)
	}
	cyl, err := geom.NewCylinderWithRef(math.P3(2.987, 0.2, 0.2), math.V3(1, 0, 0), math.V3(0, 0, 1), 0.2)
	if err != nil {
		t.Fatalf("W4 cylinder arm: %v", err)
	}
	return fakeMiterCorner{
		name: "W4", arms: curvedMiterArms{tor: tor, cyl: cyl}, r: 0.2,
		center: math.P3(2.987, 0.2, 0.2), vertex: math.P3(2.9859, 0, 0),
	}
}

// TestCurvedMiterSeamEquidistant is the derivation's load-bearing check: every sampled seam point is
// equidistant = r from BOTH arm spines to machine precision (the equal-r bisector holds). P5 samples
// pin 5.0/5.0, W4 0.2/0.2.
func TestCurvedMiterSeamEquidistant(t *testing.T) {
	for _, fx := range []fakeMiterCorner{p5MiterCorner(t), w4MiterCorner(t)} {
		res := ResolutionForPoints([]math.Point3{fx.center, fx.vertex})
		center, ok := miterCornerBallCenter(fx.arms, fx.vertex, res)
		if !ok {
			t.Fatalf("%s: corner ball centre declined", fx.name)
		}
		n, ok := seamEndpointNormal(fx.arms, center)
		if !ok {
			t.Fatalf("%s: seam endpoint normal declined", fx.name)
		}
		e1 := center.TranslateBy(n.AsVector().Scale(fx.r))
		e2 := center.TranslateBy(n.AsVector().Scale(-fx.r))
		seam, ok := walkCurvedSeam(fx.arms, fx.r, center, e1, e2, fx.vertex, res)
		if !ok {
			t.Fatalf("%s: seam sampling declined", fx.name)
		}
		if len(seam) < 5 {
			t.Fatalf("%s: seam has only %d points (want >=5)", fx.name, len(seam))
		}
		tol := 1e-9 * fx.r
		for i, p := range seam {
			m1, ok := armBallCenter(fx.arms.tor, p)
			if !ok {
				t.Fatalf("%s point %d: torus spine undefined", fx.name, i)
			}
			m2 := cylinderBallCenter(fx.arms.cyl, p)
			d1 := stdmath.Abs(float64(m1.DistanceTo(p)) - fx.r)
			d2 := stdmath.Abs(float64(m2.DistanceTo(p)) - fx.r)
			if d1 > tol || d2 > tol {
				t.Fatalf("%s point %d %v: dist-to-spine err torus=%.3g cyl=%.3g (tol %.3g)", fx.name, i, p, d1, d2, tol)
			}
		}
	}
}
