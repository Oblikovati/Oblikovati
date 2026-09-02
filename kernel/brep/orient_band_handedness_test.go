// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// handednessAgreesWithStored reports the faces whose loop-derived handedness contradicts the sense the
// body actually stores. On a valid body there are none: that agreement is what lets a shell's traversal
// consistency imply its faces agree about which side holds material.
func handednessAgreesWithStored(t *testing.T, b *topo.Body) []string {
	t.Helper()
	var bad []string
	for _, f := range facesOfAny(b) {
		wantReversed := loopHandedness(faceTrimRegion(f)) < 0
		if wantReversed != f.reversed {
			bad = append(bad, string(f.lineage.Key()))
		}
	}
	return bad
}

// TestBandHandednessReadsItsWholeCircuit pins Oblikovati/Oblikovati#3506. A band's rims are closed
// circuits in 3-D but OPEN polylines in the covering space, and a rim at constant v shoelaces to ZERO
// whichever way the band is oriented — so reading rings[0]'s area gave a plate's bore wall the
// handedness of its own opposite. That is masked in most bodies by the global signed-volume flip
// orientFaceSigns applies afterwards, so it has to be tested on loopHandedness directly.
func TestBandHandednessReadsItsWholeCircuit(t *testing.T) {
	t.Parallel()
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	plate, err := SolidBlock(math.P3(-8, -8, 1), math.P3(8, 8, 3), "plate")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	bored, err := Boolean(Difference, plate, cyl)
	if err != nil {
		t.Fatalf("Boolean(Difference, plate, cylinder): %v", err)
	}
	if bad := handednessAgreesWithStored(t, bored); len(bad) > 0 {
		t.Errorf("these faces' loop-derived handedness contradicts their stored sense: %v; a bore wall's "+
			"two rims bound the band together with the seam, so the area must be taken over that whole "+
			"circuit rather than over one rim", bad)
	}
	// The primitive's own side face carries its seam explicitly, so it is one closed circuit already
	// and must NOT be treated as a wrapping rim.
	if bad := handednessAgreesWithStored(t, cyl); len(bad) > 0 {
		t.Errorf("a plain cylinder's faces disagree with their stored sense: %v", bad)
	}
}

// TestTubeInteriorClassifiesInside is the containment half. Every face of a tube is a band or an
// annulus, so the global volume flip cannot rescue a misread rim — if the handedness is wrong the
// winding number is wrong, and a point in the wall reads outside the solid.
func TestTubeInteriorClassifiesInside(t *testing.T) {
	t.Parallel()
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	plate, err := SolidBlock(math.P3(-8, -8, 1), math.P3(8, 8, 3), "plate")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	bored, err := Boolean(Difference, plate, cyl)
	if err != nil {
		t.Fatalf("Boolean(Difference, plate, cylinder): %v", err)
	}
	for _, tc := range []struct {
		p    math.Point3
		want bool
		what string
	}{
		{math.P3(6.5, 0, 2), true, "in the plate's material, outside the bore"},
		{math.P3(0, 0, 2), false, "on the axis, inside the bore"},
		{math.P3(4.9, 0, 2), false, "just inside the bore wall"},
		{math.P3(20, 0, 2), false, "clear of the plate"},
	} {
		if got := PointInside(bored, tc.p); got != tc.want {
			t.Errorf("PointInside(%v) = %v, want %v — %s", tc.p, got, tc.want, tc.what)
		}
	}
}
