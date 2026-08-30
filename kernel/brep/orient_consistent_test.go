// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestFluxOrientationIndependentOfReversedFlag is the regression gate for orientFaceSigns: the winding
// classifier must derive each face's outward orientation from the loop geometry, NOT the stored
// Face.Reversed flag — the property that makes it correct on an imported B-rep with inconsistent
// normal-side flags (Oblikovati#3427, the J6/T5 elliptic hosts). Flipping EVERY face's reversed flag (the
// worst inconsistency) must leave the inside/outside verdict unchanged.
func TestFluxOrientationIndependentOfReversedFlag(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	faces := facesOfAny(cyl)
	flipped := make([]curvedFace, len(faces))
	for i, f := range faces {
		f.reversed = !f.reversed
		flipped[i] = f
	}
	probes := []struct {
		p    math.Point3
		want bool
	}{
		{math.P3(0, 0, 2), true},   // axis interior
		{math.P3(1.5, 0, 2), true}, // off-axis interior
		{math.P3(3, 0, 2), false},  // outside radius
		{math.P3(0, 0, 5), false},  // above top
	}
	base, flip := newFluxQuery(faces), newFluxQuery(flipped)
	for _, pr := range probes {
		if got := base.inside(pr.p); got != pr.want {
			t.Errorf("base flux inside(%v) = %v, want %v", pr.p, got, pr.want)
		}
		if got := flip.inside(pr.p); got != pr.want {
			t.Errorf("reversed-flag-flipped flux inside(%v) = %v, want %v (orientation must come from loop geometry)", pr.p, got, pr.want)
		}
	}
}

// TestSignedVolumeGlobalFlip checks the global orientation safety net: orientFaceSigns must sign the faces
// so the enclosed signed volume is positive (outward), so a sphere's interior gives a positive winding.
func TestSignedVolumeGlobalFlip(t *testing.T) {
	sph, err := SolidSphere(math.P3(0, 0, 0), 3, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	q := newFluxQuery(facesOfAny(sph))
	if !q.inside(math.P3(0, 0, 0)) {
		t.Error("sphere centre classified outside: the global volume sign did not orient the shell outward")
	}
}
