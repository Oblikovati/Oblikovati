// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The trim-side split's own unit gates. Every expectation is a CLOSED FORM read off complex/D8's own
// dimensions (the same constants the chain primitive's acceptance uses), so these measure the parts
// against the geometry they must produce rather than against what they happen to produce.

// d8RoundSurface is the radius-24 corner round complex/D8's band stops on, as a geom.Cylinder. Its Ref
// direction is whatever geom.NewCylinder picks, deliberately: every assertion below is stated on a
// parameter SPAN or a ring INDEX, both frame-independent, so the gates cannot pass by agreeing with one
// particular reference frame.
func d8RoundSurface(t *testing.T) geom.Cylinder {
	t.Helper()
	cyl, err := geom.NewCylinder(math.P3(d8CX, d8CY, -90), math.V3(0, 0, -1), d8CR)
	if err != nil {
		t.Fatalf("d8 round cylinder: %v", err)
	}
	return cyl
}

// TestRingParamBoxReadsD8sCornerRoundAsItsOwnQuarter: the round's face is bounded by two full-height
// rulings and two rim arcs, so it IS a parameter box — a quarter turn wide and the plate's own 100 tall.
func TestRingParamBoxReadsD8sCornerRoundAsItsOwnQuarter(t *testing.T) {
	t.Parallel()
	box, ok := ringParamBox(d8RoundSurface(t), d8RoundRing(), retrimChainTol)
	if !ok {
		t.Fatal("ringParamBox declined D8's corner round, which is bounded by rulings and rims only")
	}
	if u := box.uHi - box.uLo; stdmath.Abs(u-stdmath.Pi/2) > 1e-12 {
		t.Errorf("u span %.15g, closed form %.15g (a quarter turn)", u, stdmath.Pi/2)
	}
	if v := box.vHi - box.vLo; stdmath.Abs(v-100) > 1e-9 {
		t.Errorf("v span %.15g, closed form 100 (the plate's height)", v)
	}
}

// TestRingParamBoxDeclinesABoundaryThatIsNotAnIsoCurve: a face whose boundary cuts diagonally across the
// chart is not a box, and the split must say so rather than invent bounds for it.
func TestRingParamBoxDeclinesABoundaryThatIsNotAnIsoCurve(t *testing.T) {
	t.Parallel()
	ring := d8RoundRing()
	ring[1] = endSeg{from: ring[1].from, to: ring[2].to} // a helical chord replacing the u = −π/2 ruling
	ring = ring[:len(ring)-1]
	ring = append(ring, endSeg{from: ring[len(ring)-1].to, to: ring[0].from})
	if _, ok := ringParamBox(d8RoundSurface(t), ring, retrimChainTol); ok {
		t.Error("ringParamBox accepted a ring with a boundary that varies in BOTH parameters")
	}
}

// TestRingIndexOnBoxSideFindsD8sExitRuling: the band's terminal section overshoots to
// (217.39418, 35.85588, 10), which is ON the round's cylinder but 0.2527 rad past its own u = 0 ruling.
// The split must name that ruling — ring index 3 — as the boundary the landing leaves through, because
// the face across it is the flat wall that carries the rest of the trim.
func TestRingIndexOnBoxSideFindsD8sExitRuling(t *testing.T) {
	t.Parallel()
	s, ring := d8RoundSurface(t), d8RoundRing()
	box, ok := ringParamBox(s, ring, retrimChainTol)
	if !ok {
		t.Fatal("ringParamBox declined D8's corner round")
	}
	over := math.P3(d8CX-d8DX, math.Scalar(d8CY-stdmath.Sqrt(d8CR*d8CR-d8DX*d8DX)), 10)
	side := boxSideOfPoint(s, box, over, retrimChainTol)
	if side == sideInside {
		t.Fatalf("the overshoot landing %v reads as inside the round's own box %+v", over, box)
	}
	i, ok := ringIndexOnBoxSide(s, ring, box, side, retrimChainTol)
	if !ok || i != 3 {
		t.Errorf("exit ring index %d (ok %v), want 3 — the u = 0 ruling shared with the flat wall", i, ok)
	}
}

// TestBoxExitSideAcceptsALandingExactlyOnTheBound: a correct trim ENDS on the stop face's boundary, so a
// landing on a bound must read as inside — else the junction itself would be classed as an overrun.
func TestBoxExitSideAcceptsALandingExactlyOnTheBound(t *testing.T) {
	t.Parallel()
	s, ring := d8RoundSurface(t), d8RoundRing()
	box, ok := ringParamBox(s, ring, retrimChainTol)
	if !ok {
		t.Fatal("ringParamBox declined D8's corner round")
	}
	onBound := math.P3(d8CX, d8CY-d8CR, math.Scalar(d8BZ+stdmath.Sqrt(d8BR*d8BR-d8DX*d8DX)))
	if side := boxSideOfPoint(s, box, onBound, retrimChainTol); side != sideInside {
		t.Errorf("the analytic junction %v (u = 0, z = −20+√864) reads as exiting side %d", onBound, side)
	}
}

