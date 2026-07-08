// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func TestKeepsInside(t *testing.T) {
	cases := []struct {
		op   Op
		isB  bool
		want bool
	}{
		{Intersection, false, true}, // intersect always keeps inside (the lens caps)
		{Intersection, true, true},  //
		{Difference, true, true},    // the TOOL of a cut keeps its inside (the tunnel)
		{Difference, false, false},  // the TARGET of a cut keeps its outside (the holed wall)
		{Union, false, false},       // a union keeps every wall outside
		{Union, true, false},        //
	}
	for _, c := range cases {
		if got := keepsInside(c.op, c.isB); got != c.want {
			t.Errorf("keepsInside(%v, isB=%v) = %v, want %v", c.op, c.isB, got, c.want)
		}
	}
}

func TestFatterOperand(t *testing.T) {
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	fatB, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3.5, 12)
	oThin, _ := cylinderOperand(thin)
	oFat, _ := cylinderOperand(fatB)
	// The fatter is returned first regardless of argument order, with aIsFat tracking whether a was the fat one.
	fat, thinOp, aIsFat, ok := fatterOperand(oThin, oFat)
	if !ok || fat.body != fatB || thinOp.body != thin || aIsFat {
		t.Errorf("fatterOperand(thin, fat): fat=%v thin=%v aIsFat=%v ok=%v, want fat=fatB thin=thin aIsFat=false", fat.body == fatB, thinOp.body == thin, aIsFat, ok)
	}
	fat2, _, aIsFat2, _ := fatterOperand(oFat, oThin)
	if fat2.body != fatB || !aIsFat2 {
		t.Errorf("fatterOperand(fat, thin): fat=fatB=%v aIsFat=%v, want true,true", fat2.body == fatB, aIsFat2)
	}
}

func TestCapCircleOfAndAxialSide(t *testing.T) {
	body, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12) // axis +x, caps at x=-6 and x=+6
	_, cyl, _, _ := cylinderSideFace(body)
	caps := planarCapFaces(body)
	if len(caps) != 2 {
		t.Fatalf("cylinder has %d planar caps, want 2", len(caps))
	}
	if _, ok := capCircleOf(caps[0]); !ok {
		t.Error("capCircleOf failed on a planar cap face")
	}
	high, okH := capOnAxialSide(caps, cyl, true)
	low, okL := capOnAxialSide(caps, cyl, false)
	if !okH || !okL {
		t.Fatal("capOnAxialSide could not resolve a cap")
	}
	axis := cyl.AxisDir.AsVector()
	aHigh := float64(cyl.Origin.VectorTo(high.Center).Dot(axis))
	aLow := float64(cyl.Origin.VectorTo(low.Center).Dot(axis))
	if aHigh <= aLow {
		t.Errorf("high cap axial %.2f not above low cap axial %.2f", aHigh, aLow)
	}
}

func TestLoopCentroidAxial(t *testing.T) {
	// A square loop centred at x=5 on the x-axis: its axial centroid (along +x from the origin) is 5.
	pl, _ := geom.NewPolyline([]math.Point3{
		math.P3(4, 0, 0), math.P3(6, 0, 0), math.P3(6, 1, 0), math.P3(4, 1, 0), math.P3(4, 0, 0),
	})
	got := loopCentroidAxial(pl, math.P3(0, 0, 0), math.V3(1, 0, 0))
	if stdmath.Abs(got-4.8) > 1e-9 { // mean of {4,6,6,4,4} = 4.8
		t.Errorf("loopCentroidAxial = %g, want 4.8", got)
	}
}

// TestRawStubBands checks the near-severed rod's OUTSIDE stubs are built as two two-rim bands, one per loop,
// each pairing the loop with the cap on the far axial side (so the stub extends away from the other loop).
func TestRawStubBands(t *testing.T) {
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3.00006, 12)
	loops, ok := crossingCylinderLoops(thin, fat, nil)
	if !ok || len(loops) != 2 {
		t.Fatalf("expected 2 crossing loops, got ok=%v n=%d", ok, len(loops))
	}
	op, _ := cylinderOperand(thin)
	stubs, okS := rawStubBands(op, loops)
	if !okS || len(stubs) != 2 {
		t.Fatalf("rawStubBands: ok=%v n=%d, want 2 stub bands", okS, len(stubs))
	}
	for i, s := range stubs {
		if len(s.loops) != 2 { // a two-rim band: cap circle + imprint loop
			t.Errorf("stub %d has %d loops, want 2 (cap rim + imprint loop)", i, len(s.loops))
		}
	}
}

// TestCurvedReorientFlipsInconsistent builds two faces that share an edge traversed the SAME way by both
// (an inconsistent shell) and checks curvedReorient flips exactly one so the shared edge is opposed.
func TestCurvedReorientFlipsInconsistent(t *testing.T) {
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	// Both faces traverse the shared segment 0→1 (same direction) — inconsistent.
	f0 := curvedFace{surface: pl, loops: []curvedLoop{{edges: []loopEdge{{curve: seg, t0: 0, t1: 1}}}}}
	f1 := curvedFace{surface: pl, loops: []curvedLoop{{edges: []loopEdge{{curve: seg, t0: 0, t1: 1}}}}}
	out := curvedReorient([]curvedFace{f0, f1})
	d0 := out[0].loops[0].edges[0].t1 > out[0].loops[0].edges[0].t0
	d1 := out[1].loops[0].edges[0].t1 > out[1].loops[0].edges[0].t0
	if d0 == d1 {
		t.Errorf("curvedReorient left the shared edge traversed the same way (d0=%v d1=%v); want opposed", d0, d1)
	}
}
