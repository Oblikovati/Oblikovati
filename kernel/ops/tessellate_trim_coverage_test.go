// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

func TestEmitCellOutwardFlipsInwardCell(t *testing.T) {
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	m := &Mesh{}
	a := m.addVertex(math.P3(0, 0, 0), math.V3(0, 0, 1))
	b := m.addVertex(math.P3(0, 1, 0), math.V3(0, 0, 1))
	c := m.addVertex(math.P3(1, 1, 0), math.V3(0, 0, 1))
	d := m.addVertex(math.P3(1, 0, 0), math.V3(0, 0, 1))
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
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	outer := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(0, 2, 0)}
	hole := []math.Point3{math.P3(0.5, 0.5, 0), math.P3(1, 0.5, 0), math.P3(1, 1, 0), math.P3(0.5, 1, 0)}
	outerUV, holesUV, ok := toUVLoops(pl, outer, [][]math.Point3{hole})
	if !ok || len(outerUV) != 4 || len(holesUV) != 1 || len(holesUV[0]) != 4 {
		t.Fatalf("toUVLoops ok=%v outer=%v holes=%v", ok, outerUV, holesUV)
	}
	if _, ok := unwrap([]float64{0, stdmath.Pi, 2 * stdmath.Pi}); ok {
		t.Fatal("unwrap accepted a full-period loop")
	}
	closed := closePeriod([]float64{0, stdmath.Pi})
	if len(closed) != 3 || stdmath.Abs(closed[2]-2*stdmath.Pi) > 1e-12 {
		t.Fatalf("closePeriod = %v", closed)
	}
	alreadyClosed := closePeriod([]float64{0, 2 * stdmath.Pi})
	if len(alreadyClosed) != 2 {
		t.Fatalf("closePeriod appended to already closed grid: %v", alreadyClosed)
	}
}
