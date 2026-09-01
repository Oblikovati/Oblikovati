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
	t.Parallel()
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	faces := facesOfAny(cyl)
	box := cyl.RangeBox()
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
		if got := base.inside(pr.p, box); got != pr.want {
			t.Errorf("base flux inside(%v) = %v, want %v", pr.p, got, pr.want)
		}
		if got := flip.inside(pr.p, box); got != pr.want {
			t.Errorf("reversed-flag-flipped flux inside(%v) = %v, want %v (orientation must come from loop geometry)", pr.p, got, pr.want)
		}
	}
}

// TestSignedVolumeGlobalFlip checks the global orientation safety net: orientFaceSigns must sign the faces
// so the enclosed signed volume is positive (outward), so a sphere's interior gives a positive winding.
func TestSignedVolumeGlobalFlip(t *testing.T) {
	t.Parallel()
	sph, err := SolidSphere(math.P3(0, 0, 0), 3, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	q := newFluxQuery(facesOfAny(sph))
	if !q.inside(math.P3(0, 0, 0), sph.RangeBox()) {
		t.Error("sphere centre classified outside: the global volume sign did not orient the shell outward")
	}
}

// TestPointInsideFullCone is the regression gate for the cone-apex slit (Oblikovati/Oblikovati#3447).
// A full cone's side face is bounded by ONE loop that runs down the seam, round the base rim and back
// up the SAME seam ruling, meeting itself at the apex — a parametric pole. Before bridgePoleBranch the
// (u, v) unwrap pinned both seam sides to one branch, so the face's trim polygon enclosed nothing, its
// volume term came out ≈0, orientFaceSigns read the body's signed volume as negative and flipped every
// face outward-sign — and the ray classifier then reported every INTERIOR point of the cone as outside.
func TestPointInsideFullCone(t *testing.T) {
	t.Parallel()
	cone, err := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 6, 8), 3, 0, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	// Points named by the cone's own radius law: at axial station t the radius is 3(1 − t/10).
	for _, c := range []struct {
		name string
		p    math.Point3
		want bool
	}{
		{"on the axis a quarter up", math.P3(0, 1.2, 1.6), true},
		{"on the axis halfway up", math.P3(0, 3, 4), true},
		{"just inside the wall halfway up", math.P3(1.4, 3, 4), true},
		{"just outside the wall halfway up", math.P3(1.6, 3, 4), false},
		{"beyond the apex", math.P3(0, 6.6, 8.8), false},
		{"below the base", math.P3(0, -0.6, -0.8), false},
	} {
		if got := PointInside(cone, c.p); got != c.want {
			t.Errorf("PointInside(full cone, %s %v) = %v, want %v", c.name, c.p, got, c.want)
		}
	}
}

// TestFullConeFaceSignsAreOutward pins the mechanism the test above gates: every prepared face of a
// full cone must carry the +1 (outward) sign, and the body's signed volume term must be positive.
func TestFullConeFaceSignsAreOutward(t *testing.T) {
	t.Parallel()
	cone, err := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	q := newFluxQuery(facesOfAny(cone))
	if len(q.faces) != 2 {
		t.Fatalf("prepared %d faces, want 2 (side + base cap)", len(q.faces))
	}
	total := 0.0
	for i := range q.faces {
		if q.faces[i].sign != 1 {
			t.Errorf("face %d (%T) sign = %v, want +1 (outward)", i, q.faces[i].cf.surface, q.faces[i].sign)
		}
		total += q.faces[i].sign * faceVolumeTerm(&q.faces[i])
	}
	if total <= 0 {
		t.Errorf("body signed volume term = %g, want > 0", total)
	}
}
