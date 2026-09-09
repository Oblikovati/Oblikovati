// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestAnnularRingCutClosesTheBox is the corpus row for the band's own axial origin (ADR-0061). A
// rectangle revolved a full turn is an annular ring — two COAXIAL cylinders and two annular caps — and
// the assembly revolve machines a participant with exactly that tool.
//
// The ring's inner wall is REVERSED, so it frames itself from its far rim and its band reports
// [-1, 0] rather than [0, 1]. bandBase cancelled to the bottom rim itself, which is where the band
// coordinate reads vMin and not zero, so the axial window clipped that wall's imprints over the band
// BELOW it: the wall kept no fragment and the cut came back with eight unpaired edges. The outer wall,
// whose band starts at zero, was unaffected — which is why only half the tool went missing.
func TestAnnularRingCutClosesTheBox(t *testing.T) {
	t.Parallel()
	box, err := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 4), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	ring, err := SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 1, 0),
		[]math.Point2{math.P2(0.5, 0.5), math.P2(1.5, 0.5), math.P2(1.5, 1.5), math.P2(0.5, 1.5)}, "ring")
	if err != nil {
		t.Fatalf("SolidOfRevolution: %v", err)
	}
	res, err := BooleanDiag(Difference, box, ring, nil)
	if err != nil {
		t.Fatalf("the general pipeline declined a box cut by an annular ring: %v", err)
	}
	if res == nil {
		t.Fatal("the ring cut returned an empty body")
	}
	for _, e := range res.Edges() {
		if n := len(e.Uses()); n != 2 {
			t.Errorf("edge %v has %d uses, want 2 — the ring cut must close", e.Lineage(), n)
		}
	}
	// BOTH of the ring's walls bound the removed quarter, so both survive as analytic cylinders.
	walls := 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			walls++
		}
	}
	if walls != 2 {
		t.Errorf("the ring cut kept %d analytic cylinder walls, want 2 (the ring's inner and outer)", walls)
	}
}
