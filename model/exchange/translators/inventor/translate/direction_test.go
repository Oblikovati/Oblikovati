// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
)

// TestDirectionOfHonoursReversedOnAnyPlane pins that `reversed` — and only `reversed` — decides
// which way an extrude grows.
//
// The extent direction is stated in the SKETCH's frame: buildExtrusionShell grows the prism along
// plane.Normal() scaled by the span, so PositiveDir means "along this sketch's own normal". The
// file's DirectionAxis IS that normal in world coordinates (measured: dot = +1.000 on every extrude
// probed, over all 517 corpus sketch placements), so it adds nothing to the comparison.
//
// The cases below are the ones the previous `sign(dir.z)` rule got wrong. That rule was only ever a
// shortcut valid while every sketch was forced onto XY (pre-ee6ac047); once sketches carry their
// real planes it is BLIND on any sketch whose normal is perpendicular to Z, because dir.z is then 0
// and `reversed` is silently ignored. 210 of the corpus's 517 sketch placements have a normal that
// is not +Z, so this is the common case, not a corner.
func TestDirectionOfHonoursReversedOnAnyPlane(t *testing.T) {
	xDir, zDir := [3]float64{1, 0, 0}, [3]float64{0, 0, 1}
	cases := []struct {
		name string
		ex   ipt.Extrude
		want feature.ExtentDirection
	}{
		{"plain grows along the sketch normal", ipt.Extrude{Dir: zDir, DirOK: true}, feature.PositiveDir},
		{"reversed grows against it", ipt.Extrude{Dir: zDir, DirOK: true, Reversed: true}, feature.NegativeDir},
		// An X-facing sketch: dir.z is 0, so the old rule returned PositiveDir either way and threw
		// `reversed` away. CompressionRollerArmActuatorScrew is built entirely from ±X-facing
		// sketches; its screwdriver-slot cut was placed OUTSIDE the head and removed nothing.
		{"X-facing plain", ipt.Extrude{Dir: xDir, DirOK: true}, feature.PositiveDir},
		{"X-facing reversed still reverses", ipt.Extrude{Dir: xDir, DirOK: true, Reversed: true}, feature.NegativeDir},
		{"midplane straddles regardless", ipt.Extrude{Dir: xDir, DirOK: true, Reversed: true, Midplane: true}, feature.SymmetricDir},
		// The DirectionAxis only ever restates the sketch normal, so a file that omits it still says
		// everything needed via `reversed`.
		{"no direction stated still honours reversed", ipt.Extrude{Reversed: true}, feature.NegativeDir},
		{"no direction, not reversed", ipt.Extrude{}, feature.PositiveDir},
	}
	for _, c := range cases {
		if got := directionOf(c.ex); got != c.want {
			t.Errorf("%s: directionOf = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestExtentOfKeepsTwoSidedExtrudePositive pins that a two-sided extrude's second length IS its
// negative side, so the extent must stay PositiveDir — pairing Distance2 with NegativeDir would
// grow both spans the same way and double one side.
func TestExtentOfKeepsTwoSidedExtrudePositive(t *testing.T) {
	e := extentOf(ipt.Extrude{
		Distance: 3, Distance2: 2, Dir: [3]float64{0, 0, -1}, DirOK: true,
	})
	if e.Direction != feature.PositiveDir {
		t.Errorf("two-sided extent direction = %v, want PositiveDir", e.Direction)
	}
	if e.Distance2 == nil || e.Distance2() != 2 {
		t.Errorf("two-sided extent lost its second distance")
	}
	if e.Distance == nil || e.Distance() != 3 {
		t.Errorf("two-sided extent lost its primary distance")
	}
}
