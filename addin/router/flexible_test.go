// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// TestSetFlexibleOverWire checks the M12-F06 flexible flag over the wire: a sub-assembly
// occurrence becomes flexible (independent per-placement solve), a leaf part stays rigid (#822).
func TestSetFlexibleOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)

	// A leaf part cannot be flexible — the flag stays off.
	var leaf wire.OccurrenceResult
	call(t, r, s, "assembly.setFlexible", mustJSON(t, wire.SetFlexibleOccurrenceArgs{ID: occs[0].ID(), Flexible: true}), &leaf)
	if leaf.Occurrence.Flexible {
		t.Error("a leaf part occurrence should not become flexible")
	}

	// A sub-assembly occurrence can.
	sub := asm.Place("sub:1", compdef.NewAssemblyComponentDefinition(), math.Identity4())
	var subRes wire.OccurrenceResult
	call(t, r, s, "assembly.setFlexible", mustJSON(t, wire.SetFlexibleOccurrenceArgs{ID: sub.ID(), Flexible: true}), &subRes)
	if !subRes.Occurrence.Flexible {
		t.Error("a sub-assembly occurrence should become flexible over the wire")
	}

	// Turn it off again (fresh result var: Flexible is omitempty, so a false reply omits it and
	// would otherwise leave a reused var's stale true in place).
	var off wire.OccurrenceResult
	call(t, r, s, "assembly.setFlexible", mustJSON(t, wire.SetFlexibleOccurrenceArgs{ID: sub.ID(), Flexible: false}), &off)
	if off.Occurrence.Flexible || sub.Flexible() {
		t.Error("setFlexible(false) should clear the flag")
	}
}
