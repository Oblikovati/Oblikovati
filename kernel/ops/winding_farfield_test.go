// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	stdrand "math/rand"
	"testing"

	"oblikovati.org/math"
)

// uvSphereMesh builds an outward-oriented closed UV-sphere mesh (radius r at the origin) with
// nLat latitude bands × nLon longitude steps — the synthetic solid boundary the winding tests
// and benchmarks scale by triangle count.
func uvSphereMesh(nLat, nLon int, r float64) *Mesh {
	m := &Mesh{}
	at := func(i, j int) math.Point3 {
		theta := stdmath.Pi * float64(i) / float64(nLat)
		phi := 2 * stdmath.Pi * float64(j%nLon) / float64(nLon)
		return math.P3(r*stdmath.Sin(theta)*stdmath.Cos(phi), r*stdmath.Sin(theta)*stdmath.Sin(phi), r*stdmath.Cos(theta))
	}
	push := func(pts ...math.Point3) {
		for _, p := range pts {
			m.Indices = append(m.Indices, len(m.Positions))
			m.Positions = append(m.Positions, p)
		}
	}
	for i := 0; i < nLat; i++ {
		for j := 0; j < nLon; j++ {
			a, b, c, d := at(i, j), at(i+1, j), at(i+1, j+1), at(i, j+1)
			if i > 0 {
				push(a, b, d) // outward CCW seen from outside
			}
			if i < nLat-1 {
				push(b, c, d)
			}
		}
	}
	return m
}

// TestWindingFarFieldMatchesBruteOnRandomPoints is the exactness cross-check #1607 requires:
// on 1000 randomized query points — near-surface, interior, and far-field alike — the cached
// classifier must agree with the exact brute loop on EVERY point.
func TestWindingFarFieldMatchesBruteOnRandomPoints(t *testing.T) {
	mesh := uvSphereMesh(24, 32, 1)
	far := newMeshWindingFarField(mesh)
	rng := stdrand.New(stdrand.NewSource(2018)) // Barill et al. year, for flavour
	for k := 0; k < 1000; k++ {
		span := []float64{1.05, 1.5, 30}[k%3] // surface band, mid range, deep far field
		p := math.P3(rng.Float64()*2*span-span, rng.Float64()*2*span-span, rng.Float64()*2*span-span)
		if got, want := far.inside(p), pointInMesh(mesh, p); got != want {
			t.Fatalf("point %d %v: cached inside=%v, exact loop says %v", k, p, got, want)
		}
	}
}

// TestWindingFarFieldCertifiesOnlyFarQueries pins both sides of the early-out: a deep
// far-field query certifies (no triangle visits needed), while near-box and interior queries
// must NOT certify — they belong to the exact loop.
func TestWindingFarFieldCertifiesOnlyFarQueries(t *testing.T) {
	far := newMeshWindingFarField(uvSphereMesh(16, 24, 1))
	if !far.certifiedOutside(math.P3(100, 100, 100)) {
		t.Fatal("deep far-field query did not certify outside")
	}
	if far.certifiedOutside(math.P3(0, 0, 0)) || far.certifiedOutside(math.P3(1.01, 0, 0)) {
		t.Fatal("interior/near-surface query certified outside — the cap math is broken")
	}
	if far.inside(math.P3(100, 100, 100)) || !far.inside(math.P3(0, 0, 0)) {
		t.Fatal("far-field classifier misclassified an obvious point")
	}
}

// TestWindingFarFieldEmptyMesh: the degenerate no-triangle mesh classifies everything outside,
// like the brute loop (winding 0).
func TestWindingFarFieldEmptyMesh(t *testing.T) {
	if newMeshWindingFarField(&Mesh{}).inside(math.P3(0, 0, 0)) {
		t.Fatal("empty mesh classified a point inside")
	}
}

// TestInsideMeshQuerierGates covers both querier arms: below the amortization gate it must
// hand back the brute loop, above it the cached classifier — and both agree with pointInMesh.
func TestInsideMeshQuerierGates(t *testing.T) {
	mesh := uvSphereMesh(12, 16, 1)
	for _, queries := range []int{1, 8} {
		inMesh := insideMeshQuerier(mesh, queries)
		for _, p := range []math.Point3{math.P3(0, 0, 0), math.P3(0.9, 0, 0), math.P3(50, 0, 0)} {
			if got, want := inMesh(p), pointInMesh(mesh, p); got != want {
				t.Fatalf("querier(queries=%d) at %v = %v, brute loop says %v", queries, p, got, want)
			}
		}
	}
}

// TestPointBoxDistanceSq covers the far-field distance helper: inside → 0, face/corner gaps
// exact.
func TestPointBoxDistanceSq(t *testing.T) {
	b := math.NewBox(math.P3(0, 0, 0), math.P3(2, 2, 2))
	if d := pointBoxDistanceSq(math.P3(1, 1, 1), b); d != 0 {
		t.Fatalf("inside point distance² = %g, want 0", d)
	}
	if d := pointBoxDistanceSq(math.P3(5, 1, 1), b); d != 9 {
		t.Fatalf("face gap distance² = %g, want 9", d)
	}
	if d := pointBoxDistanceSq(math.P3(3, 3, 3), b); d != 3 {
		t.Fatalf("corner gap distance² = %g, want 3", d)
	}
}
