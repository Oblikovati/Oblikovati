// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
)

// TestSectionCylinderPerpendicularIsCircle exercises the analytic curved-face section path (#3473): a plane
// perpendicular to a cylinder's axis cuts its wall in an exact circle, discretized to a closed section wire
// whose length matches the analytic circumference — a triangle-sliced mesh could only chord it more coarsely
// and its accuracy would ride the tessellation Quality; the analytic curve does not.
func TestSectionCylinderPerpendicularIsCircle(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	sec, err := SectionWithPlane(cyl, math.P3(0, 0, 2), math.V3(0, 0, 1), DefaultQuality())
	if err != nil {
		t.Fatalf("SectionWithPlane: %v", err)
	}
	wires := sec.Wires()
	if len(wires) != 1 {
		t.Fatalf("cylinder mid-section has %d wires, want 1 (the circle)", len(wires))
	}
	if !wires[0].IsClosed() {
		t.Error("cylinder section circle must be closed")
	}
	want := 2 * stdmath.Pi * 2
	if l := wireLength(wires[0]); stdmath.Abs(l-want)/want > 0.01 {
		t.Errorf("section circumference = %g, want ~%g (rel %+.4f)", l, want, (l-want)/want)
	}
}
