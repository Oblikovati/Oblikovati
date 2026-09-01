// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	stdmath "math"
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
)

// TestJointEventsAndCombinedSolve checks that authoring a joint raises the JointAdd event and
// that SolveConstraints positions a jointed component through the combined constraint+joint
// solve (M12-F02).
func TestJointEventsAndCombinedSolve(t *testing.T) {
	t.Parallel()
	asm := NewAssemblyComponentDefinition()
	base := asm.Place("base:1", NewPartComponentDefinition(), math.Identity4())
	base.SetGrounded(true)
	moving := asm.Place("moving:1", NewPartComponentDefinition(), math.Translation4(math.V3(3, 4, 6)))

	var added int
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, _ JointAdd) event.Outcome {
		added++
		return event.Continue()
	})

	zAxis, _ := math.NewUnitVector3(0, 0, 1)
	j := asm.Joints().AddRotational(
		assembly.Ref{Occurrence: base, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), zAxis)},
		assembly.Ref{Occurrence: moving, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), zAxis)})

	rep := asm.SolveConstraints()
	if !rep.Converged {
		t.Fatalf("combined solve did not converge: %+v", rep)
	}
	// The rotational joint seats the moving origin on the grounded axis (x,y,z → 0).
	if p := moving.Transform().Translation(); stdmath.Abs(p.X) > 1e-6 || stdmath.Abs(p.Y) > 1e-6 || stdmath.Abs(p.Z) > 1e-6 {
		t.Errorf("moving origin = %+v, want on the axis at the origin", p)
	}
	if added != 1 {
		t.Errorf("JointAdd events = %d, want 1", added)
	}
	if !asm.Joints().Delete(j.ID()) {
		t.Error("Delete returned false for a known joint")
	}
}
