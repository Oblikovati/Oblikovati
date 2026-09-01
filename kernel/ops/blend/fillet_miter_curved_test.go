// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
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
// torus C(50,50,145)/major 45/minor 5, the vertical line arm an r=5 cylinder about the CONCAVE–CONVEX
// offset ruling (48.333, 5.031, ·) — shared bore ρ=R−r=45 ∩ outer boss ρ=R+r=55, the DRAWEXE-verified
// branch (curved-miter-closure-derivation.md §2), NOT the disproven symmetric R−r∩R−r ruling
// (65, 7.5736). Corner ball centre (48.333, 5.031, 145). The seam samples are [validated] equidistant =
// r from both spines to machine precision.
func p5MiterCorner(t *testing.T) fakeMiterCorner {
	ax, ay := p5CylArmAxisXY()
	tor, err := geom.NewTorusWithRef(math.P3(50, 50, 145), math.V3(0, 0, 1), math.V3(1, 0, 0), 45, 5)
	if err != nil {
		t.Fatalf("P5 torus arm: %v", err)
	}
	cyl, err := geom.NewCylinderWithRef(math.P3(ax, ay, 145), math.V3(0, 0, 1), math.V3(0, -1, 0), 5)
	if err != nil {
		t.Fatalf("P5 cylinder arm: %v", err)
	}
	return fakeMiterCorner{
		name: "P5", arms: curvedMiterArms{tor: tor, cyl: cyl}, r: 5,
		center: math.P3(ax, ay, 145), vertex: math.P3(65, 2.303, 150),
	}
}

// p5CylArmAxisXY is the (x,y) of P5's equal-parallel cyl∧cyl arm axis: the concave–convex branch of the
// shared bore offset (50,50)/R−r=45 ∩ outer boss offset (80,50)/R+r=55 — the DRAWEXE-verified ruling
// (48.3333, 5.0309), 45 from the shared axis and 55 from the outer axis (curved-miter-closure §2).
func p5CylArmAxisXY() (float64, float64) {
	sep := 30.0
	a := (sep*sep + 45*45 - 55*55) / (2 * sep) // signed foot distance from the shared axis toward the outer
	return 50 + a, 50 - stdmath.Sqrt(45*45-a*a)
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

// TestEqualParallelArmBranchDiscriminator pins the wrong-branch fix (curved-miter-closure §2): the
// equal-parallel cyl∧cyl arm axis is the CONCAVE–CONVEX branch (shared bore R−r=45 ∩ outer boss
// R+r=55 → (48.333,5.031)), NOT the symmetric R−r∩R−r branch (65,7.574) the |axis−sharedAxis|=45 test
// cannot rule out. The discriminator is the OUTER offset radius. intersectCoplanarCircles is the shared
// solver; here it proves the two branches diverge and that the physical (low-y, edge-side) ruling is the
// concave–convex one.
func TestEqualParallelArmBranchDiscriminator(t *testing.T) {
	t.Parallel()
	res := opstol.ForPoints([]math.Point3{math.P3(50, 50, 0), math.P3(80, 50, 0)})
	axis := math.V3(0, 0, 1)
	shared, outer := math.P3(50, 50, 0), math.P3(80, 50, 0)
	right0, right1, ok := intersectCoplanarCircles(shared, 45, outer, 55, axis, res) // R−r ∩ R+r
	if !ok {
		t.Fatal("concave–convex offset circles did not meet")
	}
	right := nearerRulingXY(right0, right1, 2.303) // the edge sits at low y
	assertXY(t, "concave–convex arm axis", right, 145.0/3.0, 5.030874, 1e-4)
	wrong0, wrong1, ok := intersectCoplanarCircles(shared, 45, outer, 45, axis, res) // the disproven R−r ∩ R−r
	if !ok {
		t.Fatal("symmetric offset circles did not meet")
	}
	wrong := nearerRulingXY(wrong0, wrong1, 2.303)
	assertXY(t, "disproven symmetric axis", wrong, 65, 50-stdmath.Sqrt(1800), 1e-4)
	if wrong.DistanceTo(right) < 5 {
		t.Fatalf("branches must diverge: concave–convex %v vs symmetric %v", right, wrong)
	}
}

// TestCurvedMiterSeamBottomOnTorusOuterHost pins the seam-BOTTOM fix (curved-miter-closure §1b): sBot is
// NOT the second mutual-tangency point, it is the torus∩cylinder point on the torus arm's OUTER host
// plane — the torus outer-contact circle (major circle pushed minor onto z=150, i.e. (50,50,150)/R45) ∩
// the cyl arm at z=150 ((48.333,5.031,150)/r5), taking the crossing nearer the corner vertex. DRAWEXE:
// sBot=(53.332,5.124,150) on the top-plane contact circle R−r=45.
func TestCurvedMiterSeamBottomOnTorusOuterHost(t *testing.T) {
	t.Parallel()
	ax, ay := p5CylArmAxisXY()
	res := opstol.ForPoints([]math.Point3{math.P3(50, 50, 150), math.P3(ax, ay, 150)})
	p0, p1, ok := intersectCoplanarCircles(math.P3(50, 50, 150), 45, math.P3(ax, ay, 150), 5, math.V3(0, 0, 1), res)
	if !ok {
		t.Fatal("torus outer-contact circle ∩ cyl-at-plane did not meet")
	}
	vp := math.P3(65, 2.303, 150)
	sBot := p0
	if p1.DistanceTo(vp) < p0.DistanceTo(vp) {
		sBot = p1
	}
	assertXY(t, "sBot", sBot, 53.332474, 5.123563, 1e-3)
	if stdmath.Abs(float64(sBot.Z)-150) > 1e-9 {
		t.Fatalf("sBot must lie on the top plane z=150, got z=%.6f", sBot.Z)
	}
	rho := stdmath.Hypot(float64(sBot.X)-50, float64(sBot.Y)-50) // must be on the R−r=45 contact circle
	if stdmath.Abs(rho-45) > 1e-3 {
		t.Fatalf("sBot must lie on the top-plane contact circle R−r=45, got ρ=%.5f", rho)
	}
}

// nearerRulingXY returns whichever of two coplanar points has the smaller |y−wantY| — the P5 edge sits at
// low y, so this selects the physical ruling the way nearerRuling(edge,…) does on the real topology.
func nearerRulingXY(p0, p1 math.Point3, wantY float64) math.Point3 {
	if stdmath.Abs(float64(p0.Y)-wantY) <= stdmath.Abs(float64(p1.Y)-wantY) {
		return p0
	}
	return p1
}

// assertXY fails unless p's (x,y) match (x,y) within tol.
func assertXY(t *testing.T, what string, p math.Point3, x, y, tol float64) {
	t.Helper()
	if stdmath.Abs(float64(p.X)-x) > tol || stdmath.Abs(float64(p.Y)-y) > tol {
		t.Fatalf("%s: got (%.6f,%.6f), want (%.6f,%.6f) tol %g", what, p.X, p.Y, x, y, tol)
	}
}

// TestCurvedMiterSeamEquidistant is the derivation's load-bearing check: every sampled seam point is
// equidistant = r from BOTH arm spines to machine precision (the equal-r bisector holds). P5 samples
// pin 5.0/5.0, W4 0.2/0.2.
func TestCurvedMiterSeamEquidistant(t *testing.T) {
	t.Parallel()
	for _, fx := range []fakeMiterCorner{p5MiterCorner(t), w4MiterCorner(t)} {
		res := opstol.ForPoints([]math.Point3{fx.center, fx.vertex})
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
		seam, ok := walkCurvedSeam(fx.arms, fx.r, center, e1, e2, res)
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
