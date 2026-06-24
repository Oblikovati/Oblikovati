// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Stereographic >hemisphere patch (M2 Phase 1, Oblikovati/Oblikovati#1334): a sphere patch covering MORE
// than a hemisphere (the 7/8 sphere left when an octant is removed) exceeds the gnomonic chart's limit;
// the stereographic chart must mesh it to its true area, 7/8·4πR², with every vertex on the sphere.

// quarterArc samples a 90° great-circle arc from a to b on the sphere (a, b orthogonal, |a|=|b|=R), n
// points, excluding the last (so concatenated arcs do not duplicate the shared corner).
func quarterArc(center, a, b math.Point3, n int) []math.Point3 {
	var out []math.Point3
	for k := 0; k < n; k++ {
		phi := (stdmath.Pi / 2) * float64(k) / float64(n)
		c, s := stdmath.Cos(phi), stdmath.Sin(phi)
		out = append(out, math.P3(
			center.X+(a.X-center.X)*math.Scalar(c)+(b.X-center.X)*math.Scalar(s),
			center.Y+(a.Y-center.Y)*math.Scalar(c)+(b.Y-center.Y)*math.Scalar(s),
			center.Z+(a.Z-center.Z)*math.Scalar(c)+(b.Z-center.Z)*math.Scalar(s)))
	}
	return out
}

func patchArea(m *Mesh) float64 {
	a := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		p0, p1, p2 := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		a += 0.5 * float64(p0.VectorTo(p1).Cross(p0.VectorTo(p2)).Length())
	}
	return a
}

func TestSpherePatchStereographicSevenEighths(t *testing.T) {
	const R = 5.0
	O := math.P3(0, 0, 0)
	A, B, C := math.P3(R, 0, 0), math.P3(0, R, 0), math.P3(0, 0, R)
	sphere, _ := geom.NewSphere(O, R)
	// Boundary of the 7/8 region (complement of the +,+,+ octant), wound A→C→B→A so the BIG region is
	// the interior (the opposite winding of the octant itself).
	var outer []math.Point3
	outer = append(outer, quarterArc(O, A, C, 8)...) // A→C (y=0 plane)
	outer = append(outer, quarterArc(O, C, B, 8)...) // C→B (x=0 plane)
	outer = append(outer, quarterArc(O, B, A, 8)...) // B→A (z=0 plane)

	m, ok := spherePatchMesh(sphere, outer, nil, DefaultQuality())
	if !ok {
		t.Fatal("spherePatchMesh declined the 7/8 patch (stereographic chart should claim it)")
	}
	for _, p := range m.Positions {
		if d := float64(p.DistanceTo(O)); stdmath.Abs(d-R) > 1e-6 {
			t.Fatalf("vertex %v off the sphere (r=%.6f)", p, d)
		}
	}
	got := patchArea(m)
	want := 7.0 / 8.0 * 4 * stdmath.Pi * R * R
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("7/8 patch area %.4f, want %.4f (rel %.4f > 2%%)", got, want, rel)
	}
}
