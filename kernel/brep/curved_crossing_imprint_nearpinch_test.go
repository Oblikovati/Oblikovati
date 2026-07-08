// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// squareLoop is a closed unit-edge square loop in the z=0 plane offset by (dx,0,0) — a synthetic imprint
// loop with a known typical chord (1) for the near-pinch gate helpers.
func squareLoop(dx float64) geom.Polyline {
	pl, _ := geom.NewPolyline([]math.Point3{
		math.P3(math.Scalar(dx), 0, 0), math.P3(math.Scalar(dx+1), 0, 0),
		math.P3(math.Scalar(dx+1), 1, 0), math.P3(math.Scalar(dx), 1, 0),
		math.P3(math.Scalar(dx), 0, 0),
	})
	return pl
}

func TestTypicalLoopChord(t *testing.T) {
	got := typicalLoopChord([]geom.Polyline{squareLoop(0), squareLoop(5)})
	if stdmath.Abs(got-1) > 1e-9 { // every edge is a unit segment
		t.Errorf("typicalLoopChord = %g, want 1", got)
	}
	if got := typicalLoopChord(nil); got != 0 {
		t.Errorf("typicalLoopChord(nil) = %g, want 0", got)
	}
}

func TestInterLoopMinDistance(t *testing.T) {
	// loop at x∈[0,1] and loop at x∈[5,6]: nearest vertices are x=1 and x=5, gap 4.
	if got := interLoopMinDistance(squareLoop(0), squareLoop(5)); stdmath.Abs(got-4) > 1e-9 {
		t.Errorf("interLoopMinDistance = %g, want 4", got)
	}
}

func TestLoopGapChordRatio(t *testing.T) {
	// gap 4, chord 1 → ratio 4.
	if got := loopGapChordRatio([]geom.Polyline{squareLoop(0), squareLoop(5)}); stdmath.Abs(got-4) > 1e-9 {
		t.Errorf("loopGapChordRatio(separated) = %g, want 4", got)
	}
	// Not exactly two loops → +Inf (never gated as near-pinch).
	if got := loopGapChordRatio([]geom.Polyline{squareLoop(0)}); !stdmath.IsInf(got, 1) {
		t.Errorf("loopGapChordRatio(one loop) = %g, want +Inf", got)
	}
}

func TestNearPinchLoops(t *testing.T) {
	// gap 4, chord 1 → ratio 4 ≥ κ: well separated, not near-pinch.
	if nearPinchLoops([]geom.Polyline{squareLoop(0), squareLoop(5)}) {
		t.Error("well-separated loops (ratio 4) classified as near-pinch")
	}
	// gap 0.3 (loop at x∈[1.3,2.3] vs [0,1] → nearest x=1 and x=1.3), chord 1 → ratio 0.3 < κ: near-pinch.
	if !nearPinchLoops([]geom.Polyline{squareLoop(0), squareLoop(1.3)}) {
		t.Error("narrow-neck loops (ratio 0.3) not classified as near-pinch")
	}
	// A single loop is never a near-pinch pair (partial-penetration imprints stay on the analytic path).
	if nearPinchLoops([]geom.Polyline{squareLoop(0)}) {
		t.Error("a single loop classified as near-pinch")
	}
}

// TestCrossingCylinderImprintRecoveredBand covers the in-package analytic path (sampleVertices) for the
// #1781-recovered band: a near-equal crossing above the near-pinch gate traces two loops and builds the
// exact three-face intersect, whereas one below the gate declines. Complements the ops-level watertight
// sweep by exercising the brep entry (and the vertex-sampling arrangement branch) directly.
func TestCrossingCylinderImprintRecoveredBand(t *testing.T) {
	const r = 3.0
	recovered, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
	recoveredZ, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r+2e-4, 12) // above the gate
	body, ok := crossingCylinderIntersectGeneral(recovered, recoveredZ, nil)
	if !ok {
		t.Fatal("recovered-band crossing intersect declined; want the analytic three-face solid")
	}
	if n := len(body.Faces()); n != 3 {
		t.Errorf("recovered-band intersect has %d faces, want 3 (rod band + two lens caps)", n)
	}
}
