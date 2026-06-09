// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func splitBox(sx, sy, sz float64) *topo.Body {
	return subd.ToBody(subd.Box(sx, sy, sz), "box")
}

// Splitting a 4×4×4 box by the mid-plane z=2 yields two valid 4×4×2 solids (vol 32 each).
func TestSplitSolidByMidPlane(t *testing.T) {
	plane, _ := geom.NewPlane(math.P3(0, 0, 2), math.V3(0, 0, 1))
	pieces, err := ops.SplitSolidByPlane(splitBox(4, 4, 4), plane)
	if err != nil {
		t.Fatalf("SplitSolidByPlane: %v", err)
	}
	if len(pieces) != 2 {
		t.Fatalf("split yielded %d pieces, want 2", len(pieces))
	}
	for i, p := range pieces {
		if r := ops.Validate(p); !r.Valid || !p.IsSolid() {
			t.Fatalf("piece %d not a valid solid: %+v", i, r)
		}
		if v := ops.BodyGeometryProperties(p, ops.DefaultQuality()).Volume; stdmath.Abs(v-32) > 1e-6 {
			t.Errorf("piece %d volume = %g, want 32 (half the box)", i, v)
		}
	}
}

// A plane clear of the body leaves it whole (one piece).
func TestSplitPlaneMissesBody(t *testing.T) {
	plane, _ := geom.NewPlane(math.P3(0, 0, 100), math.V3(0, 0, 1))
	pieces, err := ops.SplitSolidByPlane(splitBox(2, 2, 2), plane)
	if err != nil {
		t.Fatalf("SplitSolidByPlane: %v", err)
	}
	if len(pieces) != 1 {
		t.Errorf("split by a missing plane gave %d pieces, want 1 (whole body)", len(pieces))
	}
}

// Splitting by an oblique plane still gives two valid solids whose volumes sum to the original.
func TestSplitSolidByObliquePlane(t *testing.T) {
	plane, _ := geom.NewPlane(math.P3(2, 2, 2), math.V3(1, 0, 1))
	pieces, err := ops.SplitSolidByPlane(splitBox(4, 4, 4), plane)
	if err != nil {
		t.Fatalf("SplitSolidByPlane: %v", err)
	}
	if len(pieces) != 2 {
		t.Fatalf("oblique split yielded %d pieces, want 2", len(pieces))
	}
	sum := 0.0
	for _, p := range pieces {
		if r := ops.Validate(p); !r.Valid || !p.IsSolid() {
			t.Fatalf("oblique piece not a valid solid: %+v", r)
		}
		sum += ops.BodyGeometryProperties(p, ops.DefaultQuality()).Volume
	}
	if stdmath.Abs(sum-64) > 1e-6 {
		t.Errorf("oblique split volumes sum to %g, want 64 (the whole box)", sum)
	}
}
