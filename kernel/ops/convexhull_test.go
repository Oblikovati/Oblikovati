// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/subd"
	"oblikovati/math"
)

func cubeCorners(s float64) []math.Point3 {
	var p []math.Point3
	for _, x := range []float64{0, s} {
		for _, y := range []float64{0, s} {
			for _, z := range []float64{0, s} {
				p = append(p, math.P3(x, y, z))
			}
		}
	}
	return p
}

func hullVolume(t *testing.T, pts []math.Point3) float64 {
	t.Helper()
	body, err := ops.ConvexHull(pts, "hull")
	if err != nil {
		t.Fatalf("ConvexHull: %v", err)
	}
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("hull is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Fatalf("hull has %d boundary edges, want 0 (watertight)", len(open))
	}
	return ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
}

// TestConvexHullCube hulls a cube's 8 corners — the hull is the cube itself.
func TestConvexHullCube(t *testing.T) {
	if got := hullVolume(t, cubeCorners(2)); stdmath.Abs(got-8) > 1e-9 {
		t.Errorf("hull(cube) volume = %.6f, want 8", got)
	}
}

// TestConvexHullCubeApex hulls a unit cube plus an apex above the top face — a cube capped by
// a pyramid. Volume = 1 + ⅓·(1·1)·2.
func TestConvexHullCubeApex(t *testing.T) {
	pts := append(cubeCorners(1), math.P3(0.5, 0.5, 3))
	if got, want := hullVolume(t, pts), 1.0+(1.0/3.0)*1*2; stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("hull(cube+apex) volume = %.6f, want %.6f", got, want)
	}
}

// TestConvexHullSphere hulls points sampled on a sphere; the inscribed polytope approaches the
// sphere volume from below as the sampling densifies.
func TestConvexHullSphere(t *testing.T) {
	const r, n = 1.5, 400
	var pts []math.Point3
	ga := stdmath.Pi * (3 - stdmath.Sqrt(5)) // golden-angle Fibonacci sphere
	for i := 0; i < n; i++ {
		z := 1 - 2*(float64(i)+0.5)/n
		rho := stdmath.Sqrt(1 - z*z)
		th := ga * float64(i)
		pts = append(pts, math.P3(r*rho*stdmath.Cos(th), r*rho*stdmath.Sin(th), r*z))
	}
	got := hullVolume(t, pts)
	want := 4.0 / 3.0 * stdmath.Pi * r * r * r
	if got > want || (want-got)/want > 0.02 {
		t.Errorf("hull(sphere) volume = %.5f, want just under %.5f (inscribed, ≤2%%)", got, want)
	}
}

// TestConvexHullOfTwoBoxes hulls two separated unit boxes via ConvexHullOf — the result spans
// both and is a valid solid no smaller than either box.
func TestConvexHullOfTwoBoxes(t *testing.T) {
	a := subd.ToBody(subd.Box(1, 1, 1), "a")
	bm := subd.Box(1, 1, 1)
	for i := range bm.Verts {
		bm.Verts[i] = bm.Verts[i].TranslateBy(math.V3(3, 0, 0)) // [3,4]×[0,1]×[0,1]
	}
	b := subd.ToBody(bm, "b")
	hull, err := ops.ConvexHullOf("hull", a, b)
	if err != nil {
		t.Fatalf("ConvexHullOf: %v", err)
	}
	if r := ops.Validate(hull); !r.Valid || !hull.IsSolid() {
		t.Fatalf("hull of two boxes invalid: %+v", r)
	}
	// The hull spans x∈[0,4]; its volume strictly exceeds the two boxes (2) and is bounded by
	// the enclosing 4×1×1 box (4).
	v := ops.BodyGeometryProperties(hull, ops.DefaultQuality()).Volume
	if v <= 2 || v > 4+1e-9 {
		t.Errorf("hull(two boxes) volume = %.4f, want in (2, 4]", v)
	}
}

// TestConvexHullDegenerate rejects a coplanar cloud (no 3D hull).
func TestConvexHullDegenerate(t *testing.T) {
	planar := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(1, 1, 0)}
	if _, err := ops.ConvexHull(planar, "hull"); err == nil {
		t.Error("expected an error hulling coplanar points, got nil")
	}
}
