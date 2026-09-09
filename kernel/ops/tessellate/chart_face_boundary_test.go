// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestAWrappingTraceEndsAWholeTurnFromWhereItStarted is the whole reason the lift carries one extra
// entry: a boundary loop that goes the whole way round a periodic axis is CLOSED in 3D and OPEN in the
// covering space, and the step that says so is the closing one, from the last sample back to the first.
func TestAWrappingTraceEndsAWholeTurnFromWhereItStarted(t *testing.T) {
	t.Parallel()
	rim := make([]float64, 0, 8)
	for i := range 8 {
		rim = append(rim, 2*stdmath.Pi*float64(i)/8)
	}
	got := continuousTrace(rim, true)
	if len(got) != len(rim)+1 {
		t.Fatalf("continuousTrace length %d, want %d", len(got), len(rim)+1)
	}
	if d := got[len(got)-1] - got[0]; stdmath.Abs(d-2*stdmath.Pi) > 1e-9 { // tol:numeric — the fold's own rounding
		t.Errorf("a rim's trace spans %g, want one whole turn", d)
	}
	lens := continuousTrace([]float64{0.1, 0.2, 0.3}, true)
	if d := lens[len(lens)-1] - lens[0]; stdmath.Abs(d) > 1e-9 { // tol:numeric
		t.Errorf("a contractible loop's trace spans %g, want zero", d)
	}
}

// TestABoundedAxisTraceJustCloses: on an axis that does not wrap there is no winding to find, so the
// trace closes on its own first value.
func TestABoundedAxisTraceJustCloses(t *testing.T) {
	t.Parallel()
	got := continuousTrace([]float64{1, 2, 3}, false)
	if len(got) != 4 || got[3] != got[0] {
		t.Errorf("continuousTrace bounded = %v, want the samples closed on the first", got)
	}
}

// TestALoopLiftsOntoTheChartsBranch: a torus rim circle inverts to parameters in [0, 2π), and the lift
// must carry the whole chain by whole periods onto the branch the chart was recorded on.
func TestALoopLiftsOntoTheChartsBranch(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	const vRim = 1.0
	rim := make([]math.Point3, 0, 24)
	for i := range 24 {
		rim = append(rim, tor.PointAt(2*stdmath.Pi*float64(i)/24, vRim))
	}
	r := doublyPeriodicRegion() // u on [π, 3π], v on [0, 2π]
	c, ok := liftLoopOntoChart(tor, r, rim)
	if !ok {
		t.Fatal("liftLoopOntoChart declined a full rim circle")
	}
	if len(c.uv) != len(rim)+1 || len(c.p3) != len(rim)+1 {
		t.Fatalf("chain lengths %d/%d, want %d", len(c.uv), len(c.p3), len(rim)+1)
	}
	// A chain that wraps spans a whole period, so it cannot sit INSIDE a one-period window; what the
	// shift settles is which period it sits on, and the covering's replicas supply the rest.
	if mid := (c.uMin + c.uMax) / 2; !r.inWindow(mid, vRim) {
		t.Errorf("the lifted rim is centred at u=%g, off the chart's branch [%g,%g]", mid, r.uLo, r.uHi)
	}
	if stdmath.Abs(c.uMax-c.uMin-2*stdmath.Pi) > 1e-6 { // tol:numeric — 24 samples of a full turn
		t.Errorf("the lifted rim spans %g in u, want one whole turn", c.uMax-c.uMin)
	}
	if stdmath.Abs(c.vMin-vRim) > 1e-9 || stdmath.Abs(c.vMax-vRim) > 1e-9 { // tol:numeric
		t.Errorf("the lifted rim spans v [%g,%g], want the constant %g", c.vMin, c.vMax, vRim)
	}
	// The gate's own key set, on this chain's own weld grid: one segment per sample, the closing one
	// included, because a wrapping chain's last point is its first continued a period along.
	if got := len(chainSegmentKeys([]chartChain{c}, geom.ResolutionForPoints(c.p3).Weld())); got != len(rim) {
		t.Errorf("the lifted rim keys %d segments, want one per sample (%d)", got, len(rim))
	}
}

// TestLiftDeclinesALoopThatBoundsNothing.
func TestLiftDeclinesALoopThatBoundsNothing(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := liftLoopOntoChart(cyl, chartRegion{uPer: true}, []math.Point3{math.P3(2, 0, 0), math.P3(0, 2, 0)}); ok {
		t.Error("liftLoopOntoChart accepted a two-point loop")
	}
}

// TestSurfaceParamsOfLoopInvertsEveryPoint.
func TestSurfaceParamsOfLoopInvertsEveryPoint(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]float64{{0.3, 1}, {1.1, 2}, {2.5, 3}}
	loop := []math.Point3{cyl.PointAt(0.3, 1), cyl.PointAt(1.1, 2), cyl.PointAt(2.5, 3)}
	us, vs := surfaceParamsOfLoop(cyl, loop)
	for i := range want {
		if stdmath.Abs(us[i]-want[i][0]) > 1e-9 || stdmath.Abs(vs[i]-want[i][1]) > 1e-9 { // tol:numeric
			t.Errorf("params[%d] = (%g,%g), want (%g,%g)", i, us[i], vs[i], want[i][0], want[i][1])
		}
	}
}

// TestMeanChainChordIsTheBoundarysOwnScale — the quantity the interior clearance is measured against.
func TestMeanChainChordIsTheBoundarysOwnScale(t *testing.T) {
	t.Parallel()
	p := []math.Point3{math.P3(0, 0, 0), math.P3(3, 0, 0), math.P3(3, 4, 0)}
	if got := meanChainChord(p); stdmath.Abs(got-3.5) > 1e-12 { // tol:numeric — (3 + 4) / 2
		t.Errorf("meanChainChord = %g, want 3.5", got)
	}
	if got := meanChainChord(p[:1]); got != 0 {
		t.Errorf("meanChainChord of a single point = %g, want 0", got)
	}
}
