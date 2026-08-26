// SPDX-License-Identifier: GPL-2.0-only

package fit

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// normalsAligned reports whether two vectors are parallel or anti-parallel within tol (a plane
// normal is only defined up to sign and is compared by direction, so neither need be unit).
func normalsAligned(a, b math.Vector3, tol float64) bool {
	cos := float64(a.Dot(b)) / (a.Length() * b.Length())
	return stdmath.Abs(stdmath.Abs(cos)-1) < tol
}

func TestPlaneFitAxisAligned(t *testing.T) {
	// Points on z = 5 → normal ±Z, origin z = 5.
	pts := []math.Point3{
		math.P3(0, 0, 5), math.P3(1, 0, 5), math.P3(0, 1, 5),
		math.P3(1, 1, 5), math.P3(2, 3, 5), math.P3(-1, 2, 5),
	}
	pl, err := Plane(pts)
	if err != nil {
		t.Fatalf("Plane: %v", err)
	}
	if !normalsAligned(pl.Normal(), math.V3(0, 0, 1), 1e-6) {
		t.Errorf("normal = %v, want ±Z", pl.Normal())
	}
	if stdmath.Abs(float64(pl.Origin.Z)-5) > 1e-9 {
		t.Errorf("origin Z = %v, want 5", pl.Origin.Z)
	}
}

func TestPlaneFitTilted(t *testing.T) {
	// Plane x + y + z = 0 → normal ∝ (1,1,1). Build points spanning it.
	want := math.V3(1, 1, 1)
	var pts []math.Point3
	for _, uv := range [][2]float64{{1, 0}, {0, 1}, {-1, 0}, {0, -1}, {1, 1}, {2, -1}, {-2, 3}} {
		u, v := uv[0], uv[1]
		// two in-plane directions of x+y+z=0: (1,-1,0) and (1,1,-2)
		x := u*1 + v*1
		y := u*-1 + v*1
		z := u*0 + v*-2
		pts = append(pts, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)))
	}
	pl, err := Plane(pts)
	if err != nil {
		t.Fatalf("Plane: %v", err)
	}
	if !normalsAligned(pl.Normal(), want, 1e-6) {
		t.Errorf("normal = %v, want ∝ (1,1,1)", pl.Normal())
	}
}

func TestPlaneFitNoisyStaysClose(t *testing.T) {
	// z = 0 plane with small deterministic noise → normal stays near ±Z.
	var pts []math.Point3
	for i := range 50 {
		x := float64(i%7) - 3
		y := float64(i%5) - 2
		z := 0.01 * float64((i*13)%5-2) // tiny out-of-plane jitter
		pts = append(pts, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)))
	}
	pl, err := Plane(pts)
	if err != nil {
		t.Fatalf("Plane: %v", err)
	}
	if !normalsAligned(pl.Normal(), math.V3(0, 0, 1), 1e-2) {
		t.Errorf("noisy normal = %v, want near ±Z", pl.Normal())
	}
}

func TestPlaneFitErrors(t *testing.T) {
	if _, err := Plane([]math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1)}); err == nil {
		t.Error("want error with fewer than 3 points")
	}
	// Collinear points (all on the X axis) span no plane.
	collinear := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0), math.P3(5, 0, 0)}
	if _, err := Plane(collinear); err == nil {
		t.Error("want error for collinear points")
	}
}

func TestJacobiEigenKnownMatrix(t *testing.T) {
	// Diagonal matrix → eigenvalues are the diagonal, eigenvectors the axes.
	vals, _ := jacobiEigen3(mat3{{3, 0, 0}, {0, 1, 0}, {0, 0, 2}})
	got := map[int]bool{}
	for _, v := range vals {
		got[int(stdmath.Round(v))] = true
	}
	if !got[1] || !got[2] || !got[3] {
		t.Errorf("eigenvalues = %v, want {1,2,3}", vals)
	}
}
