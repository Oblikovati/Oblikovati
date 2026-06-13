// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestAssemblyPatternCircularOverWire replicates a seed component four-up around the Z
// axis: the pattern adds three new occurrences (elements 1–3 beyond the seed), so the
// tree grows from one to four.
func TestAssemblyPatternCircularOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0)

	var created wire.NewOccurrencesResult
	args := fmt.Sprintf(`{"seed":%d,"kind":"circular","origin":[0,0,0],"axis":[0,0,1],"angle":%g,"count":4}`, occs[0].ID(), stdmath.Pi/2)
	call(t, r, s, "assembly.patternCreate", args, &created)
	if len(created.Created) != 3 {
		t.Fatalf("circular pattern created %d occurrences, want 3 (4 total minus the seed)", len(created.Created))
	}

	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 4 {
		t.Errorf("tree = %d occurrences, want 4 after the 4-up pattern", len(tree.Occurrences))
	}

	if _, err := r.Handle(s, "assembly.patternCreate", []byte(fmt.Sprintf(`{"seed":%d,"kind":"helical","count":2}`, occs[0].ID()))); err == nil {
		t.Error("an unknown pattern kind should fail")
	}
}

// TestAssemblyMirrorOverWire mirrors a component (placed at x=2) across the YZ plane
// through the origin: the new occurrence sits at x=-2.
func TestAssemblyMirrorOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 2)

	var mirrored wire.NewOccurrencesResult
	args := fmt.Sprintf(`{"sources":[%d],"origin":[0,0,0],"normal":[1,0,0]}`, occs[0].ID())
	call(t, r, s, "assembly.mirror", args, &mirrored)
	if len(mirrored.Created) != 1 {
		t.Fatalf("mirror created %d occurrences, want 1", len(mirrored.Created))
	}
	if got := mirrored.Created[0].Transform.Cells[3]; got != -2 {
		t.Errorf("mirror x-translation = %g, want -2 (reflected across x=0)", got)
	}

	if _, err := r.Handle(s, "assembly.mirror", []byte(`{"sources":[99999],"origin":[0,0,0],"normal":[1,0,0]}`)); err == nil {
		t.Error("mirror of an unknown occurrence id should fail")
	}
}

// TestAssemblyCopyOverWire copies a component, producing an independent occurrence at the
// same placement.
func TestAssemblyCopyOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 3)

	var copied wire.NewOccurrencesResult
	call(t, r, s, "assembly.copy", fmt.Sprintf(`{"sources":[%d]}`, occs[0].ID()), &copied)
	if len(copied.Created) != 1 {
		t.Fatalf("copy created %d occurrences, want 1", len(copied.Created))
	}
	if copied.Created[0].ID == occs[0].ID() || copied.Created[0].Transform.Cells[3] != 3 {
		t.Errorf("copy = %+v, want a new occurrence at the source's placement (x=3)", copied.Created[0])
	}
}

// TestAssemblySubstituteOverWire substitutes two components with one simplified part: the
// substitute occurrence is flagged, and the sources are suppressed.
func TestAssemblySubstituteOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	lod := openPartDoc(t, s, "lod.obk")

	var sub wire.OccurrenceResult
	args := fmt.Sprintf(`{"sources":[%d,%d],"document":%d,"name":"lod:1","transform":%s}`,
		occs[0].ID(), occs[1].ID(), uint64(lod.ID()), transformJSON(0, 0, 0))
	call(t, r, s, "assembly.substitute", args, &sub)
	if sub.Occurrence.Name != "lod:1" || !sub.Occurrence.Substitute {
		t.Fatalf("substitute = %+v, want a substitute occurrence named lod:1", sub.Occurrence)
	}
	if !occs[0].Suppressed() || !occs[1].Suppressed() {
		t.Error("substituted sources should be suppressed")
	}

	if _, err := r.Handle(s, "assembly.substitute", []byte(`{"sources":[99999],"document":1,"name":"x","transform":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}`)); err == nil {
		t.Error("substitute of an unknown source id should fail")
	}
}
