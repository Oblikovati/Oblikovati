// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// perpendicularToSurface asserts the residual p−foot is perpendicular to both surface
// tangents (the defining property of the projection foot).
func perpendicularToSurface(t *testing.T, s Surface, p math.Point3) (u, v float64, foot math.Point3) {
	t.Helper()
	u, v, foot = ClosestPointOnSurface(s, p)
	du, dv := s.DerivativesAt(u, v)
	r := foot.VectorTo(p)
	if stdmath.Abs(float64(du.Dot(r))) > 1e-6 || stdmath.Abs(float64(dv.Dot(r))) > 1e-6 {
		t.Errorf("foot not perpendicular: du·r=%v dv·r=%v", du.Dot(r), dv.Dot(r))
	}
	return u, v, foot
}

func TestClosestPointOnPlane(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	_, _, foot := perpendicularToSurface(t, pl, math.P3(2, 3, 5))
	if foot.DistanceTo(math.P3(2, 3, 0)) > 1e-9 {
		t.Errorf("plane projection = %v, want (2,3,0)", foot)
	}
}

func TestClosestPointOnSphere(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	_, _, foot := perpendicularToSurface(t, sp, math.P3(10, 0, 0))
	if stdmath.Abs(float64(foot.AsVector().Length())-5) > 1e-6 {
		t.Errorf("sphere projection radius = %v, want 5", foot.AsVector().Length())
	}
	if foot.DistanceTo(math.P3(5, 0, 0)) > 1e-6 {
		t.Errorf("sphere projection = %v, want (5,0,0)", foot)
	}
}

func TestClosestPointOnCylinder(t *testing.T) {
	cy, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	u, _, foot := perpendicularToSurface(t, cy, math.P3(10, 0, 7))
	_ = u
	radial := stdmath.Hypot(float64(foot.X), float64(foot.Y))
	if stdmath.Abs(radial-3) > 1e-6 || stdmath.Abs(float64(foot.Z)-7) > 1e-6 {
		t.Errorf("cylinder projection = %v, want radius 3 / z 7", foot)
	}
}

func TestClosestPointOnConeAndTorus(t *testing.T) {
	// The cone and torus exercise the Gauss–Newton refinement (ParamAt is approximate).
	co, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6)
	perpendicularToSurface(t, co, math.P3(4, 1, 3))
	to, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1)
	_, _, foot := perpendicularToSurface(t, to, math.P3(9, 0, 0))
	// On the torus, the far point projects to the outer equator at major+minor = 6.
	if r := stdmath.Hypot(float64(foot.X), float64(foot.Y)); stdmath.Abs(r-6) > 1e-6 {
		t.Errorf("torus projection outer radius = %v, want 6", r)
	}
}

func TestSignedDistanceToPlane(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if d := SignedDistanceToSurface(pl, math.P3(0, 0, 4)); stdmath.Abs(d-4) > 1e-9 {
		t.Errorf("signed distance above plane = %v, want +4", d)
	}
	if d := SignedDistanceToSurface(pl, math.P3(0, 0, -2)); stdmath.Abs(d+2) > 1e-9 {
		t.Errorf("signed distance below plane = %v, want -2", d)
	}
}

// TestClosestPointDegenerateFrame exercises the degenerate-frame guard by projecting a
// point at a sphere pole (where ∂P/∂v vanishes).
func TestClosestPointDegenerateFrame(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 2)
	_, _, foot := ClosestPointOnSurface(sp, math.P3(0, 0, 10))
	if foot.DistanceTo(math.P3(0, 0, 2)) > 1e-6 {
		t.Errorf("pole projection = %v, want (0,0,2)", foot)
	}
}

// TestClosestPointConeApexGuard projects a point level with a cone's apex, where the
// tangent frame collapses (du = 0 at v = 0); the iteration must bail safely rather than
// divide by a zero determinant.
func TestClosestPointConeApexGuard(t *testing.T) {
	co, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6)
	u, v, foot := ClosestPointOnSurface(co, math.P3(1, 0, 0))
	if stdmath.IsNaN(u) || stdmath.IsNaN(v) || stdmath.IsNaN(float64(foot.X)) {
		t.Errorf("apex-level projection produced NaN: u=%v v=%v foot=%v", u, v, foot)
	}
}

// TestGaussNewtonStepDegenerate covers the singular normal-equation guard directly.
func TestGaussNewtonStepDegenerate(t *testing.T) {
	if _, _, ok := gaussNewtonStep(math.V3(0, 0, 0), math.V3(0, 0, 0), 1, 1); ok {
		t.Error("a zero tangent frame should report a non-invertible step")
	}
	if _, _, ok := gaussNewtonStep(math.V3(1, 0, 0), math.V3(0, 1, 0), 2, 3); !ok {
		t.Error("a well-conditioned frame should report an invertible step")
	}
}

// TestClampFinite covers the lower/upper/unbounded clamp branches.
func TestClampFinite(t *testing.T) {
	if clampFinite(-5, 0, 10) != 0 || clampFinite(15, 0, 10) != 10 || clampFinite(5, 0, 10) != 5 {
		t.Error("clampFinite should pin to [lo,hi]")
	}
	if clampFinite(99, stdmath.Inf(-1), stdmath.Inf(1)) != 99 {
		t.Error("clampFinite should leave an unbounded value unchanged")
	}
}
