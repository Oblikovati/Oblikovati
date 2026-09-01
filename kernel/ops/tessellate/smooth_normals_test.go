// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// twoFacets builds two triangles that share an edge (as a faceted mesh does: each facet has its
// OWN vertices at the shared positions), facet A normal +Z and facet B tilted by `dihedral`.
func twoFacets(dihedral float64) *Mesh {
	nA := math.V3(0, 0, 1)
	nB := math.V3(0, math.Scalar(stdmath.Sin(dihedral)), math.Scalar(stdmath.Cos(dihedral)))
	s0, s1 := math.P3(0, 0, 0), math.P3(1, 0, 0) // the shared edge
	return &Mesh{
		Positions: []math.Point3{s0, s1, math.P3(0, -1, 0), s0, s1, math.P3(0, 1, 0)},
		Normals:   []math.Vector3{nA, nA, nA, nB, nB, nB},
		Indices:   []int{0, 1, 2, 3, 4, 5},
	}
}

func TestSmoothShadeNormalsAveragesBelowCrease(t *testing.T) {
	t.Parallel()
	// 20° dihedral is below the 35° crease → the shared-edge vertices average to a tilted normal.
	sm := SmoothShadeNormals(twoFacets(20*stdmath.Pi/180), DefaultCreaseAngle())
	got := sm[0] // facet A's copy of the shared vertex s0
	if stdmath.Abs(float64(got.Z)-1) < 1e-9 && stdmath.Abs(float64(got.Y)) < 1e-9 {
		t.Fatalf("shared vertex normal was not smoothed: %v (still flat +Z)", got)
	}
	// it should be the unit average of nA and nB (tilted ~10° toward +Y)
	if got.Y <= 0 || got.Z <= 0 {
		t.Fatalf("smoothed normal not the blend of the two facets: %v", got)
	}
	if l := float64(got.Length()); stdmath.Abs(l-1) > 1e-9 {
		t.Errorf("smoothed normal not unit length: |n|=%g", l)
	}
}

func TestSmoothShadeNormalsKeepsSharpEdge(t *testing.T) {
	t.Parallel()
	// 90° dihedral exceeds the crease → the shared-edge vertices keep their facet normals.
	sm := SmoothShadeNormals(twoFacets(90*stdmath.Pi/180), DefaultCreaseAngle())
	if got := sm[0]; stdmath.Abs(float64(got.Z)-1) > 1e-9 || stdmath.Abs(float64(got.Y)) > 1e-9 {
		t.Fatalf("sharp edge was smoothed: shared vertex normal = %v, want +Z", got)
	}
}

// TestSmoothShadeNormalsDoesNotMutateInput guards that mass-properties (which use the raw mesh)
// are unaffected: SmoothShadeNormals returns a new slice and leaves m.Normals alone.
func TestSmoothShadeNormalsDoesNotMutateInput(t *testing.T) {
	t.Parallel()
	m := twoFacets(20 * stdmath.Pi / 180)
	before := m.Normals[0]
	_ = SmoothShadeNormals(m, DefaultCreaseAngle())
	if m.Normals[0] != before {
		t.Errorf("input normals mutated: %v != %v", m.Normals[0], before)
	}
}
