// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func TestKeepsInside(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	body, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12) // axis +x, caps at x=-6 and x=+6
	_, cyl, _, _ := cylinderSideFace(body)
	caps := planarCapFaces(body)
	if len(caps) != 2 {
		t.Fatalf("cylinder has %d planar caps, want 2", len(caps))
	}
	if _, ok := capCircleOf(caps[0]); !ok {
		t.Error("capCircleOf failed on a planar cap face")
	}
	// A face with no loops, and one whose edge is not a circle, are not cap circles.
	if _, ok := capCircleOf(curvedFace{}); ok {
		t.Error("capCircleOf accepted a loopless face")
	}
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	notCirc := curvedFace{loops: []curvedLoop{{edges: []loopEdge{{curve: seg, t0: 0, t1: 1}}}}}
	if _, ok := capCircleOf(notCirc); ok {
		t.Error("capCircleOf accepted a non-circle edge")
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
	t.Parallel()
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
	t.Parallel()
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

// nearPinchPair builds a thin (radius r, axis x) and a fat (radius r+dr, axis z) crossing cylinder — the
// unequal-radius near-pinch pair the cut/join constructors handle.
func nearPinchPair(t *testing.T, r, dr float64) (thin, fat *topo.Body) {
	t.Helper()
	var err error
	if thin, err = SolidCylinder(math.P3(-2*r, 0, 0), math.V3(1, 0, 0), r, 4*r); err != nil {
		t.Fatalf("SolidCylinder thin: %v", err)
	}
	if fat, err = SolidCylinder(math.P3(0, 0, -2*r), math.V3(0, 0, 1), r+dr, 4*r); err != nil {
		t.Fatalf("SolidCylinder fat: %v", err)
	}
	return thin, fat
}

// TestNearPinchCrossingCutJoinBuild covers the near-pinch constructors at the brep level (the ops watertight
// sweep drives them via dispatch, but per-package coverage needs a direct brep caller): both cut directions
// (fat−thin drill = 4 faces, thin−fat sever = 6 faces), the join (7 faces), and that each builds a valid
// closed manifold solid.
func TestNearPinchCrossingCutJoinBuild(t *testing.T) {
	t.Parallel()
	thin, fat := nearPinchPair(t, 3.0, 6e-5)
	cases := []struct {
		name      string
		build     func() (*topo.Body, bool)
		wantFaces int
	}{
		{"drill fat−thin", func() (*topo.Body, bool) { return nearPinchCrossingCut(fat, thin, nil) }, 4},
		{"sever thin−fat", func() (*topo.Body, bool) { return nearPinchCrossingCut(thin, fat, nil) }, 6},
		{"join", func() (*topo.Body, bool) { return nearPinchCrossingJoin(thin, fat, nil) }, 7},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, ok := c.build()
			if !ok || body == nil {
				t.Fatal("near-pinch constructor declined; want an analytic solid")
			}
			if n := len(body.Faces()); n != c.wantFaces {
				t.Errorf("%s built %d faces, want %d", c.name, n, c.wantFaces)
			}
		})
	}
}

// TestNearPinchGateDeclinesNonNearPinch pins that the gate — and thus the cut/join constructors — decline a
// crossing that is NOT the unequal narrow-neck band, so those pairs fall through to the ordinary pipeline: an
// EQUAL-radius Steinmetz pair (its own constructor owns it) and a WELL-SEPARATED thin-rod crossing.
func TestNearPinchGateDeclinesNonNearPinch(t *testing.T) {
	t.Parallel()
	eqA, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	eqB, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12) // equal radii
	if _, ok := nearPinchCrossingCut(eqA, eqB, nil); ok {
		t.Error("near-pinch cut accepted an equal-radius Steinmetz pair; want decline")
	}
	if _, ok := nearPinchCrossingJoin(eqA, eqB, nil); ok {
		t.Error("near-pinch join accepted an equal-radius Steinmetz pair; want decline")
	}
	thinRod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1, 12)
	fatCyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12) // well separated (r 1 vs 3)
	if _, ok := nearPinchCrossingCut(thinRod, fatCyl, nil); ok {
		t.Error("near-pinch cut accepted a well-separated crossing; want decline")
	}
}

// TestCurvedReorientFlipsInconsistent builds two faces that share an edge traversed the SAME way by both
// (an inconsistent shell) and checks curvedReorient flips exactly one so the shared edge is opposed.
func TestCurvedReorientFlipsInconsistent(t *testing.T) {
	t.Parallel()
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
