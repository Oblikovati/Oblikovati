// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// TestJointGapSeatsOrigins the joint's gap moves the axial seating between the two origins (#1970):
// a rotational joint's axial-gap residual shifts by exactly the gap, so origins that would sit
// coincident are instead held that far apart along the joint axis.
func TestJointGapSeatsOrigins(t *testing.T) {
	a := LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))
	b := LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))
	id := math.Identity4()
	r0 := jointResiduals(types.JointRotational, a, b, id, id, false, 0)
	r2 := jointResiduals(types.JointRotational, a, b, id, id, false, 2)
	if len(r0) == 0 || len(r0) != len(r2) {
		t.Fatalf("residual counts %d/%d", len(r0), len(r2))
	}
	last := len(r0) - 1
	if d := stdmath.Abs(r2[last] - r0[last]); stdmath.Abs(d-2) > 1e-9 {
		t.Errorf("gap did not move the axial seating: Δ=%.6f, want 2", d)
	}
}

// TestLockedJointReportsZeroDOF locking a rotational joint drops its free DOF to 0 in the report
// (without suppressing it); unlocking restores it, and protected is independent metadata (#1974).
func TestLockedJointReportsZeroDOF(t *testing.T) {
	j := &jointBase{kind: types.JointRotational}
	if j.DegreesOfFreedom() != 1 {
		t.Fatalf("rotational nominal DOF = %d, want 1", j.DegreesOfFreedom())
	}
	j.setLocked(true)
	if j.DegreesOfFreedom() != 0 || !j.Locked() || j.Suppressed() {
		t.Errorf("locked joint: DOF=%d locked=%v suppressed=%v, want 0/true/false", j.DegreesOfFreedom(), j.Locked(), j.Suppressed())
	}
	j.setLocked(false)
	if j.DegreesOfFreedom() != 1 {
		t.Errorf("unlocked DOF = %d, want 1", j.DegreesOfFreedom())
	}
	j.setProtected(true)
	if !j.Protected() || j.Locked() {
		t.Error("protected must be independent of locked")
	}
}
