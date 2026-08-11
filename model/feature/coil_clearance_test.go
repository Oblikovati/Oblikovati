// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Regression for Oblikovati#2080. A coil that rises less over one revolution than its profile is
// deep runs into the turn below it. Measured on a 1-deep wire: pitch 1.0 was clean, pitch 0.8
// delivered 256 interpenetrating face pairs — and the body was still reported valid and solid,
// with a volume (84.73) all but indistinguishable from a clean coil's (84.78). So neither the
// topology gate nor a volume sanity check could ever have caught this.

// pitchedCoil builds a plain helix over a wire-deep square offset 4 from the axis.
func pitchedCoil(t *testing.T, pitch, wire float64) (*PartFeatures, *PartFeature) {
	t.Helper()
	fs := NewPartFeatures(nil)
	pf := NewCoilFeatures(fs).Add(offsetSquareSketch(4, wire), 0, yAxis(),
		func() float64 { return pitch }, func() float64 { return 3 }, 0, ops.NewBody)
	fs.Recompute()
	return fs, pf
}

// TestCoilRefusesTurnsThatRunIntoEachOther: below the profile's own depth the coil must refuse,
// not deliver a solid whose turns occupy the same space.
func TestCoilRefusesTurnsThatRunIntoEachOther(t *testing.T) {
	for _, pitch := range []float64{0.8, 0.5} {
		_, pf := pitchedCoil(t, pitch, 1)
		if pf.Health().OK() {
			t.Errorf("pitch %g on a 1-deep wire built a coil whose turns must overlap", pitch)
		}
	}
}

// TestCoilAcceptsTurnsThatClear is the other half: the gate must not blunt ordinary coils. Pitch
// equal to the profile depth is the boundary — the turns touch but do not overlap — and must build.
func TestCoilAcceptsTurnsThatClear(t *testing.T) {
	for _, pitch := range []float64{1.0, 1.5, 2.0} {
		fs, pf := pitchedCoil(t, pitch, 1)
		if !pf.Health().OK() {
			t.Fatalf("pitch %g on a 1-deep wire was refused: %+v", pitch, pf.Health())
		}
		if hits := ops.SelfIntersections(fs.Result()[0], ops.DefaultQuality()); len(hits) > 0 {
			t.Errorf("pitch %g was accepted but has %d interpenetrating face pairs", pitch, len(hits))
		}
	}
}

// TestCoilRefusalNamesTheProfileDepth: the message has to say what to change. A bare "coil is
// invalid" leaves the author guessing which of pitch, revolutions or profile is at fault.
func TestCoilRefusalNamesTheProfileDepth(t *testing.T) {
	_, pf := pitchedCoil(t, 0.5, 1)
	msg := pf.Health().Reason
	for _, want := range []string{"passes through itself", "deep along the axis"} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal %q does not mention %q", msg, want)
		}
	}
}

// TestCoilProfileAxialDepthMeasuresAlongTheAxis: the depth is taken along the COIL AXIS, not from
// the sketch's own extents, so a profile that is wider than it is deep is measured the way the
// clearance actually works.
func TestCoilProfileAxialDepthMeasuresAlongTheAxis(t *testing.T) {
	// A section 6 wide across the axis and 2 deep along it (the axis here is Y).
	section := []math.Point3{math.P3(4, 0, 0), math.P3(10, 0, 0), math.P3(10, 2, 0), math.P3(4, 2, 0)}
	if got := coilProfileAxialDepth([][]math.Point3{section}, yAxis()); stdmath.Abs(got-2) > 1e-12 {
		t.Errorf("axial depth = %g, want 2 (the extent along Y, not the 6 across it)", got)
	}
}

// TestTaperedCoilMayRiseLessThanItIsDeep is why the gate is geometric rather than arithmetic on
// the rail. A tapered coil moves every turn to a new radius, so it can rise less over a revolution
// than the profile is deep and still never touch itself. A "rise must beat the profile depth" rule
// would refuse real conical springs.
func TestTaperedCoilMayRiseLessThanItIsDeep(t *testing.T) {
	fs := NewPartFeatures(nil)
	// Taper 0.9 rad moves each turn tan(0.9)*0.8 = 1.007 outward — just past the profile's own
	// 1.0 radial width — while the rise stays 0.8, below the profile's 1.0 depth. Measured: 0.6 rad
	// gives only 0.547 of offset and does overlap, so this margin is the real boundary, not a guess.
	pf := NewCoilFeatures(fs).Add(offsetSquareSketch(4, 1), 0, yAxis(),
		func() float64 { return 0.8 }, func() float64 { return 3 }, 0.9, ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("a tapered coil rising 0.8 per turn on a 1-deep wire was refused: %+v", pf.Health())
	}
	if hits := ops.SelfIntersections(fs.Result()[0], ops.DefaultQuality()); len(hits) > 0 {
		t.Errorf("the tapered coil was accepted but has %d interpenetrating pairs", len(hits))
	}
}

// TestSurfaceCoilIsGatedToo: an open coiled sheet that passes through itself is just as wrong as a
// solid one, and takes the same path through coilTool.
func TestSurfaceCoilIsGatedToo(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	s.Circles().AddByCenterRadius(math.P2(4, 0), 0.5)
	fs := NewPartFeatures(nil)
	pf := NewCoilFeatures(fs).AddDefinition(&CoilDefinition{
		Sketch: s, Axis: yAxis(),
		Pitch: angleConst(0.4), Revolutions: angleConst(3), Operation: ops.Surface,
	})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("a surface coil whose turns overlap (0.4 rise, 1.0 deep) was accepted")
	}
}
