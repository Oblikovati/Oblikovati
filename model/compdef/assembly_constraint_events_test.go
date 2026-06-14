// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
)

// TestConstraintEventsAndSolve checks that authoring and solving constraints on an
// assembly definition raises the relationship events (ConstraintAdd / AssemblyResolved /
// ConstraintDelete) and that SolveConstraints repositions a free component (M12-F01).
func TestConstraintEventsAndSolve(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	base := asm.Place("base:1", NewPartComponentDefinition(), math.Identity4())
	base.SetGrounded(true)
	moving := asm.Place("moving:1", NewPartComponentDefinition(), math.Translation4(math.V3(0, 0, 10)))

	var added, resolved, deleted int
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, _ ConstraintAdd) event.Outcome {
		added++
		return event.Continue()
	})
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, _ AssemblyResolved) event.Outcome {
		resolved++
		return event.Continue()
	})
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, _ ConstraintDelete) event.Outcome {
		deleted++
		return event.Continue()
	})

	zUp, _ := math.NewUnitVector3(0, 0, 1)
	zDown, _ := math.NewUnitVector3(0, 0, -1)
	m := asm.Constraints().AddMate(
		assembly.Ref{Occurrence: base, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zUp)},
		assembly.Ref{Occurrence: moving, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zDown)},
		0, types.MateSolutionOpposed)

	rep := asm.SolveConstraints()
	if !rep.Converged {
		t.Fatalf("solve did not converge: %+v", rep)
	}
	if z := moving.Transform().Translation().Z; stdmath.Abs(z) > 1e-6 {
		t.Errorf("moving component z = %v, want ~0 (faces coincident)", z)
	}
	if !asm.Constraints().Delete(m.ID()) {
		t.Fatal("Delete returned false for a known constraint")
	}
	if added != 1 || resolved != 1 || deleted != 1 {
		t.Errorf("events: added=%d resolved=%d deleted=%d, want 1/1/1", added, resolved, deleted)
	}
}
