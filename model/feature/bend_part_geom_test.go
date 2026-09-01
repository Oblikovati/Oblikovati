// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/math"
)

// TestBendSolidBoxFoldsUpValid bends a long flat bar 90° at its middle and checks the
// result is a single valid solid whose moving flange folded up (the +Z extent grew well
// beyond the original thickness) while volume is conserved within the bend allowance.
func TestBendSolidBoxFoldsUpValid(t *testing.T) {
	t.Parallel()
	bar := subd.ToBody(subd.Box(10, 2, 1), "bar") // L=10 (X), W=2 (Y), T=1 (Z), corner at origin
	before := query.BodyGeometryProperties(bar, ops.DefaultQuality()).Volume

	// Bend line across the bar at x=5 on the top face, along +Y; fold the +X half up.
	bent, err := bendSolid(bar, math.P3(5, 0, 1), math.V3(0, 1, 0), math.V3(0, 0, 1), 1.0, stdmath.Pi/2, "bend")
	if err != nil {
		t.Fatalf("bendSolid: %v", err)
	}
	if r := ops.Validate(bent); !r.Valid {
		t.Fatalf("bent body invalid: %v", r.Issues)
	}
	props := query.BodyGeometryProperties(bent, ops.DefaultQuality())
	if props.Volume <= 0 {
		t.Fatalf("bent volume = %g, want positive", props.Volume)
	}
	// Volume is roughly conserved (the arc adds/removes a little material vs. a sharp fold).
	if rel := stdmath.Abs(props.Volume-before) / before; rel > 0.25 {
		t.Errorf("bent volume %g vs original %g differ by %.0f%%, want <25%%", props.Volume, before, rel*100)
	}
	box := bent.RangeBox()
	if box.Max.Z < 4 { // a 5-long flange folded 90° up should reach well above the 1.0 thickness
		t.Errorf("bent top Z = %g, want the moving flange folded up (>4)", box.Max.Z)
	}
}
