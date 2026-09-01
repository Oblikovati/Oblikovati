// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// TestSetFlexibleChildOverWire checks the M12-F06 independent solve over the wire: a flexible
// sub-assembly occurrence positions its child independently, and a non-flexible occurrence is
// rejected (#822).
func TestSetFlexibleChildOverWire(t *testing.T) {
	t.Parallel()
	r, s, asm, _ := assemblySessionWithBoxes(t) // empty parent assembly

	subDef := compdef.NewAssemblyComponentDefinition()
	subDef.Place("p:1", blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Identity4())
	flex := asm.Place("sub:1", subDef, math.Identity4())
	flex.SetFlexible(true)

	m := types.Matrix{Cells: math.Translation4(math.V3(0, 0, 7)).Cells()}
	var res wire.OccurrenceResult
	call(t, r, s, "assembly.setFlexibleChild", mustJSON(t, wire.SetFlexibleChildArgs{Occurrence: flex.ID(), Child: "p:1", Transform: m}), &res)
	if got, ok := flex.ChildTransform("p:1"); !ok || got.Translation().Z != 7 {
		t.Errorf("flexible child override = %v (ok=%v), want a z=7 placement", got, ok)
	}

	// A non-flexible occurrence is rejected.
	rigid := asm.Place("sub:2", subDef, math.Identity4())
	args := mustJSON(t, wire.SetFlexibleChildArgs{Occurrence: rigid.ID(), Child: "p:1", Transform: m})
	if _, err := r.Handle(s, "assembly.setFlexibleChild", []byte(args)); err == nil {
		t.Error("setFlexibleChild on a non-flexible occurrence should error")
	}

	// An unknown child name is rejected.
	bad := mustJSON(t, wire.SetFlexibleChildArgs{Occurrence: flex.ID(), Child: "nope", Transform: m})
	if _, err := r.Handle(s, "assembly.setFlexibleChild", []byte(bad)); err == nil {
		t.Error("setFlexibleChild with an unknown child name should error")
	}
}