// TestSingleOffRunTakesOneTailRunAndRefusesEverythingElse pins the "one entry, one exit" precondition:
// only a single contiguous run of off-face stations, crossing ONE bound, touching ONE end of the section.
func TestSingleOffRunTakesOneTailRunAndRefusesEverythingElse(t *testing.T) {
	t.Parallel()
	in, up := sideInside, sideUHi
	for _, tc := range []struct {
		name    string
		sides   []boxSide
		wantOK  bool
		atTail  bool
		wantVal boxSide
	}{
		{"tail run", []boxSide{in, in, in, up, up}, true, true, up},
		{"head run", []boxSide{up, up, in, in, in}, true, false, up},
		{"mid run", []boxSide{in, up, up, in, in}, false, false, in},
		{"two runs", []boxSide{up, in, in, in, up}, false, false, in},
		{"two bounds", []boxSide{in, in, in, sideVHi, up}, false, false, in},
		{"all on", []boxSide{in, in, in, in, in}, false, false, in},
		{"all off", []boxSide{up, up, up, up, up}, false, false, in},
	} {
		side, atTail, ok := singleOffRun(tc.sides)
		if ok != tc.wantOK || (ok && (atTail != tc.atTail || side != tc.wantVal)) {
			t.Errorf("%s: got (%d, %v, %v), want (%d, %v, %v)", tc.name, side, atTail, ok, tc.wantVal, tc.atTail, tc.wantOK)
		}
	}
}

// TestBisectOutwardZeroLandsOnTheAnalyticRoot: the junction solver is a plain bisection, so its accuracy
// is the bracket halved farEndJunctionSteps times. Gated on an exact root, not a tolerance.
func TestBisectOutwardZeroLandsOnTheAnalyticRoot(t *testing.T) {
	t.Parallel()
	got, ok := bisectOutwardZero(func(x float64) (float64, bool) { return x - 1.0/3.0, true }, 0, 1)
	if !ok {
		t.Fatal("bisectOutwardZero declined a bracket that straddles its root")
	}
	if stdmath.Abs(got-1.0/3.0) > 1e-15 {
		t.Errorf("root %.17g, closed form %.17g", got, 1.0/3.0)
	}
}

// TestPieceSegKeepsAnExactArcAndFitsAnythingElse: a stop plane perpendicular to the slide axis lands the
// section arc on itself, translated — an EXACT circle, which must ship as a geom.Arc3d rather than as a
// b-spline fit of it. Anything that is not on one circle must still be carried, as the fit.
func TestPieceSegKeepsAnExactArcAndFitsAnythingElse(t *testing.T) {
	t.Parallel()
	circle, helix := make([]math.Point3, 9), make([]math.Point3, 9)
	for i := range circle {
		a := float64(i) / 8 * stdmath.Pi / 2
		circle[i] = math.P3(math.Scalar(30*stdmath.Cos(a)), 7, math.Scalar(30*stdmath.Sin(a)))
		helix[i] = math.P3(math.Scalar(30*stdmath.Cos(a)), math.Scalar(7+3*a), math.Scalar(30*stdmath.Sin(a)))
	}
	if seg, ok := pieceSeg(circle, 1e-9); !ok || !seg.arc {
		t.Errorf("a quarter circle shipped as arc=%v (ok %v) — an exact section arc must stay analytic", seg.arc, ok)
	}
	if seg, ok := pieceSeg(helix, 1e-9); !ok || seg.arc {
		t.Errorf("a helical station list shipped as arc=%v (ok %v) — it is on no single circle", seg.arc, ok)
	}
}

// ★ TestAReversedChainSegmentTrimsToTheCorrectHalf is the FALSIFICATION of the endpoint-only reversal.
//
// spliceCornerBiteChain reverses a chain to close the kept span, and the reversal used to swap a non-arc
// segment's endpoints while leaving its curve object pointing the ORIGINAL way. A reversed segment was
// then SELF-INCONSISTENT, and both readers of a segment's parameter got it wrong:
//
//   - a trim by parameter (segToParam, the far-end clip's own operation) kept the half adjacent to the
//     ORIGINAL start — the half the caller meant to DROP;
//   - discretizeEdge drew a boundary that leapt to the far end, walked back and leapt again (how
//     simple/M4 N3 N9 came to self-cross), costing complex/D8's mirror round −6.8 % and its band +31.7 %.
//
// This pins the first directly, on D8's own trim curve: take the FIRST half of the reversed segment and
// require its carried curve's own midpoint to be at parameter 0.75 of the original curve (the half next
// to the reversed segment's `from`). The endpoint-only reversal puts it at 0.25 — the wrong half —
// which is the assertion below going red.
func TestAReversedChainSegmentTrimsToTheCorrectHalf(t *testing.T) {
	t.Parallel()
	c := d8TrimCurve{lo: -stdmath.Pi / 2, hi: 0}
	fwd := endSeg{from: c.PointAt(0), to: c.PointAt(1), curve: c, mid: c.PointAt(0.5)}
	rev := reversedEndSeg(fwd)
	if d := rev.curve.PointAt(0).DistanceTo(rev.from); float64(d) > 1e-12 {
		t.Errorf("a reversed segment's curve must start at its own `from`; it starts %.6g away", d)
	}
	half := segToParam(rev, 0.5, segPointAt(rev, 0.5))
	lo, hi := half.curve.Domain()
	got := half.curve.PointAt((lo + hi) / 2)
	if d := got.DistanceTo(c.PointAt(0.75)); float64(d) > 1e-12 {
		t.Errorf("the kept half is centred %.6g from c(0.75); it is %.6g from c(0.25) — the WRONG half",
			d, float64(got.DistanceTo(c.PointAt(0.25))))
	}
}
