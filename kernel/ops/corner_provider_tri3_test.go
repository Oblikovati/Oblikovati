// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// sphereTriLoopAdj builds a spherical-triangle loop on Sphere{origin, R}: three quarter great-circle
// arcs meeting at the axis points (R,0,0),(0,R,0),(0,0,R), each side's Adjacent = that one sphere,
// Cont=G1. This exercises the full degenerate-4 + ribbon + pole-fold path directly (the tier walk
// would route a true sphere to analyticSphere first, so tri3 is the fallback we call here explicitly).
func sphereTriLoopAdj(t *testing.T, r float64) RailLoop {
	t.Helper()
	o := math.P3(0, 0, 0)
	sph, err := geom.NewSphere(o, r)
	if err != nil {
		t.Fatal(err)
	}
	mk := func(normal, refDir math.Vector3) Side {
		a, e := geom.NewArc3d(o, normal, refDir, r, 0, stdmath.Pi/2)
		if e != nil {
			t.Fatal(e)
		}
		return Side{Curve: a, Adjacent: sph, Cont: G1}
	}
	return RailLoop{Sides: []Side{
		mk(math.V3(0, 0, 1), math.V3(1, 0, 0)), // (R,0,0)->(0,R,0)
		mk(math.V3(1, 0, 0), math.V3(0, 1, 0)), // (0,R,0)->(0,0,R)
		mk(math.V3(0, 1, 0), math.V3(0, 0, 1)), // (0,0,R)->(R,0,0)
	}}
}

// TestSphereTriLoopAdjIsClosed catches an endpoint-chaining mistake early: the three arcs must chain
// A->B->C->A within a tight absolute tol.
func TestSphereTriLoopAdjIsClosed(t *testing.T) {
	t.Parallel()
	if !sphereTriLoopAdj(t, 6).Closed(1e-9) {
		t.Fatal("sphereTriLoopAdj is not a closed cycle A->B->C->A")
	}
}

// TestTri3FitsAndBuilds proves the provider fills the spherical triangle with a certified,
// tangent-to-sphere degenerate-4 patch that interpolates its three boundary rails. The MaxAngleDev
// gate proves the pole anti-fold did NOT mask a real crease (the fill stays tangent along the arcs).
func TestTri3FitsAndBuilds(t *testing.T) {
	t.Parallel()
	loop := sphereTriLoopAdj(t, 6)
	p := tri3Provider{}
	if !p.Fits(loop) {
		t.Fatal("tri3Provider must Fit a 3-valence loop")
	}
	patch, cert, ok := p.Build(loop, blendScale())
	if !ok {
		t.Fatal("Build returned ok=false on a valid spherical-triangle loop")
	}
	if !cert.Valid(blendScale()) {
		t.Fatalf("certificate invalid: %+v", cert)
	}
	if !cert.NoFold {
		t.Error("patch folds")
	}
	if patch.Kind != BlendKindTri3 {
		t.Errorf("Kind = %q, want %q", patch.Kind, BlendKindTri3)
	}
	if cert.MaxAngleDev >= seamAngularTol {
		t.Errorf("MaxAngleDev = %g, want < %g (fill not tangent to sphere)", cert.MaxAngleDev, seamAngularTol)
	}
	assertTri3InterpolatesRails(t, loop, patch.Surface.(geom.BSplineSurface))
}

// assertTri3InterpolatesRails samples each real rail↔fill-edge pair per the degenerate-4 mapping and
// asserts the fill's boundary tracks the intended sphere arc. Corners are pinned (weld-tight); the arc
// interiors carry the asBSplineCurve rebuild-approximation of a quarter circle, so they are checked to
// track the true arc within a modest rebuild-fit tol (mirrors coons4's assertInterpolatesRails). The
// pole=corner[0]=(R,0,0), so base c0 is s1 (B->C), leg d0 is s0 reversed (B->A), leg d1 is s2 (C->A).
func assertTri3InterpolatesRails(t *testing.T, loop RailLoop, fill geom.BSplineSurface) {
	t.Helper()
	weld := blendScale().Weld()
	rebuildTol := 1e-3 * 6.0 // arc rebuild fit gate (R=6); exact corners stay weld-tight
	checks := []struct {
		e    fillEdge
		side int
		rev  bool
	}{
		{edgeVMin, 1, false}, // base c0: s1 B->C
		{edgeUMin, 0, true},  // leg d0: s0 reversed B->A
		{edgeUMax, 2, false}, // leg d1: s2 C->A
	}
	for _, ck := range checks {
		for _, f := range []float64{0, 0.5, 1} {
			want := sidePointAt(loop.Sides[ck.side].Curve, f, ck.rev)
			u, v := ck.e.fillParam(fill, f)
			got := fill.PointAt(u, v)
			tol := weld
			if f != 0 && f != 1 {
				tol = rebuildTol
			}
			if got.DistanceTo(want) > tol {
				t.Errorf("edge %v f=%g: fill %v vs rail %v (d=%g > %g)", ck.e, f, got, want, got.DistanceTo(want), tol)
			}
		}
	}
}

// TestTri3RejectsValence4 proves the provider declines a 4-side loop (its Fit gate is the valence).
func TestTri3RejectsValence4(t *testing.T) {
	t.Parallel()
	loop := quarterCylLoop(t, 8)
	if (tri3Provider{}).Fits(loop) {
		t.Fatal("tri3Provider must NOT Fit a 4-valence loop")
	}
}

// TestTri3BuildGuardsValence proves Build self-guards on loop.Valence() (mirrors coons4Provider.Build
// and analyticSphereProvider.Build): calling Build directly (skipping Fits, as a misbehaving caller
// might) on a Valence-2 loop must honest-reject rather than panic inside choosePole/cornerNormalAgreement,
// which indexes loop.Sides[0..2] and would go out of range on fewer than 3 sides.
func TestTri3BuildGuardsValence(t *testing.T) {
	t.Parallel()
	loop := RailLoop{Sides: []Side{{}, {}}}
	_, _, ok := (tri3Provider{}).Build(loop, blendScale())
	if ok {
		t.Fatal("Build must return ok=false for a Valence-2 loop, not attempt to fill it")
	}
}

// TestTri3TwistedRejects flips one rail's orientation so the loop no longer chains; Build must NOT
// panic and must honest-reject (ok==false OR an invalid certificate) at the pole rather than ship a
// self-intersecting patch.
func TestTri3TwistedRejects(t *testing.T) {
	t.Parallel()
	loop := sphereTriLoopAdj(t, 6)
	o := math.P3(0, 0, 0)
	rev, err := geom.NewArc3d(o, math.V3(-1, 0, 0), math.V3(0, 0, 1), 6, 0, stdmath.Pi/2) // (0,0,R)->(0,R,0)
	if err != nil {
		t.Fatal(err)
	}
	loop.Sides[1].Curve = rev // breaks the A->B->C->A chain: side1 now starts at C, not B
	_, cert, ok := (tri3Provider{}).Build(loop, blendScale())
	if ok && cert.Valid(blendScale()) {
		t.Fatalf("expected honest-reject on a twisted loop, got a valid certificate: %+v", cert)
	}
}
