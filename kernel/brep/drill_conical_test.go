// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/brep"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
)

// A blind hole with a conical drill point: a Ø2 bore 2 deep closed by a 118° cone tip in a
// 10×10×6 slab. Valid watertight solid; removed = cylinder + cone-tip volume.
func TestCutBlindConicalHole(t *testing.T) {
	const half = 59.0 * stdmath.Pi / 180 // 118° included drill point
	d, err := brep.CutBlindConicalHole(box(0, 0, 0, 10, 10, 6), math.P3(5, 5, 6), math.V3(0, 0, -1), 1, 2, half)
	if err != nil {
		t.Fatalf("CutBlindConicalHole: %v", err)
	}
	if r := ops.Validate(d); !r.Valid || !d.IsSolid() {
		t.Fatalf("conical-drilled slab is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(d); len(open) != 0 {
		t.Fatalf("conical hole has %d boundary edges, want 0 (watertight)", len(open))
	}
	if n := countConeFaces(d); n != 1 {
		t.Errorf("conical hole has %d cone faces, want 1 (the drill point)", n)
	}
	if n := countCylFaces(d); n != 1 {
		t.Errorf("conical hole has %d cylinder faces, want 1 (the bore)", n)
	}
	// Removed = cylinder (π·r²·2) + cone tip (⅓·π·r²·tipHeight), tipHeight = r/tan(half).
	tip := 1.0 / stdmath.Tan(half)
	want := stdmath.Pi*1*1*2 + stdmath.Pi*1*1*tip/3
	removed := 10.0*10.0*6.0 - vol(d)
	if removed <= 0 || removed > want+1e-9 || (want-removed)/want > 0.04 {
		t.Errorf("removed volume = %g, want a hair under %g (cylinder + cone tip)", removed, want)
	}
}

// A conical point deep enough to exit the part is rejected.
func TestCutBlindConicalRejectsDeepTip(t *testing.T) {
	const half = 59.0 * stdmath.Pi / 180
	_, err := brep.CutBlindConicalHole(box(0, 0, 0, 10, 10, 3), math.P3(5, 5, 3), math.V3(0, 0, -1), 1, 3, half)
	if err == nil {
		t.Error("expected an error for a conical hole that exits the part, got nil")
	}
}
