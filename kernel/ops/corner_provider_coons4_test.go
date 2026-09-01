// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

// quarterCylLoop builds a real quarter-cylinder patch loop (axis Z, radius R, z∈[0,H], θ∈[0,90°]):
// two arcs (z=0, z=H) plus two generator lines, ALL four Adjacent = that one cylinder, all Cont=G1.
// Corners chain in loop order A(R,0,0)→B(0,R,0)→C(0,R,H)→D(R,0,H)→A, so RailLoop.Closed holds.
func quarterCylLoop(t *testing.T, height float64) RailLoop {
	t.Helper()
	r, h := 5.0, height
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatal(err)
	}
	arcBottom, e0 := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, 0, stdmath.Pi/2) // (R,0,0)->(0,R,0)
	arcTop, e1 := geom.NewArc3d(math.P3(0, 0, h), math.V3(0, 0, -1), math.V3(0, 1, 0), r, 0, stdmath.Pi/2)   // (0,R,H)->(R,0,H)
	if e0 != nil || e1 != nil {
		t.Fatalf("arc: %v %v", e0, e1)
	}
	line := func(a, b math.Point3) geom.Curve3 { return geom.LineSegment{StartPoint: a, EndPoint: b} }
	s := func(c geom.Curve3) Side { return Side{Curve: c, Adjacent: cyl, Cont: G1} }
	return RailLoop{Sides: []Side{
		s(arcBottom), // s0: A(R,0,0) -> B(0,R,0)
		s(line(math.P3(0, r, 0), math.P3(0, r, h))), // s1: B -> C(0,R,H)
		s(arcTop), // s2: C(0,R,H) -> D(R,0,H)
		s(line(math.P3(r, 0, h), math.P3(r, 0, 0))), // s3: D -> A
	}}
}

// TestQuarterCylLoopIsClosed catches any endpoint-chaining mistake early: all four corners must
// chain A→B→C→D→A within a tight absolute tol.
func TestQuarterCylLoopIsClosed(t *testing.T) {
	t.Parallel()
	if !quarterCylLoop(t, 8).Closed(1e-9) {
		t.Fatal("quarterCylLoop is not a closed cycle A->B->C->D->A")
	}
}

// TestCoons4FitsAndBuilds proves the provider fills the quarter-cylinder loop with a certified,
// tangent-to-cylinder patch that interpolates its four boundary rails.
func TestCoons4FitsAndBuilds(t *testing.T) {
	t.Parallel()
	loop := quarterCylLoop(t, 8)
	p := coons4Provider{}
	if !p.Fits(loop) {
		t.Fatal("coons4Provider must Fit a 4-valence loop")
	}
	patch, cert, ok := p.Build(loop, blendScale())
	if !ok {
		t.Fatal("Build returned ok=false on a valid quarter-cylinder loop")
	}
	if !cert.Valid(blendScale()) {
		t.Fatalf("certificate invalid: %+v", cert)
	}
	if !cert.NoFold {
		t.Error("patch folds")
	}
	if patch.Kind != BlendKindCoons4 {
		t.Errorf("Kind = %q, want %q", patch.Kind, BlendKindCoons4)
	}
	if cert.MaxAngleDev >= tessellate.SeamAngularTol {
		t.Errorf("MaxAngleDev = %g, want < %g (fill not tangent to cylinder)", cert.MaxAngleDev, tessellate.SeamAngularTol)
	}
	assertInterpolatesRails(t, loop, patch.Surface.(geom.BSplineSurface))
}

// assertInterpolatesRails samples each rail↔fill-edge pair per the Side→Coons mapping and asserts the
// fill's boundary tracks the intended cylinder rail. The exact corners are pinned (weld-tight); the
// LINE sides are exact rails (weld-tight); the ARC sides carry the asBSplineCurve rebuild-approximation
// of a quarter circle, so their interiors are checked to lie ON the cylinder within a modest fit tol.
func assertInterpolatesRails(t *testing.T, loop RailLoop, fill geom.BSplineSurface) {
	t.Helper()
	weld := blendScale().Weld()
	cornerCheck := []struct {
		e    fillEdge
		side int
		rev  bool
		line bool
	}{
		{edgeVMin, 0, false, false}, // arc s0 A->B
		{edgeVMax, 2, true, false},  // arc s2 reversed -> D->C
		{edgeUMin, 3, true, true},   // line s3 reversed -> A->D
		{edgeUMax, 1, false, true},  // line s1 B->C
	}
	for _, ck := range cornerCheck {
		for _, f := range []float64{0, 0.5, 1} {
			want := sidePointAt(loop.Sides[ck.side].Curve, f, ck.rev)
			u, v := ck.e.fillParam(fill, f)
			got := fill.PointAt(u, v)
			tol := weld
			if !ck.line && f != 0 && f != 1 {
				tol = 1e-3 * 5.0 // arc rebuild fit gate (R=5); exact corners still weld-tight
			}
			if got.DistanceTo(want) > tol {
				t.Errorf("edge %v f=%g: fill %v vs rail %v (d=%g > %g)", ck.e, f, got, want, got.DistanceTo(want), tol)
			}
		}
	}
}

// sidePointAt samples curve c at fraction f of its domain (reversed if rev).
func sidePointAt(c geom.Curve3, f float64, rev bool) math.Point3 {
	lo, hi := c.Domain()
	if rev {
		f = 1 - f
	}
	return c.PointAt(lo + f*(hi-lo))
}

// TestCoons4RejectsSliver proves Build does not panic and honest-rejects a degenerate near-zero-height
// loop (two generator rails collapse to near-zero length → a folded/zero-Jacobian sliver).
func TestCoons4RejectsSliver(t *testing.T) {
	t.Parallel()
	loop := quarterCylLoop(t, 1e-9)
	p := coons4Provider{}
	_, cert, ok := p.Build(loop, blendScale())
	if ok && cert.Valid(blendScale()) {
		t.Fatalf("expected honest-reject on a sliver loop, got a valid certificate: %+v", cert)
	}
}
