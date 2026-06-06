// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
)

// ring returns an n-gon of radius r centered on the axis at height z, wound CCW (viewed from
// +z) when ccw is set and CW otherwise — to drive the winding-invariance test.
func ring(z, r float64, n int, ccw bool) []math.Point3 {
	out := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		k := i
		if !ccw {
			k = n - i
		}
		a := 2 * stdmath.Pi * float64(k) / float64(n)
		out[i] = math.P3(math.Scalar(r*stdmath.Cos(a)), math.Scalar(r*stdmath.Sin(a)), math.Scalar(z))
	}
	return out
}

func tubeRings(ccw bool) (outer, inner [][]math.Point3) {
	const n = 48
	outer = [][]math.Point3{ring(0, 2, n, ccw), ring(4, 2, n, ccw)}
	inner = [][]math.Point3{ring(0, 1, n, ccw), ring(4, 1, n, ccw)}
	return outer, inner
}

// TestTubeSolidIsWatertightWithCorrectVolume checks the direct tube mesh: an outer/inner
// cylindrical pair skins a watertight pipe whose volume is the annulus area × height.
func TestTubeSolidIsWatertightWithCorrectVolume(t *testing.T) {
	outer, inner := tubeRings(true)
	body, err := tubeSolid(skinnedSections(outer, 48, false), skinnedSections(inner, 48, false), false, "tube")
	if err != nil {
		t.Fatalf("tubeSolid: %v", err)
	}
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("tube is not a valid solid: valid=%v solid=%v closed=%v", r.Valid, body.IsSolid(), r.Closed)
	}
	want := stdmath.Pi * (4 - 1) * 4 // (R²−r²)·h
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, want) > 0.02 {
		t.Errorf("tube volume = %g, want ≈%g", v, want)
	}
}

// TestTubeSolidWindingInvariant checks that a tube built from CW-wound input rings is still a
// valid outward solid with positive volume — the coherent mesh lets the signed-volume flip pick
// the global orientation, so callers need not pre-normalize ring winding.
func TestTubeSolidWindingInvariant(t *testing.T) {
	for _, ccw := range []bool{true, false} {
		outer, inner := tubeRings(ccw)
		body, err := tubeSolid(skinnedSections(outer, 48, false), skinnedSections(inner, 48, false), false, "tube")
		if err != nil {
			t.Fatalf("ccw=%v tubeSolid: %v", ccw, err)
		}
		gp := ops.BodyGeometryProperties(body, ops.DefaultQuality())
		if r := ops.Validate(body); !r.Valid || !body.IsSolid() || gp.Volume <= 0 {
			t.Fatalf("ccw=%v tube invalid: valid=%v solid=%v vol=%g", ccw, r.Valid, body.IsSolid(), gp.Volume)
		}
	}
}

// TestTubeSolidClosedIsToroidal checks a closed-loop tube (no caps): the section sequence wraps,
// giving a hollow torus-like shell that is still a valid watertight solid.
func TestTubeSolidClosedIsToroidal(t *testing.T) {
	const n, segs = 32, 24
	outer := make([][]math.Point3, segs)
	inner := make([][]math.Point3, segs)
	for s := 0; s < segs; s++ {
		a := 2 * stdmath.Pi * float64(s) / float64(segs)
		cx, cy := 6*stdmath.Cos(a), 6*stdmath.Sin(a) // sweep center around a radius-6 circle
		out := make([]math.Point3, n)
		in := make([]math.Point3, n)
		for i := 0; i < n; i++ {
			b := 2 * stdmath.Pi * float64(i) / float64(n)
			// tube cross-section in the plane containing the axis and the sweep radius
			ro, ri := 1.5, 0.8
			dirx, diry := stdmath.Cos(a), stdmath.Sin(a)
			off := func(rr float64) math.Point3 {
				return math.P3(math.Scalar(cx+rr*stdmath.Cos(b)*dirx), math.Scalar(cy+rr*stdmath.Cos(b)*diry), math.Scalar(rr*stdmath.Sin(b)))
			}
			out[i] = off(ro)
			in[i] = off(ri)
		}
		outer[s] = out
		inner[s] = in
	}
	body, err := tubeSolid(outer, inner, true, "torus-tube")
	if err != nil {
		t.Fatalf("closed tubeSolid: %v", err)
	}
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("closed tube not a valid solid: valid=%v solid=%v closed=%v manifold=%v", r.Valid, body.IsSolid(), r.Closed, r.Manifold)
	}
}
