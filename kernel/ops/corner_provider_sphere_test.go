// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// sphereTriLoop builds a closed spherical-triangle RailLoop: three quarter great-circles of
// Sphere{origin, r}, one per coordinate plane, chaining (r,0,0)->(0,r,0)->(0,0,r)->(r,0,0). This is
// exactly the shape an equal-radius trihedral corner leaves behind (see analyticSphereProvider docs).
func sphereTriLoop(t *testing.T, r float64) RailLoop {
	t.Helper()
	o := math.P3(0, 0, 0)
	mk := func(normal, refDir math.Vector3) Side {
		a, err := geom.NewArc3d(o, normal, refDir, r, 0, stdmath.Pi/2)
		if err != nil {
			t.Fatalf("arc: %v", err)
		}
		return Side{Curve: a, Cont: G1}
	}
	return RailLoop{Sides: []Side{
		mk(math.V3(0, 0, 1), math.V3(1, 0, 0)), // (r,0,0)->(0,r,0) in XY
		mk(math.V3(1, 0, 0), math.V3(0, 1, 0)), // (0,r,0)->(0,0,r) in YZ
		mk(math.V3(0, 1, 0), math.V3(0, 0, 1)), // (0,0,r)->(r,0,0) in XZ
	}}
}

// TestSphereTriLoopFixtureIsClosed pins the fixture itself: the three quarter-arcs must chain into a
// single closed cycle (Task requires verifying the arcs' endpoints actually meet).
func TestSphereTriLoopFixtureIsClosed(t *testing.T) {
	t.Parallel()
	if !sphereTriLoop(t, 4).Closed(1e-9) {
		t.Fatal("sphereTriLoop fixture must be a closed loop")
	}
}

// TestAnalyticSphereFitsAndBuilds pins the happy path: three equal-radius concentric rail arcs
// recognise as a sphere, and Build returns a valid, correctly-sized certified patch.
func TestAnalyticSphereFitsAndBuilds(t *testing.T) {
	t.Parallel()
	loop := sphereTriLoop(t, 4)
	p := analyticSphereProvider{}
	if !p.Fits(loop) {
		t.Fatal("expected Fits true for a concentric equal-radius rail loop")
	}
	scale := blendScale()
	patch, cert, ok := p.Build(loop, scale)
	if !ok {
		t.Fatal("expected Build ok=true")
	}
	if !cert.Valid(scale) {
		t.Fatalf("expected a valid certificate, got %+v", cert)
	}
	if !cert.NoFold {
		t.Error("a sphere never folds: NoFold must be true")
	}
	if patch.Kind != BlendKindSphere {
		t.Errorf("expected Kind %q, got %q", BlendKindSphere, patch.Kind)
	}
	sph, isSphere := patch.Surface.(geom.Sphere)
	if !isSphere {
		t.Fatalf("expected patch.Surface to be a geom.Sphere, got %T", patch.Surface)
	}
	if d := stdmath.Abs(sph.Radius - 4); d > scale.Weld() {
		t.Errorf("radius: got %v, want ~4 (within %v), diff %v", sph.Radius, scale.Weld(), d)
	}
	if d := sph.Center.DistanceTo(math.P3(0, 0, 0)); d > scale.Weld() {
		t.Errorf("center: got %v, want ~origin (within %v), diff %v", sph.Center, scale.Weld(), d)
	}
}

// TestAnalyticSphereRejectsNonConcentric pins the rejection path: shifting one rail arc's center far
// off the other two's must fail both Fits and Build, so the tier walk moves on (ADR-3).
func TestAnalyticSphereRejectsNonConcentric(t *testing.T) {
	t.Parallel()
	loop := sphereTriLoop(t, 4)
	shifted, err := geom.NewArc3d(math.P3(1, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 4, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("arc: %v", err)
	}
	loop.Sides[0] = Side{Curve: shifted, Cont: G1}

	p := analyticSphereProvider{}
	if p.Fits(loop) {
		t.Fatal("expected Fits false for a non-concentric rail loop")
	}
	if _, _, ok := p.Build(loop, blendScale()); ok {
		t.Fatal("expected Build ok=false for a non-concentric rail loop")
	}
}

// TestAnalyticSphereRejectsNonArc pins that a loop with any non-arc side (here a straight line) is
// never recognised as a sphere corner — recognition depends on every side being an exact Arc3d.
func TestAnalyticSphereRejectsNonArc(t *testing.T) {
	t.Parallel()
	loop := sphereTriLoop(t, 4)
	loop.Sides[1] = Side{
		Curve: geom.NewLineSegment(curveStart(loop.Sides[1].Curve), curveEnd(loop.Sides[1].Curve)),
		Cont:  G1,
	}

	p := analyticSphereProvider{}
	if p.Fits(loop) {
		t.Fatal("expected Fits false for a loop containing a non-arc side")
	}
}
