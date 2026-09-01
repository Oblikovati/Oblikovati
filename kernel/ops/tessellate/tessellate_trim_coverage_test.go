// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func TestEmitCellOutwardFlipsInwardCell(t *testing.T) {
	t.Parallel()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	m := &Mesh{}
	a := m.AddVertex(math.P3(0, 0, 0), math.V3(0, 0, 1))
	b := m.AddVertex(math.P3(0, 1, 0), math.V3(0, 0, 1))
	c := m.AddVertex(math.P3(1, 1, 0), math.V3(0, 0, 1))
	d := m.AddVertex(math.P3(1, 0, 0), math.V3(0, 0, 1))
	emitCellOutward(m, pl, 0, 1, 0, 1, a, b, c, d)
	want := []int{0, 2, 1, 0, 3, 2}
	if len(m.Indices) != len(want) {
		t.Fatalf("indices = %v, want %v", m.Indices, want)
	}
	for i := range want {
		if m.Indices[i] != want[i] {
			t.Fatalf("indices = %v, want %v", m.Indices, want)
		}
	}
}

func TestUVLoopHelpersAndPeriodClosure(t *testing.T) {
	t.Parallel()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	outer := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(0, 2, 0)}
	hole := []math.Point3{math.P3(0.5, 0.5, 0), math.P3(1, 0.5, 0), math.P3(1, 1, 0), math.P3(0.5, 1, 0)}
	outerUV, holesUV, ok := ToUVLoops(pl, outer, [][]math.Point3{hole})
	if !ok || len(outerUV) != 4 || len(holesUV) != 1 || len(holesUV[0]) != 4 {
		t.Fatalf("ToUVLoops ok=%v outer=%v holes=%v", ok, outerUV, holesUV)
	}
	if _, ok := Unwrap([]float64{0, stdmath.Pi, 2 * stdmath.Pi}); ok {
		t.Fatal("Unwrap accepted a full-period loop")
	}
	closed := bracketPeriod([]float64{0, stdmath.Pi})
	if len(closed) != 3 || stdmath.Abs(closed[0]) > 1e-12 || stdmath.Abs(closed[2]-2*stdmath.Pi) > 1e-12 {
		t.Fatalf("bracketPeriod = %v; want [0, π, 2π]", closed)
	}
	// A seam sample read back as ~2π−ε must snap to the 0 column, so the period brackets [0, 2π] with the
	// seam shared at both ends — the fix for the dropped seam cell (a one-cell crack against the caps).
	seam := bracketPeriod([]float64{0.5, 1.0, 2*stdmath.Pi - 1e-9})
	if len(seam) != 4 || stdmath.Abs(seam[0]) > 1e-12 || stdmath.Abs(seam[3]-2*stdmath.Pi) > 1e-12 {
		t.Fatalf("bracketPeriod(seam≈2π) = %v; want [0, 0.5, 1, 2π]", seam)
	}
}
