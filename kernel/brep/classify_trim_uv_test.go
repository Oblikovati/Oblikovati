// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// surfacePeriodic classifies each analytic surface's wrap: plane neither, cylinder/cone u only,
// sphere u only (latitude is bounded), torus both.
func TestSurfacePeriodic(t *testing.T) {
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), 1)
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, c := range []struct {
		name       string
		s          geom.Surface
		uPer, vPer bool
	}{
		{"plane", pl, false, false},
		{"cylinder", cyl, true, false},
		{"sphere", sph, true, false},
		{"torus", tor, true, true},
	} {
		if u, v := surfacePeriodic(c.s); u != c.uPer || v != c.vPer {
			t.Errorf("surfacePeriodic(%s) = (%v,%v), want (%v,%v)", c.name, u, v, c.uPer, c.vPer)
		}
	}
}

// continueUV keeps a wrapped parameter sequence monotone across the seam in both directions.
func TestContinueUVUnwrap(t *testing.T) {
	// u wraps from ~2π back to ~0: the seam jump is unwrapped forward past 2π.
	ring := []math.Point2{math.P2(2*stdmath.Pi-0.1, 3)}
	u, _ := continueUV(ring, 0.1, 3, true, false)
	if stdmath.Abs(u-(2*stdmath.Pi+0.1)) > 1e-9 {
		t.Errorf("u unwrap across seam = %g, want ≈2π+0.1", u)
	}
	// v periodic (torus tube) unwraps too; a non-periodic axis is left untouched.
	_, vv := continueUV([]math.Point2{math.P2(0, 2*stdmath.Pi-0.1)}, 0, 0.1, false, true)
	if stdmath.Abs(vv-(2*stdmath.Pi+0.1)) > 1e-9 {
		t.Errorf("v unwrap across seam = %g, want ≈2π+0.1", vv)
	}
	if _, keep := continueUV([]math.Point2{math.P2(0, 100)}, 0, 0.1, false, false); keep != 0.1 {
		t.Errorf("non-periodic v left = %g, want 0.1", keep)
	}
}

// castAxis routes the even-odd uv ray to a non-periodic UNBOUNDED axis (plane, cylinder, cone) and
// declines for a sphere/torus (no exterior endpoint), where the classifier uses the geodesic winding.
func TestCastAxisRouting(t *testing.T) {
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	cone, _ := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/4)
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), 1)
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, c := range []struct {
		name   string
		s      geom.Surface
		wantOK bool
	}{
		{"plane", pl, true}, {"cylinder", cyl, true}, {"cone", cone, true},
		{"sphere", sph, false}, {"torus", tor, false},
	} {
		uPer, vPer := surfacePeriodic(c.s)
		if _, ok := castAxis(c.s, uPer, vPer); ok != c.wantOK {
			t.Errorf("castAxis(%s) ok=%v, want %v (sphere/torus must defer to the geodesic winding)", c.name, ok, c.wantOK)
		}
	}
}
