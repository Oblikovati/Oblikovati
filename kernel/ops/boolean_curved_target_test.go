// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/brep"
	"oblikovati/kernel/ops"
	"oblikovati/math"
)

// TestBooleanCutCurvedTargetRemovesTunnel is a regression for the boolean that
// did nothing when the *target* (minuend) was a curved body. classify() bounded
// the cylinder by its seam vertices only, judged it disjoint from the tool, and
// returned the target uncut. With curved-edge-aware RangeBox the cut runs: a box
// bored through a cylinder removes its volume.
//
// cyl(r=3.5, h=4) volume = pi*r^2*h = 153.94 (faceted slightly less); a 2x2 box
// tunnel through it removes 2*2*4 = 16.
func TestBooleanCutCurvedTargetRemovesTunnel(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3.5, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	full := ops.BodyGeometryProperties(cyl, ops.DefaultQuality()).Volume
	tool := csgBox(math.P3(-1, -1, -1), 2, 2, 6) // 2x2 cross-section, pokes through both caps

	res, err := ops.Boolean(ops.Cut, cyl, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("cut result not a valid solid: %+v", r)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if want := full - 16; stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("curved-target cut volume = %.5f, want %.5f (full %.5f − 16)", got, want, full)
	}
}
