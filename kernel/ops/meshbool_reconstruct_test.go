// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// TestReconstructSteppedShaftKeepsAnalyticWalls is the ADR-0054 payoff and the #2167
// mechanism in miniature: two coaxial cylinders of different radius stacked and unioned.
// Both walls are UNTOUCHED (nothing removes them) so they must survive as analytic
// cylinders; only the shared cap is SPLIT into an annulus whose outer and inner rims
// reuse the two original rim circles. The faceted fallback would turn both walls into
// mismatched facet grids (the visible seam); reconstruction keeps them one smooth
// surface each. Asserts: valid closed solid, exactly two analytic cylinder walls, and
// the exact stepped volume.
func TestReconstructSteppedShaftKeepsAnalyticWalls(t *testing.T) {
	lower, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 5)
	if err != nil {
		t.Fatalf("lower cylinder: %v", err)
	}
	upper, err := brep.SolidCylinder(math.P3(0, 0, 5), math.V3(0, 0, 1), 1.5, 5)
	if err != nil {
		t.Fatalf("upper cylinder: %v", err)
	}

	body, ok := reconstructBoolean(lower, upper, meshbool.Union, DefaultQuality())
	if !ok {
		t.Fatal("reconstruction declined a pure gluing union it should handle")
	}
	if !body.IsSolid() || !Validate(body).Valid {
		t.Fatalf("reconstructed shaft is not a valid solid (solid=%v valid=%v)", body.IsSolid(), Validate(body).Valid)
	}
	if n := cylinderFaceCount(body); n != 2 {
		t.Fatalf("reconstructed shaft has %d analytic cylinder walls, want 2", n)
	}

	want := stdmath.Pi*9*5 + stdmath.Pi*2.25*5 // π r² h for each stage
	if v := soupVolume(bodyToSoup(body, PropertyQuality())); stdmath.Abs(v-want) > 0.02*want {
		t.Fatalf("reconstructed shaft volume = %.4f, want ~%.4f", v, want)
	}
}

// TestReconstructBoxUnionExactVolume proves the reconstruction also handles the planar
// case end to end: two overlapping boxes union to a valid solid of the exact volume,
// through the same tag-grouped reconstruction path.
func TestReconstructBoxUnionExactVolume(t *testing.T) {
	a := boxBodyAt(2, 2, 2, math.V3(0, 0, 0), "a")
	b := boxBodyAt(2, 2, 2, math.V3(1, 0, 0), "b") // overlaps a in x: union volume = 12
	body, ok := reconstructBoolean(a, b, meshbool.Union, DefaultQuality())
	if !ok {
		t.Fatal("reconstruction declined a planar box union")
	}
	if !body.IsSolid() || !Validate(body).Valid {
		t.Fatal("reconstructed box union is not a valid solid")
	}
	if v := soupVolume(bodyToSoup(body, PropertyQuality())); stdmath.Abs(v-12) > 1e-6 {
		t.Fatalf("reconstructed box union volume = %.6f, want 12", v)
	}
}
