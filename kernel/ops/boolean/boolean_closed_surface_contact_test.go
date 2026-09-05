// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
)

// A section that CROSSES the receiving face's own boundary is contact, and the pairing must say so.
// It used to ask whether the section sat WHOLLY inside the receiver's trim, so a crossing read as "no
// contact at all": nothing was imprinted, and a sphere intersected with a box passed through WHOLE — a
// valid solid of entirely the wrong shape, which is worse than any decline (ADR-0061 stage 4).
//
// The pairing that CARRIES such a crossing is still to come; until it does, the honest answer is the
// named decline, and this pins that it is a decline rather than a wrong body.
func TestASectionCrossingItsReceiverDeclinesRatherThanPassingWhole(t *testing.T) {
	t.Parallel()
	sphere, err := brep.SolidSphere(math.P3(0, 0, 0), 5, "s")
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	// The box keeps x ≤ 2 and z ≤ 0; both section circles leave through their own face's edge.
	box, err := brep.SolidBlock(math.P3(-10, -10, -10), math.P3(2, 10, 0), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	res, err := brep.Boolean(brep.Intersection, sphere, box)
	if err == nil {
		t.Fatalf("the mixed boolean answered with %d faces where it does not model the contact; want the named decline",
			len(res.Faces()))
	}
	if !errors.Is(err, brep.ErrUnsupportedMixedBoolean) {
		t.Errorf("declined with %v, want ErrUnsupportedMixedBoolean", err)
	}
}
