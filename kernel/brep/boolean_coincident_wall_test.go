// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Two coaxial cylinders of one radius that overlap are the DEGENERATE OVERLAP: their walls lie on the
// same surface, so their contact is a region and not a curve. The general pipeline declined the whole
// family because it looked only for a crossing, and an intersector asked for the crossing between two
// identical surfaces answers that it cannot (ADR-0045, ADR-0061 stage 4).
//
// The union is one taller cylinder: the two wall bands share the rim where the first ends, and the two
// surviving caps close it.
func TestCoaxialWallsUnionThroughTheGeneralPath(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		baseZ, h float64
	}{
		{"overlapping bands", 3, 4},
		// Abutting cap to cap is the same union with a different contact: the walls touch at a rim
		// rather than overlapping, and the two discs at the join are COINCIDENT planes, each presenting
		// the other with a copy of its own boundary.
		{"abutting cap to cap", 4, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertCoaxialUnionIsOneCylinder(t, tc.baseZ, tc.h)
		})
	}
}

// assertCoaxialUnionIsOneCylinder unions the r=2 cylinder over z∈[0,4] with the one of height h based
// at baseZ, and pins the union as a single cylinder closed by two caps.
func assertCoaxialUnionIsOneCylinder(t *testing.T, baseZ, h float64) {
	t.Helper()
	a, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder a: %v", err)
	}
	b, err := SolidCylinder(math.P3(0, 0, math.Scalar(baseZ)), math.V3(0, 0, 1), 2, h)
	if err != nil {
		t.Fatalf("SolidCylinder b: %v", err)
	}
	res, err := Boolean(Union, a, b)
	if err != nil {
		t.Fatalf("Boolean(Union) on coaxial cylinders: %v", err)
	}
	assertWatertight(t, res)
	assertEveryFaceWinds(t, res)
	// The union of two coaxial cylinders IS one cylinder. Two bands meeting at a rim would be a valid
	// solid of the same volume carrying an edge the model does not have — the surface is smooth across
	// it — which a user could select and a fillet would try to round (mergeCoincidentFaces).
	walls := 0
	for _, f := range res.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); isCyl {
			walls++
		}
	}
	if walls != 1 {
		t.Errorf("the coaxial union has %d cylinder faces, want 1 (the two bands merge across the rim they share)", walls)
	}
	if n := len(res.Faces()); n != 3 {
		t.Errorf("%d faces, want 3 (one wall and the two surviving caps)", n)
	}
	// The wall spans z∈[0,7] — no gap, which would tear the solid, and no overlap, which would double
	// the surface where the two operands agree.
	assertWallSpansTile(t, res, 0, baseZ+h)
}

// assertWallSpansTile checks that the result's cylinder walls cover the axial range [lo, hi] exactly
// once. The spans are read from each band's WORLD anchors: a band's own v is measured from its own
// surface's base, so two walls on one cylinder both report [0, h] and only the world points place them.
func assertWallSpansTile(t *testing.T, res *topo.Body, lo, hi float64) {
	t.Helper()
	var spans [][2]float64
	for _, cf := range facesOfAny(res) {
		if rs, ok := ruledFaceOf(cf); ok {
			spans = append(spans, [2]float64{float64(rs.band.bottom.Z), float64(rs.band.top.Z)})
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i][0] < spans[j][0] })
	at := lo
	for _, s := range spans {
		if stdmath.Abs(s[0]-at) > 1e-9 {
			t.Fatalf("wall spans %v do not tile [%v, %v]: the next band starts at %v, not %v", spans, lo, hi, s[0], at)
		}
		at = s[1]
	}
	if stdmath.Abs(at-hi) > 1e-9 {
		t.Errorf("wall spans %v end at %v, want %v", spans, at, hi)
	}
}

// TestCoincidentWallImprintIsTheOtherBandsRims is the unit statement: the imprint between two walls on
// one surface is each band's rims where they fall strictly inside the other's, and nothing else.
func TestCoincidentWallImprintIsTheOtherBandsRims(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 2, 4)
	pa, pb := partitionFaces(a), partitionFaces(b)
	curves, ok := wallWallImprint(pa.wall[0], pb.wall[0])
	if !ok {
		t.Fatal("two coincident walls are undecided; their overlap is a region, and it is decidable")
	}
	if len(curves) != 2 {
		t.Fatalf("coincident walls imprinted %d curves, want 2 (b's bottom rim inside a, a's top rim inside b)", len(curves))
	}
	for _, cv := range curves {
		z := float64(cv.PointAt(0).Z)
		if stdmath.Abs(z-3) > 1e-9 && stdmath.Abs(z-4) > 1e-9 {
			t.Errorf("an imprint circle sits at z=%v; the bands meet only at z=3 and z=4", z)
		}
	}
}

// TestDisjointCoaxialWallsImprintNothing: two coaxial cylinders that do not overlap share a surface but
// no region, so the decided imprint is empty and neither wall is split.
func TestDisjointCoaxialWallsImprintNothing(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 9), math.V3(0, 0, 1), 2, 4)
	pa, pb := partitionFaces(a), partitionFaces(b)
	curves, ok := wallWallImprint(pa.wall[0], pb.wall[0])
	if !ok {
		t.Fatal("two disjoint coaxial walls are undecided; they plainly do not meet")
	}
	if len(curves) != 0 {
		t.Errorf("disjoint coaxial walls imprinted %d curves, want none", len(curves))
	}
}
