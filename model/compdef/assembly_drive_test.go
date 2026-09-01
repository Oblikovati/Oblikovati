// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
)

// TestDriveJointThroughAssembly checks the assembly definition drives one of its joints —
// sweeping the rotational variable and returning motion frames — and leaves the assembly in
// its pre-drive pose (M12-F03).
func TestDriveJointThroughAssembly(t *testing.T) {
	t.Parallel()
	asm := NewAssemblyComponentDefinition()
	base := asm.Place("base:1", NewPartComponentDefinition(), math.Identity4())
	base.SetGrounded(true)
	moving := asm.Place("moving:1", NewPartComponentDefinition(), math.Translation4(math.V3(0, 0, 6)))

	zAxis, _ := math.NewUnitVector3(0, 0, 1)
	j := asm.Joints().AddRotational(
		assembly.Ref{Occurrence: base, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), zAxis)},
		assembly.Ref{Occurrence: moving, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), zAxis)})

	before := moving.Transform()
	span := stdmath.Pi / 2
	res, err := asm.DriveJoint(j.ID(), assembly.NewDriveSettings(types.DriveAngular, 0, span, span/4, 1, false, false))
	if err != nil {
		t.Fatalf("DriveJoint: %v", err)
	}
	if len(res.Frames) != 5 {
		t.Fatalf("frames = %d, want 5", len(res.Frames))
	}
	if !moving.Transform().IsEqualTo(before, 1e-9) {
		t.Errorf("assembly not restored after drive")
	}

	if _, err := asm.DriveJoint(404, assembly.NewDriveSettings(types.DriveAngular, 0, 1, 0.5, 1, false, false)); err == nil {
		t.Error("DriveJoint(unknown id) = nil error, want failure")
	}
}
