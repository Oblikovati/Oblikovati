// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The reconstructed boolean must store PLANAR faces with the surface normal pointing OUTWARD
// (reversed=false), the same convention brep.BooleanDiag upholds — many consumers read a planar
// face's surface normal AS its outward normal (boss, hole, chamfer-corner, thicken, flat-pattern,
// pick), so a reversed planar face silently inverts them (#2247).

func planeZ(t *testing.T) geom.Plane {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1)) // surface normal +Z
	if err != nil {
		t.Fatal(err)
	}
	return pl
}

// TestCanonicalPlanarOutwardFlipsReversedPlane: a reversed planar face is rewritten to an outward
// plane (normal negated over the same point-set) with reversed cleared — the outward normal is
// preserved (was -Z), now the stored surface normal.
func TestCanonicalPlanarOutwardFlipsReversedPlane(t *testing.T) {
	cf := canonicalPlanarOutward(curvedFace{surface: planeZ(t), reversed: true})
	if cf.reversed {
		t.Fatal("a planar face must be canonicalized to reversed=false")
	}
	if n := cf.surface.NormalAt(0, 0); n.Z >= 0 {
		t.Fatalf("surface normal must flip to the outward -Z, got %v", n)
	}
}

// TestCanonicalPlanarOutwardKeepsUnreversedPlane: an already-outward plane is untouched.
func TestCanonicalPlanarOutwardKeepsUnreversedPlane(t *testing.T) {
	pl := planeZ(t)
	cf := canonicalPlanarOutward(curvedFace{surface: pl, reversed: false})
	if cf.reversed {
		t.Fatal("reversed must stay false")
	}
	if n := cf.surface.NormalAt(0, 0); n.Z <= 0 {
		t.Fatalf("surface normal must stay +Z, got %v", n)
	}
}

// TestCanonicalPlanarOutwardKeepsReversedCurved: a reversed CURVED face keeps reversed — an
// inward-facing cylinder (a bore wall) has no negated-normal surface, so reversed IS its form.
func TestCanonicalPlanarOutwardKeepsReversedCurved(t *testing.T) {
	cyl := geom.Cylinder{Origin: math.P3(0, 0, 0), AxisDir: math.V3(0, 0, 1).AsUnit(), Radius: 1}
	cf := canonicalPlanarOutward(curvedFace{surface: cyl, reversed: true})
	if !cf.reversed {
		t.Fatal("a reversed curved face must keep reversed=true (a bore wall's legitimate form)")
	}
}
