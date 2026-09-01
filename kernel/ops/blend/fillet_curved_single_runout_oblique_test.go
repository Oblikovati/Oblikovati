// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// R3 kernel regression: the per-END oblique rail re-termination (obliqueRetermRails / reterminateRailEnds)
// that greens D4/E3. It asserts (a) an OBLIQUE end moves that rail terminus onto its oblique foot, (b) a
// PERPENDICULAR end leaves the rail bit-identical (the do-no-harm guarantee for E3's pole and every prior
// perpendicular runout), and (c) an off-line/off-circle foot DECLINES rather than snapping.

// obliqueRailBench is a named fixture (not an inline stub) supplying a straight ruling rail, an arc rail on
// a known circle, and feet on/off those loci — the closed-form inputs obliqueRetermRails consumes.
type obliqueRailBench struct {
	tol       float64
	straight  endSeg // ruling rail from x=0 to x=10 along +X
	arc       endSeg // quarter-circle rail on the unit-radius circle centred at origin in the XY plane
	arcCenter math.Point3
}

func newObliqueRailBench(t *testing.T) obliqueRailBench {
	t.Helper()
	center := math.P3(0, 0, 0)
	arc, err := geom.NewArc3d(center, math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("bench arc construction failed: %v", err)
	}
	return obliqueRailBench{
		tol:       1e-9,
		straight:  endSeg{from: math.P3(0, 0, 0), to: math.P3(10, 0, 0)},
		arc:       endSeg{from: arc.PointAt(0), to: arc.PointAt(1), curve: arc, mid: arc.PointAt(0.5), arc: true},
		arcCenter: center,
	}
}

// TestReterminateRailEndsStraight moves each end of a straight ruling onto an on-line foot and checks the
// gated end moved while the ungated end stayed put (byte-identical) — the perpendicular do-no-harm property.
func TestReterminateRailEndsStraight(t *testing.T) {
	t.Parallel()
	b := newObliqueRailBench(t)
	startFoot, endFoot := math.P3(2, 0, 0), math.P3(8, 0, 0)
	got, ok := reterminateRailEnds(b.straight, startFoot, endFoot, true, true, b.tol)
	if !ok || pointFar(got.from, startFoot, b.tol) || pointFar(got.to, endFoot, b.tol) {
		t.Fatalf("both-ends re-termination: ok=%v from=%v to=%v, want %v→%v", ok, got.from, got.to, startFoot, endFoot)
	}
	onlyStart, ok := reterminateRailEnds(b.straight, startFoot, endFoot, true, false, b.tol)
	if !ok || pointFar(onlyStart.from, startFoot, b.tol) || pointFar(onlyStart.to, b.straight.to, b.tol) {
		t.Fatalf("start-only re-termination: from=%v to=%v, want %v→%v (to unchanged)", onlyStart.from, onlyStart.to, startFoot, b.straight.to)
	}
}

// TestReterminateRailEndsPerpendicularNoop asserts an end with NO re-termination requested is returned
// verbatim — the byte-identity guarantee that keeps E3's pole and every prior perpendicular runout intact.
func TestReterminateRailEndsPerpendicularNoop(t *testing.T) {
	t.Parallel()
	b := newObliqueRailBench(t)
	got, ok := reterminateRailEnds(b.straight, math.P3(2, 0, 0), math.P3(8, 0, 0), false, false, b.tol)
	if !ok || got.from != b.straight.from || got.to != b.straight.to {
		t.Fatalf("no-op re-termination changed the rail: %v→%v, want %v→%v", got.from, got.to, b.straight.from, b.straight.to)
	}
}

// TestReterminateRailEndsOffLineDeclines asserts a foot off the ruling line DECLINES (do-no-harm) — the
// shared-edge identity (foot == rail terminus) must hold, never be snapped onto the line.
func TestReterminateRailEndsOffLineDeclines(t *testing.T) {
	t.Parallel()
	b := newObliqueRailBench(t)
	if _, ok := reterminateRailEnds(b.straight, math.P3(2, 5, 0), b.straight.to, true, false, b.tol); ok {
		t.Fatalf("off-line foot (2,5,0) was accepted, want a decline")
	}
}

// TestReterminateRailEndsArc re-terminates the arc rail's start onto an on-circle foot and checks the moved
// end lands on it while staying on the SAME circle (radius preserved) — the torus-arm contact-circle case.
func TestReterminateRailEndsArc(t *testing.T) {
	t.Parallel()
	b := newObliqueRailBench(t)
	foot := math.P3(stdmath.Cos(stdmath.Pi/6), stdmath.Sin(stdmath.Pi/6), 0) // 30° on the unit circle
	got, ok := reterminateRailEnds(b.arc, foot, b.arc.to, true, false, b.tol)
	if !ok || pointFar(got.from, foot, b.tol) {
		t.Fatalf("arc start re-termination: ok=%v from=%v, want %v", ok, got.from, foot)
	}
	if r := float64(b.arcCenter.DistanceTo(got.mid)); stdmath.Abs(r-1) > b.tol {
		t.Fatalf("re-swept arc mid %v is off the unit circle (radius %.9f), want 1", got.mid, r)
	}
}

// TestObliqueRetermRailsGatesOnRegime asserts obliqueRetermRails re-terminates ONLY the oblique end's rail
// terminus and leaves a perpendicular end untouched — the E3 MIXED-arm contract (pole perp, cap oblique).
func TestObliqueRetermRailsGatesOnRegime(t *testing.T) {
	t.Parallel()
	b := newObliqueRailBench(t)
	perpStart := math.P3(0, 0, 0) // run0 perpendicular: foot already IS railA.from → no move
	oblEnd := math.P3(8, 0, 0)    // run1 oblique: railA.to must move from 10 to 8
	run0 := armRunout{regime: runoutPerpendicular, feet: [2]math.Point3{perpStart, perpStart}}
	run1 := armRunout{regime: runoutOblique, feet: [2]math.Point3{oblEnd, oblEnd}}
	rA, _, reason := obliqueRetermRails(b.straight, b.straight, run0, run1, b.tol)
	if reason != "" {
		t.Fatalf("obliqueRetermRails declined: %s", reason)
	}
	if rA.from != b.straight.from || pointFar(rA.to, oblEnd, b.tol) {
		t.Fatalf("mixed-arm re-termination: %v→%v, want %v→%v (perp start fixed, oblique end moved)", rA.from, rA.to, b.straight.from, oblEnd)
	}
}

// pointFar reports whether two points differ by more than tol.
func pointFar(a, b math.Point3, tol float64) bool {
	return float64(a.DistanceTo(b)) > tol
}
